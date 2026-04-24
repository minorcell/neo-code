package infra

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var runtimeGOOSForOpenResource = runtime.GOOS
var execCommandForOpenResource = exec.Command

// OpenExternalResource 在本机默认应用中打开 URL 或本地文件。
func OpenExternalResource(target string) error {
	normalizedTarget, err := normalizeOpenResourceTarget(target)
	if err != nil {
		return err
	}

	commandName, commandArgs, err := openResourceCommand(runtimeGOOSForOpenResource, normalizedTarget)
	if err != nil {
		return err
	}
	command := execCommandForOpenResource(commandName, commandArgs...)
	if runErr := command.Run(); runErr != nil {
		return fmt.Errorf("open resource %q: %w", normalizedTarget, runErr)
	}
	return nil
}

// normalizeOpenResourceTarget 将输入归一化为可打开的 URL 或存在的绝对文件路径。
func normalizeOpenResourceTarget(target string) (string, error) {
	trimmedTarget := strings.TrimSpace(target)
	if trimmedTarget == "" {
		return "", fmt.Errorf("open resource: target is empty")
	}

	if parsed, parseErr := url.Parse(trimmedTarget); parseErr == nil && parsed != nil {
		scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
		if scheme == "http" || scheme == "https" || scheme == "file" {
			return trimmedTarget, nil
		}
	}

	absolutePath := trimmedTarget
	if !filepath.IsAbs(absolutePath) {
		resolvedPath, resolveErr := filepath.Abs(absolutePath)
		if resolveErr != nil {
			return "", fmt.Errorf("open resource: resolve absolute path: %w", resolveErr)
		}
		absolutePath = resolvedPath
	}
	fileInfo, statErr := os.Stat(absolutePath)
	if statErr != nil {
		return "", fmt.Errorf("open resource: stat %q: %w", absolutePath, statErr)
	}
	if fileInfo.IsDir() {
		return "", fmt.Errorf("open resource: %q is a directory", absolutePath)
	}
	return absolutePath, nil
}

// openResourceCommand 根据平台构造打开目标资源的命令。
func openResourceCommand(goos string, target string) (string, []string, error) {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "windows":
		return "cmd", []string{"/c", "start", "", target}, nil
	case "darwin":
		return "open", []string{target}, nil
	default:
		return "xdg-open", []string{target}, nil
	}
}
