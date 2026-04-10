package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	agentruntime "neo-code/internal/runtime"
)

// permissionPromptOption 表示权限审批面板中的一个可选项。
type permissionPromptOption struct {
	Label    string
	Hint     string
	Decision agentruntime.PermissionResolutionDecision
}

var permissionPromptOptions = []permissionPromptOption{
	{
		Label:    "Allow once",
		Hint:     "仅本次放行",
		Decision: agentruntime.PermissionResolutionAllowOnce,
	},
	{
		Label:    "Allow session",
		Hint:     "本会话同类请求持续放行",
		Decision: agentruntime.PermissionResolutionAllowSession,
	},
	{
		Label:    "Reject",
		Hint:     "拒绝本次请求（可记忆拒绝）",
		Decision: agentruntime.PermissionResolutionReject,
	},
}

// permissionPromptState 保存当前待审批请求与选项状态。
type permissionPromptState struct {
	Request    agentruntime.PermissionRequestPayload
	Selected   int
	Submitting bool
}

// normalizePermissionPromptSelection 保证选项下标始终落在有效范围。
func normalizePermissionPromptSelection(selected int) int {
	if len(permissionPromptOptions) == 0 {
		return 0
	}
	if selected < 0 {
		return len(permissionPromptOptions) - 1
	}
	if selected >= len(permissionPromptOptions) {
		return 0
	}
	return selected
}

// permissionPromptOptionAt 返回指定下标对应的审批选项。
func permissionPromptOptionAt(selected int) permissionPromptOption {
	index := normalizePermissionPromptSelection(selected)
	return permissionPromptOptions[index]
}

// parsePermissionShortcut 将快捷输入映射为审批决策。
func parsePermissionShortcut(input string) (agentruntime.PermissionResolutionDecision, bool) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes", "once", "allow_once":
		return agentruntime.PermissionResolutionAllowOnce, true
	case "a", "always", "allow_session":
		return agentruntime.PermissionResolutionAllowSession, true
	case "n", "no", "reject", "deny":
		return agentruntime.PermissionResolutionReject, true
	default:
		return "", false
	}
}

// formatPermissionPromptLines 构造权限审批面板展示文本。
func formatPermissionPromptLines(state permissionPromptState) []string {
	lines := []string{
		fmt.Sprintf("权限审批：%s (%s)", fallbackText(state.Request.ToolName, "unknown_tool"), fallbackText(state.Request.Operation, "unknown")),
		fmt.Sprintf("目标：%s", fallbackText(state.Request.Target, "(empty)")),
		"使用 ↑/↓ 选择，Enter 确认（快捷键：y=once, a=session, n=reject）",
	}

	for index, item := range permissionPromptOptions {
		prefix := "  "
		if normalizePermissionPromptSelection(state.Selected) == index {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s  - %s", prefix, item.Label, item.Hint))
	}

	if state.Submitting {
		lines = append(lines, "正在提交审批结果...")
	}
	return lines
}

// fallbackText 返回去空格后的值；为空时返回默认文案。
func fallbackText(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// renderPermissionPrompt 渲染审批输入框内容，替代普通输入框文本编辑状态。
func (a App) renderPermissionPrompt() string {
	if a.pendingPermission == nil {
		return a.input.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, formatPermissionPromptLines(*a.pendingPermission)...)
}
