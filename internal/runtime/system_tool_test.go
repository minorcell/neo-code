package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	agentsession "neo-code/internal/session"
	"neo-code/internal/tools"
)

// TestExecuteSystemToolNilService 验证在 nil *Service 上调用返回错误。
func TestExecuteSystemToolNilService(t *testing.T) {
	t.Parallel()

	var s *Service
	_, err := s.ExecuteSystemTool(context.Background(), SystemToolInput{ToolName: "bash"})
	if err == nil {
		t.Fatal("expected error on nil service, got nil")
	}
	if !strings.Contains(err.Error(), "service is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestExecuteSystemToolEmptyToolName 验证空工具名返回错误。
func TestExecuteSystemToolEmptyToolName(t *testing.T) {
	t.Parallel()

	service := NewWithFactory(
		newRuntimeConfigManager(t),
		&stubToolManager{},
		newMemoryStore(),
		&scriptedProviderFactory{provider: &scriptedProvider{}},
		nil,
	)

	_, err := service.ExecuteSystemTool(context.Background(), SystemToolInput{ToolName: ""})
	if err == nil {
		t.Fatal("expected error for empty tool name, got nil")
	}
	if !strings.Contains(err.Error(), "tool name is empty") {
		t.Fatalf("unexpected error: %v", err)
	}

	// 空白字符串也应返回错误
	_, err = service.ExecuteSystemTool(context.Background(), SystemToolInput{ToolName: "   "})
	if err == nil {
		t.Fatal("expected error for whitespace-only tool name, got nil")
	}
}

// TestExecuteSystemToolCancelledContext 验证已取消的上下文立即返回错误。
func TestExecuteSystemToolCancelledContext(t *testing.T) {
	t.Parallel()

	service := NewWithFactory(
		newRuntimeConfigManager(t),
		&stubToolManager{},
		newMemoryStore(),
		&scriptedProviderFactory{provider: &scriptedProvider{}},
		nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.ExecuteSystemTool(ctx, SystemToolInput{ToolName: "bash"})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

// TestExecuteSystemToolSuccess 验证基本成功执行路径。
func TestExecuteSystemToolSuccess(t *testing.T) {
	t.Parallel()

	tm := &stubToolManager{
		result: tools.ToolResult{Content: "ok"},
	}
	service := NewWithFactory(
		newRuntimeConfigManager(t),
		tm,
		newMemoryStore(),
		&scriptedProviderFactory{provider: &scriptedProvider{}},
		nil,
	)

	result, err := service.ExecuteSystemTool(context.Background(), SystemToolInput{
		ToolName:  "bash",
		Arguments: []byte(`{"command":"echo hello"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("result should not be an error")
	}
	if result.Name != "bash" {
		t.Fatalf("expected tool name 'bash', got %q", result.Name)
	}
	if result.ToolCallID == "" {
		t.Fatal("expected non-empty tool call ID")
	}

	// 验证事件发射
	events := collectRuntimeEvents(service.Events())
	assertEventContains(t, events, EventToolStart)
	assertEventContains(t, events, EventToolResult)
}

// TestExecuteSystemToolWithSession 验证提供 sessionID 时能正确加载会话并执行。
func TestExecuteSystemToolWithSession(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	session, err := store.CreateSession(context.Background(), agentsession.CreateSessionInput{
		Title: "test-session",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	tm := &stubToolManager{
		result: tools.ToolResult{Content: "done"},
	}
	service := NewWithFactory(
		newRuntimeConfigManager(t),
		tm,
		store,
		&scriptedProviderFactory{provider: &scriptedProvider{}},
		nil,
	)

	result, err := service.ExecuteSystemTool(context.Background(), SystemToolInput{
		SessionID: session.ID,
		ToolName:  "bash",
		Arguments: []byte(`{"command":"ls"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("result should not be an error")
	}

	events := collectRuntimeEvents(service.Events())
	assertEventContains(t, events, EventToolStart)
	assertEventContains(t, events, EventToolResult)
}

// TestExecuteSystemToolWithSessionLoadError 验证会话加载失败时返回错误。
func TestExecuteSystemToolWithSessionLoadError(t *testing.T) {
	t.Parallel()

	tm := &stubToolManager{
		result: tools.ToolResult{Content: "should not run"},
	}
	service := NewWithFactory(
		newRuntimeConfigManager(t),
		tm,
		newMemoryStore(), // 空 store，无会话数据
		&scriptedProviderFactory{provider: &scriptedProvider{}},
		nil,
	)

	_, err := service.ExecuteSystemTool(context.Background(), SystemToolInput{
		SessionID: "nonexistent-session",
		ToolName:  "bash",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent session, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

// TestExecuteSystemToolCustomRunID 验证自定义 RunID 被正确使用。
func TestExecuteSystemToolCustomRunID(t *testing.T) {
	t.Parallel()

	var capturedInput tools.ToolCallInput
	tm := &stubToolManager{
		executeFn: func(ctx context.Context, input tools.ToolCallInput) (tools.ToolResult, error) {
			capturedInput = input
			return tools.ToolResult{Content: "ok"}, nil
		},
	}
	service := NewWithFactory(
		newRuntimeConfigManager(t),
		tm,
		newMemoryStore(),
		&scriptedProviderFactory{provider: &scriptedProvider{}},
		nil,
	)

	result, err := service.ExecuteSystemTool(context.Background(), SystemToolInput{
		RunID:    "my-custom-run-id",
		ToolName: "bash",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("result should not be an error")
	}

	// 验证事件中包含自定义 RunID
	events := collectRuntimeEvents(service.Events())
	found := false
	for _, e := range events {
		if e.RunID == "my-custom-run-id" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected event with RunID 'my-custom-run-id' in %d events", len(events))
	}
	_ = capturedInput
}

// TestExecuteSystemToolDefaultWorkdir 验证 workdir 为空时使用配置默认值。
func TestExecuteSystemToolDefaultWorkdir(t *testing.T) {
	t.Parallel()

	cfgManager := newRuntimeConfigManager(t)
	cfg := cfgManager.Get()

	var capturedInput tools.ToolCallInput
	tm := &stubToolManager{
		executeFn: func(ctx context.Context, input tools.ToolCallInput) (tools.ToolResult, error) {
			capturedInput = input
			return tools.ToolResult{Content: "ok"}, nil
		},
	}
	service := NewWithFactory(
		cfgManager,
		tm,
		newMemoryStore(),
		&scriptedProviderFactory{provider: &scriptedProvider{}},
		nil,
	)

	_, err := service.ExecuteSystemTool(context.Background(), SystemToolInput{
		ToolName: "bash",
		Workdir:  "", // 空值，应使用配置默认值
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedInput.Workdir != cfg.Workdir {
		t.Fatalf("expected workdir %q, got %q", cfg.Workdir, capturedInput.Workdir)
	}
}

// TestExecuteSystemToolToolExecutionError 验证工具执行错误时 result.IsError 被设置为 true。
func TestExecuteSystemToolToolExecutionError(t *testing.T) {
	t.Parallel()

	execErr := errors.New("tool execution failed")
	tm := &stubToolManager{
		result: tools.ToolResult{Content: "partial output"},
		err:    execErr,
	}
	service := NewWithFactory(
		newRuntimeConfigManager(t),
		tm,
		newMemoryStore(),
		&scriptedProviderFactory{provider: &scriptedProvider{}},
		nil,
	)

	result, err := service.ExecuteSystemTool(context.Background(), SystemToolInput{
		ToolName: "bash",
	})
	if err == nil {
		t.Fatal("expected error from tool execution, got nil")
	}
	if !errors.Is(err, execErr) {
		t.Fatalf("expected wrapped exec error, got: %v", err)
	}
	if !result.IsError {
		t.Fatal("result.IsError should be true when execution fails")
	}
	if result.Name != "bash" {
		t.Fatalf("expected tool name 'bash', got %q", result.Name)
	}

	events := collectRuntimeEvents(service.Events())
	assertEventContains(t, events, EventToolStart)
	assertEventContains(t, events, EventToolResult)
}

// TestNewSystemToolRunID 验证 run ID 生成格式与空名称回退。
func TestNewSystemToolRunID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		prefix string
	}{
		{
			name:   "normal tool name",
			input:  "Bash",
			prefix: "system-tool-bash-",
		},
		{
			name:   "mixed case normalized",
			input:  "ReadFile",
			prefix: "system-tool-readfile-",
		},
		{
			name:   "whitespace trimmed",
			input:  "  bash  ",
			prefix: "system-tool-bash-",
		},
		{
			name:   "empty name falls back to tool",
			input:  "",
			prefix: "system-tool-tool-",
		},
		{
			name:   "whitespace-only falls back to tool",
			input:  "   ",
			prefix: "system-tool-tool-",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := newSystemToolRunID(tc.input)
			if !strings.HasPrefix(got, tc.prefix) {
				t.Fatalf("expected prefix %q, got %q", tc.prefix, got)
			}
			// 验证前缀之后是数字（时间戳）
			suffix := strings.TrimPrefix(got, tc.prefix)
			if suffix == "" {
				t.Fatal("expected numeric suffix after prefix")
			}
			for _, ch := range suffix {
				if ch < '0' || ch > '9' {
					t.Fatalf("expected numeric suffix, got %q in %q", string(ch), got)
				}
			}
		})
	}
}

// TestNewSystemToolCallID 验证 call ID 生成格式与空名称回退。
func TestNewSystemToolCallID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		prefix string
	}{
		{
			name:   "normal tool name",
			input:  "Bash",
			prefix: "call-bash-",
		},
		{
			name:   "mixed case normalized",
			input:  "ReadFile",
			prefix: "call-readfile-",
		},
		{
			name:   "whitespace trimmed",
			input:  "  bash  ",
			prefix: "call-bash-",
		},
		{
			name:   "empty name falls back to tool",
			input:  "",
			prefix: "call-tool-",
		},
		{
			name:   "whitespace-only falls back to tool",
			input:  "   ",
			prefix: "call-tool-",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := newSystemToolCallID(tc.input)
			if !strings.HasPrefix(got, tc.prefix) {
				t.Fatalf("expected prefix %q, got %q", tc.prefix, got)
			}
			suffix := strings.TrimPrefix(got, tc.prefix)
			if suffix == "" {
				t.Fatal("expected numeric suffix after prefix")
			}
			for _, ch := range suffix {
				if ch < '0' || ch > '9' {
					t.Fatalf("expected numeric suffix, got %q in %q", string(ch), got)
				}
			}
		})
	}
}
