package tui

import (
	"strings"
	"testing"

	agentruntime "neo-code/internal/runtime"
)

func TestNormalizePermissionPromptSelectionWrap(t *testing.T) {
	if got := normalizePermissionPromptSelection(-1); got != len(permissionPromptOptions)-1 {
		t.Fatalf("expected -1 to wrap to last index, got %d", got)
	}
	if got := normalizePermissionPromptSelection(len(permissionPromptOptions)); got != 0 {
		t.Fatalf("expected overflow index to wrap to 0, got %d", got)
	}
}

func TestPermissionPromptOptionAt(t *testing.T) {
	option := permissionPromptOptionAt(-1)
	if option.Decision != agentruntime.PermissionResolutionReject {
		t.Fatalf("expected wrapped option to be reject, got %q", option.Decision)
	}
}

func TestParsePermissionShortcut(t *testing.T) {
	tests := map[string]agentruntime.PermissionResolutionDecision{
		"y":      agentruntime.PermissionResolutionAllowOnce,
		"once":   agentruntime.PermissionResolutionAllowOnce,
		"a":      agentruntime.PermissionResolutionAllowSession,
		"always": agentruntime.PermissionResolutionAllowSession,
		"n":      agentruntime.PermissionResolutionReject,
		"deny":   agentruntime.PermissionResolutionReject,
	}
	for input, want := range tests {
		got, ok := parsePermissionShortcut(input)
		if !ok || got != want {
			t.Fatalf("parsePermissionShortcut(%q) = (%q,%v), want (%q,true)", input, got, ok, want)
		}
	}
	if _, ok := parsePermissionShortcut("unknown"); ok {
		t.Fatalf("expected unknown shortcut to fail")
	}
}

func TestFormatPermissionPromptLines(t *testing.T) {
	lines := formatPermissionPromptLines(permissionPromptState{
		Request: agentruntime.PermissionRequestPayload{
			ToolName:  "bash",
			Operation: "exec",
			Target:    "git status",
		},
		Selected:   1,
		Submitting: true,
	})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "权限审批") {
		t.Fatalf("expected prompt header, got %q", joined)
	}
	if !strings.Contains(joined, "> Allow session") {
		t.Fatalf("expected selected option marker, got %q", joined)
	}
	if !strings.Contains(joined, "正在提交审批结果") {
		t.Fatalf("expected submitting hint, got %q", joined)
	}
}

func TestRenderPermissionPrompt(t *testing.T) {
	app := App{
		appRuntimeState: appRuntimeState{
			pendingPermission: &permissionPromptState{
				Request: agentruntime.PermissionRequestPayload{
					ToolName: "bash",
					Target:   "git status",
				},
				Selected: 0,
			},
		},
	}
	rendered := app.renderPermissionPrompt()
	if !strings.Contains(rendered, "权限审批") {
		t.Fatalf("expected rendered permission prompt, got %q", rendered)
	}
}

func TestParsePermissionPayloadHelpers(t *testing.T) {
	req := agentruntime.PermissionRequestPayload{RequestID: "perm-1"}
	if got, ok := parsePermissionRequestPayload(req); !ok || got.RequestID != "perm-1" {
		t.Fatalf("unexpected parsePermissionRequestPayload result: %+v ok=%v", got, ok)
	}
	if _, ok := parsePermissionRequestPayload((*agentruntime.PermissionRequestPayload)(nil)); ok {
		t.Fatalf("expected nil request pointer to fail parsing")
	}

	resolved := agentruntime.PermissionResolvedPayload{RequestID: "perm-2"}
	if got, ok := parsePermissionResolvedPayload(resolved); !ok || got.RequestID != "perm-2" {
		t.Fatalf("unexpected parsePermissionResolvedPayload result: %+v ok=%v", got, ok)
	}
	if _, ok := parsePermissionResolvedPayload((*agentruntime.PermissionResolvedPayload)(nil)); ok {
		t.Fatalf("expected nil resolved pointer to fail parsing")
	}
}
