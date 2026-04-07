package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"neo-code/internal/config"
	"neo-code/internal/tools"
	"neo-code/internal/tools/mcp"
)

func TestNewProgram(t *testing.T) {
	disableBuiltinProviderAPIKeys(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	program, err := NewProgram(context.Background())
	if err != nil {
		t.Fatalf("NewProgram() error = %v", err)
	}
	if program == nil {
		t.Fatalf("expected tea program")
	}

	configPath := filepath.Join(home, ".neocode", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created at %q: %v", configPath, err)
	}
}

func TestNewProgramNormalizesInvalidCurrentModelOnStartup(t *testing.T) {
	disableBuiltinProviderAPIKeys(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	configDir := filepath.Join(home, ".neocode")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	raw := []byte("selected_provider: openai\ncurrent_model: unsupported-current\nshell: powershell\n")
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	program, err := NewProgram(context.Background())
	if err != nil {
		t.Fatalf("NewProgram() error = %v", err)
	}
	if program == nil {
		t.Fatalf("expected tea program")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "current_model: "+config.OpenAIDefaultModel) {
		t.Fatalf("expected startup normalization to rewrite current_model, got:\n%s", string(data))
	}
}

func TestBuildToolRegistryUsesWebFetchConfig(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("1234567890"))
	}))
	defer server.Close()

	cfg := config.Default().Clone()
	cfg.Workdir = t.TempDir()
	cfg.Tools.WebFetch.MaxResponseBytes = 4

	registry, err := buildToolRegistry(cfg)
	if err != nil {
		t.Fatalf("buildToolRegistry() error = %v", err)
	}
	tool, err := registry.Get("webfetch")
	if err != nil {
		t.Fatalf("registry.Get(webfetch) error = %v", err)
	}

	args, err := json.Marshal(map[string]string{"url": server.URL})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, execErr := tool.Execute(context.Background(), tools.ToolCallInput{
		Name:      "webfetch",
		Arguments: args,
	})
	if execErr != nil {
		t.Fatalf("webfetch execute error = %v", execErr)
	}
	if truncated, ok := result.Metadata["truncated"].(bool); !ok || !truncated {
		t.Fatalf("expected truncated metadata, got %+v", result.Metadata)
	}
	if result.Content == "" {
		t.Fatalf("expected formatted webfetch content")
	}
}

func TestBuildMCPRegistryFromConfig(t *testing.T) {
	t.Parallel()

	stubClient := &stubMCPServerClient{
		tools: []mcp.ToolDescriptor{
			{Name: "search", Description: "search docs", InputSchema: map[string]any{"type": "object"}},
		},
	}

	cfg := config.Default().Clone()
	cfg.Workdir = t.TempDir()
	cfg.Tools.MCP.Servers = []config.MCPServerConfig{
		{
			ID:      "docs",
			Enabled: true,
			Source:  "stdio",
			Stdio: config.MCPStdioConfig{
				Command: "mock",
			},
		},
	}

	originalRegister := registerMCPStdioServer
	t.Cleanup(func() { registerMCPStdioServer = originalRegister })
	registerMCPStdioServer = func(registry *mcp.Registry, cfg config.Config, server config.MCPServerConfig) error {
		if err := registry.RegisterServer(server.ID, "stdio", server.Version, stubClient); err != nil {
			return err
		}
		return registry.RefreshServerTools(context.Background(), server.ID)
	}

	registry, err := buildMCPRegistry(cfg)
	if err != nil {
		t.Fatalf("buildMCPRegistry() error = %v", err)
	}
	if registry == nil {
		t.Fatalf("expected non-nil mcp registry")
	}
	snapshots := registry.Snapshot()
	if len(snapshots) != 1 || snapshots[0].ServerID != "docs" {
		t.Fatalf("unexpected snapshots: %+v", snapshots)
	}
}

func TestBuildToolRegistryIncludesMCPFromConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Default().Clone()
	cfg.Workdir = t.TempDir()
	cfg.Tools.MCP.Servers = []config.MCPServerConfig{
		{
			ID:      "docs",
			Enabled: true,
			Source:  "stdio",
			Stdio: config.MCPStdioConfig{
				Command: "mock",
			},
		},
	}

	originalRegister := registerMCPStdioServer
	t.Cleanup(func() { registerMCPStdioServer = originalRegister })
	registerMCPStdioServer = func(registry *mcp.Registry, cfg config.Config, server config.MCPServerConfig) error {
		client := &stubMCPServerClient{
			tools: []mcp.ToolDescriptor{
				{Name: "search", Description: "search docs", InputSchema: map[string]any{"type": "object"}},
			},
		}
		if err := registry.RegisterServer(server.ID, "stdio", server.Version, client); err != nil {
			return err
		}
		return registry.RefreshServerTools(context.Background(), server.ID)
	}

	registry, err := buildToolRegistry(cfg)
	if err != nil {
		t.Fatalf("buildToolRegistry() error = %v", err)
	}
	specs, err := registry.ListAvailableSpecs(context.Background(), tools.SpecListInput{})
	if err != nil {
		t.Fatalf("ListAvailableSpecs() error = %v", err)
	}
	found := false
	for _, spec := range specs {
		if spec.Name == "mcp.docs.search" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected mcp.docs.search in specs, got %+v", specs)
	}
}

func TestResolveMCPServerEnvAndWorkdir(t *testing.T) {
	t.Setenv("MCP_TOKEN", "secret")
	env, err := resolveMCPServerEnv(config.MCPServerConfig{
		Env: []config.MCPEnvVarConfig{
			{Name: "TOKEN", ValueEnv: "MCP_TOKEN"},
			{Name: "MODE", Value: "test"},
		},
	})
	if err != nil {
		t.Fatalf("resolveMCPServerEnv() error = %v", err)
	}
	joined := strings.Join(env, ",")
	if !strings.Contains(joined, "TOKEN=secret") || !strings.Contains(joined, "MODE=test") {
		t.Fatalf("unexpected env result: %+v", env)
	}

	base := t.TempDir()
	relative := resolveMCPServerWorkdir(base, "tools/mcp")
	if !strings.HasSuffix(filepath.ToSlash(relative), "tools/mcp") {
		t.Fatalf("unexpected relative workdir: %q", relative)
	}
	absoluteTarget := filepath.Join(t.TempDir(), "absolute")
	absolute := resolveMCPServerWorkdir(base, absoluteTarget)
	if absolute != filepath.Clean(absoluteTarget) {
		t.Fatalf("unexpected absolute workdir: %q", absolute)
	}
}

func TestInitialMCPRefreshTimeoutAndDurationConversion(t *testing.T) {
	t.Parallel()

	cfg := config.Default().Clone()
	cfg.ToolTimeoutSec = 1
	timeout := initialMCPRefreshTimeout(cfg)
	if timeout < 5*time.Second {
		t.Fatalf("expected minimum timeout >= 5s, got %v", timeout)
	}
	if durationFromSeconds(0) != 0 {
		t.Fatalf("expected zero duration for non-positive input")
	}
	if durationFromSeconds(2) != 2*time.Second {
		t.Fatalf("expected 2s duration")
	}
}

func TestBuildToolManagerWrapsRegistry(t *testing.T) {
	t.Parallel()

	registry := tools.NewRegistry()
	registry.Register(stubToolForBootstrap{name: "bash", content: "ok"})
	workdir := t.TempDir()
	manager, err := buildToolManager(registry)
	if err != nil {
		t.Fatalf("buildToolManager() error = %v", err)
	}
	if manager == nil {
		t.Fatalf("expected tool manager")
	}

	specs, err := manager.ListAvailableSpecs(context.Background(), tools.SpecListInput{})
	if err != nil {
		t.Fatalf("ListAvailableSpecs() error = %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %+v", specs)
	}

	_, execErr := manager.Execute(context.Background(), tools.ToolCallInput{
		Name:      "bash",
		Arguments: []byte(`{"command":"echo hi"}`),
		Workdir:   workdir,
	})
	if execErr == nil {
		t.Fatalf("expected bash to require approval by default policy")
	}

	_, execErr = manager.Execute(context.Background(), tools.ToolCallInput{
		Name:      "bash",
		Arguments: []byte(`{"command":"echo hi","workdir":"../outside"}`),
		Workdir:   workdir,
	})
	if execErr == nil {
		t.Fatalf("expected sandbox rejection for outside workdir")
	}
}

func TestBuildToolManagerAllowsWebfetchWhitelist(t *testing.T) {
	t.Parallel()

	registry := tools.NewRegistry()
	registry.Register(stubToolForBootstrap{name: "webfetch", content: "ok"})
	manager, err := buildToolManager(registry)
	if err != nil {
		t.Fatalf("buildToolManager() error = %v", err)
	}

	result, execErr := manager.Execute(context.Background(), tools.ToolCallInput{
		Name:      "webfetch",
		Arguments: []byte(`{"url":"https://github.com/1024XEngineer/neo-code"}`),
		Workdir:   t.TempDir(),
	})
	if execErr != nil {
		t.Fatalf("expected whitelist webfetch allow, got %v", execErr)
	}
	if result.Content != "ok" {
		t.Fatalf("expected ok result, got %+v", result)
	}
}

func TestEnsureConsoleUTF8SetsOutputThenInput(t *testing.T) {
	originalOutput := setConsoleOutputCodePage
	originalInput := setConsoleInputCodePage
	t.Cleanup(func() {
		setConsoleOutputCodePage = originalOutput
		setConsoleInputCodePage = originalInput
	})

	calls := make([]string, 0, 2)
	setConsoleOutputCodePage = func(codePage uint32) error {
		if codePage != utf8CodePage {
			t.Fatalf("expected utf8 code page %d, got %d", utf8CodePage, codePage)
		}
		calls = append(calls, "output")
		return nil
	}
	setConsoleInputCodePage = func(codePage uint32) error {
		if codePage != utf8CodePage {
			t.Fatalf("expected utf8 code page %d, got %d", utf8CodePage, codePage)
		}
		calls = append(calls, "input")
		return nil
	}

	ensureConsoleUTF8()

	if len(calls) != 2 || calls[0] != "output" || calls[1] != "input" {
		t.Fatalf("expected output->input order, got %+v", calls)
	}
}

func TestEnsureConsoleUTF8SkipsInputWhenOutputFails(t *testing.T) {
	originalOutput := setConsoleOutputCodePage
	originalInput := setConsoleInputCodePage
	t.Cleanup(func() {
		setConsoleOutputCodePage = originalOutput
		setConsoleInputCodePage = originalInput
	})

	outputErr := errors.New("output failed")
	setConsoleOutputCodePage = func(codePage uint32) error {
		return outputErr
	}
	inputCalled := false
	setConsoleInputCodePage = func(codePage uint32) error {
		inputCalled = true
		return nil
	}

	ensureConsoleUTF8()

	if inputCalled {
		t.Fatalf("expected input code page setup to be skipped when output setup fails")
	}
}

type stubToolForBootstrap struct {
	name    string
	content string
}

func (s stubToolForBootstrap) Name() string           { return s.name }
func (s stubToolForBootstrap) Description() string    { return "stub" }
func (s stubToolForBootstrap) Schema() map[string]any { return map[string]any{"type": "object"} }
func (s stubToolForBootstrap) MicroCompactPolicy() tools.MicroCompactPolicy {
	return tools.MicroCompactPolicyCompact
}
func (s stubToolForBootstrap) Execute(ctx context.Context, call tools.ToolCallInput) (tools.ToolResult, error) {
	return tools.ToolResult{Name: s.name, Content: s.content}, nil
}

func disableBuiltinProviderAPIKeys(t *testing.T) {
	t.Helper()
	t.Setenv(config.OpenAIDefaultAPIKeyEnv, "")
	t.Setenv(config.GeminiDefaultAPIKeyEnv, "")
	t.Setenv(config.OpenLLDefaultAPIKeyEnv, "")
	t.Setenv(config.QiniuDefaultAPIKeyEnv, "")
}

type stubMCPServerClient struct {
	tools []mcp.ToolDescriptor
}

func (s *stubMCPServerClient) ListTools(ctx context.Context) ([]mcp.ToolDescriptor, error) {
	return append([]mcp.ToolDescriptor(nil), s.tools...), nil
}

func (s *stubMCPServerClient) CallTool(ctx context.Context, toolName string, arguments []byte) (mcp.CallResult, error) {
	return mcp.CallResult{Content: "ok"}, nil
}

func (s *stubMCPServerClient) HealthCheck(ctx context.Context) error {
	return nil
}
