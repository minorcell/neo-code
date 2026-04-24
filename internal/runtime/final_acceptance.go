package runtime

import (
	"context"
	"strings"

	providertypes "neo-code/internal/provider/types"
	"neo-code/internal/runtime/acceptance"
	"neo-code/internal/runtime/controlplane"
	"neo-code/internal/runtime/verify"
	agentsession "neo-code/internal/session"
)

const finalContinueReminder = "There are unfinished required todos or unmet acceptance checks. Continue execution. Do not finalize yet."

// beforeAcceptFinal 在 runtime 接受模型 final 前执行双门控验收。
func (s *Service) beforeAcceptFinal(
	ctx context.Context,
	state *runState,
	snapshot TurnBudgetSnapshot,
	assistant providertypes.Message,
	completionPassed bool,
) (acceptance.AcceptanceDecision, error) {
	if state == nil {
		return acceptance.AcceptanceDecision{}, nil
	}

	verificationCfg := snapshot.Config.Runtime.Verification.Clone()
	if !verificationCfg.FinalInterceptValue() {
		return acceptance.AcceptanceDecision{
			Status:             acceptance.AcceptanceAccepted,
			StopReason:         controlplane.StopReasonCompatibilityFallback,
			UserVisibleSummary: "已通过兼容路径接受 final（final_intercept 关闭）。",
			InternalSummary:    "verification final intercept disabled, compatibility fallback accepted",
			HasProgress:        true,
		}, nil
	}

	policy := acceptance.DefaultPolicy{
		Executor: verify.PolicyCommandExecutor{},
	}
	engine := acceptance.NewEngine(policy)

	maxNoProgress := verificationCfg.MaxNoProgress
	if maxNoProgress <= 0 {
		maxNoProgress = 3
	}
	input := acceptance.FinalAcceptanceInput{
		CompletionGate: acceptance.CompletionGateDecision{
			Passed: completionPassed,
			Reason: string(state.completion.CompletionBlockedReason),
		},
		VerificationInput: verify.FinalVerifyInput{
			SessionID:          state.session.ID,
			RunID:              state.runID,
			TaskID:             state.taskID,
			Workdir:            snapshot.Workdir,
			Messages:           buildVerifyMessages(state.session.Messages),
			Todos:              buildVerifyTodos(state.session.Todos),
			LastAssistantFinal: renderPartsForVerification(assistant.Parts),
			ToolResults:        nil,
			RuntimeState: verify.RuntimeStateSnapshot{
				Turn:                 state.turn,
				MaxTurns:             resolveRuntimeMaxTurns(snapshot.Config.Runtime),
				MaxTurnsReached:      state.maxTurnsReached,
				FinalInterceptStreak: state.finalInterceptStreak,
			},
			Metadata: map[string]any{
				"task_type": inferTaskType(state),
			},
			VerificationConfig: verificationCfg,
		},
		NoProgressExceeded: state.finalInterceptStreak >= maxNoProgress,
		MaxTurnsReached:    state.maxTurnsReached,
		MaxTurnsLimit:      state.maxTurnsLimit,
	}

	return engine.EvaluateFinal(ctx, input)
}

// recordAcceptanceTerminal 将 acceptance 输出映射为 runtime 唯一终态记录。
func recordAcceptanceTerminal(state *runState, decision acceptance.AcceptanceDecision) {
	if state == nil {
		return
	}
	status := acceptance.TerminalStatusFromAcceptance(decision.Status)
	state.markTerminalDecision(status, decision.StopReason, strings.TrimSpace(decision.InternalSummary))
}

// buildVerifyTodos 将 session todo 转换为 verifier 快照。
func buildVerifyTodos(items []agentsession.TodoItem) []verify.TodoSnapshot {
	if len(items) == 0 {
		return nil
	}
	todos := make([]verify.TodoSnapshot, 0, len(items))
	for _, item := range items {
		todos = append(todos, verify.TodoSnapshot{
			ID:            strings.TrimSpace(item.ID),
			Content:       strings.TrimSpace(item.Content),
			Status:        strings.TrimSpace(string(item.Status)),
			Required:      item.RequiredValue(),
			BlockedReason: string(item.BlockedReasonValue()),
			RetryCount:    item.RetryCount,
			RetryLimit:    item.RetryLimit,
			FailureReason: strings.TrimSpace(item.FailureReason),
		})
	}
	return todos
}

// buildVerifyMessages 将会话消息压缩为 verifier 所需最小快照。
func buildVerifyMessages(messages []providertypes.Message) []verify.MessageLike {
	if len(messages) == 0 {
		return nil
	}
	out := make([]verify.MessageLike, 0, len(messages))
	for _, message := range messages {
		out = append(out, verify.MessageLike{
			Role:    strings.TrimSpace(message.Role),
			Content: renderPartsForVerification(message.Parts),
		})
	}
	return out
}

// renderPartsForVerification 将消息分片合并为 verifier 侧可读文本。
func renderPartsForVerification(parts []providertypes.ContentPart) string {
	if len(parts) == 0 {
		return ""
	}
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Kind != providertypes.ContentPartText {
			continue
		}
		text := strings.TrimSpace(part.Text)
		if text == "" {
			continue
		}
		segments = append(segments, text)
	}
	return strings.Join(segments, "\n")
}

// inferTaskType 基于 task_id 与 task_state 文本推断当前任务类型。
func inferTaskType(state *runState) string {
	if state == nil {
		return "unknown"
	}
	corpus := strings.ToLower(strings.TrimSpace(
		state.taskID + " " + state.session.TaskState.Goal + " " + state.session.TaskState.NextStep,
	))
	switch {
	case strings.Contains(corpus, "fix bug"), strings.Contains(corpus, "bugfix"):
		return "fix_bug"
	case strings.Contains(corpus, "refactor"):
		return "refactor"
	case strings.Contains(corpus, "edit code"), strings.Contains(corpus, "modify code"), strings.Contains(corpus, "patch"):
		return "edit_code"
	case strings.Contains(corpus, "create file"), strings.Contains(corpus, "scaffold"):
		return "create_file"
	case strings.Contains(corpus, "docs"), strings.Contains(corpus, "documentation"):
		return "docs"
	case strings.Contains(corpus, "config"), strings.Contains(corpus, "yaml"), strings.Contains(corpus, "json"):
		return "config"
	default:
		return "unknown"
	}
}

// applyAcceptanceResultProgress 根据 acceptance 输出更新 final 拦截熔断计数器。
func applyAcceptanceResultProgress(state *runState, decision acceptance.AcceptanceDecision) {
	if state == nil {
		return
	}
	switch decision.Status {
	case acceptance.AcceptanceContinue:
		if decision.HasProgress {
			state.finalInterceptStreak = 0
			return
		}
		state.finalInterceptStreak++
	default:
		state.finalInterceptStreak = 0
	}
}
