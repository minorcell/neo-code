package context

import "strings"

type promptSection struct {
	id      string
	title   string
	content string
}

func defaultSystemPromptSections() []promptSection {
	return []promptSection{
		{
			id:    "agent-identity",
			title: "Agent Identity",
			content: strings.TrimSpace(`
You are NeoCode, a local coding agent focused on completing the current task end-to-end.
Preserve the main loop of user input, agent reasoning, tool execution, result observation, and UI feedback.`),
		},
		{
			id:    "tool-usage",
			title: "Tool Usage",
			content: strings.TrimSpace(`
- Use tools when they reduce uncertainty or are required to complete the task safely.
- Inspect tool failures, explain the relevant error, and continue with the safest useful next step.
- Do not claim work is done unless the needed files, commands, or verification actually succeeded.`),
		},
		{
			id:    "workspace-safety",
			title: "Workspace Safety",
			content: strings.TrimSpace(`
- Stay within the current workspace unless the user clearly asks for something else.
- Avoid destructive actions such as deleting files, rewriting unrelated work, or changing history unless explicitly requested.
- Respect project rules and local constraints before making changes.`),
		},
		{
			id:    "code-changes",
			title: "Code Changes",
			content: strings.TrimSpace(`
- Prefer minimal, testable changes that keep module boundaries clear.
- Follow the existing architecture and keep provider, runtime, tools, config, and TUI responsibilities separated.
- When behavior changes, update the relevant tests or documentation needed to keep the implementation verifiable.`),
		},
		{
			id:    "failure-recovery",
			title: "Failure Recovery",
			content: strings.TrimSpace(`
- If blocked, identify the concrete blocker and try the next reasonable path before giving up.
- Surface risky assumptions, partial progress, or missing verification instead of hiding them.
- When constraints prevent completion, return the best safe result and explain what remains.`),
		},
		{
			id:    "response-style",
			title: "Response Style",
			content: strings.TrimSpace(`
- Be concise, accurate, and collaborative.
- Keep updates focused on useful progress, decisions, and verification.
- Base claims on the current workspace state instead of generic advice.`),
		},
	}
}

func composeSystemPrompt(sections ...promptSection) string {
	rendered := make([]string, 0, len(sections))
	for _, section := range sections {
		part := renderPromptSection(section)
		if part == "" {
			continue
		}
		rendered = append(rendered, part)
	}
	return strings.Join(rendered, "\n\n")
}

func renderPromptSection(section promptSection) string {
	title := strings.TrimSpace(section.title)
	content := strings.TrimSpace(section.content)

	switch {
	case title == "" && content == "":
		return ""
	case title == "":
		return content
	case content == "":
		return ""
	default:
		return "## " + title + "\n\n" + content
	}
}
