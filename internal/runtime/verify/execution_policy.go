package verify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"neo-code/internal/config"
)

var (
	// ErrVerificationExecutionDenied 表示 verifier 命令被执行策略拒绝。
	ErrVerificationExecutionDenied = errors.New("verification execution denied")
	// ErrVerificationExecutionError 表示 verifier 命令执行过程中发生系统错误。
	ErrVerificationExecutionError = errors.New("verification execution error")
)

var readonlyGitSubcommands = map[string]struct{}{
	"diff":      {},
	"status":    {},
	"show":      {},
	"log":       {},
	"rev-parse": {},
	"ls-files":  {},
}

// CommandExecutionRequest 描述一次 verifier 命令执行请求。
type CommandExecutionRequest struct {
	Command       string
	Workdir       string
	TimeoutSec    int
	OutputCapByte int
	Policy        config.VerificationExecutionPolicyConfig
}

// CommandExecutionResult 描述 verifier 命令执行结果。
type CommandExecutionResult struct {
	ExitCode    int
	Stdout      string
	Stderr      string
	TimedOut    bool
	Truncated   bool
	DurationMS  int64
	CommandName string
}

// CommandExecutor 约束 verifier 命令执行能力，便于测试替换。
type CommandExecutor interface {
	Execute(ctx context.Context, req CommandExecutionRequest) (CommandExecutionResult, error)
}

// PolicyCommandExecutor 在 runtime 进程内执行 non-interactive verifier 命令。
type PolicyCommandExecutor struct{}

// Execute 在白名单策略下执行 verifier 命令并返回结构化结果。
func (PolicyCommandExecutor) Execute(ctx context.Context, req CommandExecutionRequest) (CommandExecutionResult, error) {
	normalizedCommand := strings.TrimSpace(req.Command)
	commandName := commandHead(normalizedCommand)
	if normalizedCommand == "" || commandName == "" {
		return CommandExecutionResult{}, fmt.Errorf("%w: empty command", ErrVerificationExecutionDenied)
	}

	allowed, reason := isCommandAllowed(commandName, normalizedCommand, req.Policy)
	if !allowed {
		return CommandExecutionResult{}, fmt.Errorf("%w: %s", ErrVerificationExecutionDenied, reason)
	}

	timeoutSec := req.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = req.Policy.DefaultTimeout
	}
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	outputCap := req.OutputCapByte
	if outputCap <= 0 {
		outputCap = req.Policy.DefaultOutputCap
	}
	if outputCap <= 0 {
		outputCap = 128 * 1024
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := shellCommand(runCtx, normalizedCommand)
	if workdir := strings.TrimSpace(req.Workdir); workdir != "" {
		cmd.Dir = workdir
	}
	cmd.Env = append(os.Environ(),
		"CI=1",
		"GIT_TERMINAL_PROMPT=0",
	)

	stdout := newCappedBuffer(outputCap)
	stderr := newCappedBuffer(outputCap)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)
	result := CommandExecutionResult{
		ExitCode:    0,
		Stdout:      stdout.String(),
		Stderr:      stderr.String(),
		Truncated:   stdout.Truncated() || stderr.Truncated(),
		DurationMS:  duration.Milliseconds(),
		CommandName: commandName,
	}
	if runErr == nil {
		return result, nil
	}

	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
		return result, fmt.Errorf("%w: command timeout", ErrVerificationExecutionError)
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	return result, fmt.Errorf("%w: %v", ErrVerificationExecutionError, runErr)
}

// isCommandAllowed 判断命令是否符合 verification non-interactive 白名单策略。
func isCommandAllowed(commandName string, raw string, policy config.VerificationExecutionPolicyConfig) (bool, string) {
	denied := make(map[string]struct{}, len(policy.DeniedCommands))
	for _, item := range policy.DeniedCommands {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" {
			continue
		}
		denied[normalized] = struct{}{}
	}
	if _, blocked := denied[commandName]; blocked {
		return false, fmt.Sprintf("command %q is denied by verification policy", commandName)
	}

	allowed := make(map[string]struct{}, len(policy.AllowedCommands))
	for _, item := range policy.AllowedCommands {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}
	if len(allowed) > 0 {
		if _, ok := allowed[commandName]; !ok {
			return false, fmt.Sprintf("command %q is not in allowed_commands", commandName)
		}
	}

	if commandName == "git" {
		sub := gitSubcommand(raw)
		if sub == "" {
			return false, "git subcommand is required"
		}
		if _, ok := readonlyGitSubcommands[sub]; !ok {
			return false, fmt.Sprintf("git subcommand %q is not read-only", sub)
		}
	}
	return true, ""
}

// shellCommand 按平台构建执行命令，统一以非交互 shell 运行 verifier 指令。
func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "powershell", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command)
	}
	return exec.CommandContext(ctx, "sh", "-lc", command)
}

// commandHead 返回命令首个 token（小写）。
func commandHead(command string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(fields[0]))
}

// gitSubcommand 提取 git 命令的二级子命令（小写）。
func gitSubcommand(command string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) < 2 {
		return ""
	}
	if strings.ToLower(strings.TrimSpace(fields[0])) != "git" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(fields[1]))
}

type cappedBuffer struct {
	limit     int
	buffer    bytes.Buffer
	truncated bool
}

// newCappedBuffer 创建带大小上限的输出缓冲区，避免 verifier 命令输出无限膨胀。
func newCappedBuffer(limit int) *cappedBuffer {
	if limit <= 0 {
		limit = 128 * 1024
	}
	return &cappedBuffer{limit: limit}
}

// Write 实现 io.Writer，仅保留上限范围内的输出。
func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b == nil {
		return len(p), nil
	}
	if b.buffer.Len() >= b.limit {
		b.truncated = true
		return len(p), nil
	}
	remain := b.limit - b.buffer.Len()
	if len(p) > remain {
		b.truncated = true
		_, _ = b.buffer.Write(p[:remain])
		return len(p), nil
	}
	_, _ = b.buffer.Write(p)
	return len(p), nil
}

// String 返回当前缓冲区文本。
func (b *cappedBuffer) String() string {
	if b == nil {
		return ""
	}
	return b.buffer.String()
}

// Truncated 返回输出是否发生截断。
func (b *cappedBuffer) Truncated() bool {
	if b == nil {
		return false
	}
	return b.truncated
}
