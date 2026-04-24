package verify

import (
	"context"

	"neo-code/internal/runtime/controlplane"
)

// Orchestrator 按固定顺序执行 verifier 并收敛 verification gate。
type Orchestrator struct {
	Verifiers []FinalVerifier
}

// RunFinalVerification 执行 verifier 列表并生成统一 gate 决议。
func (o Orchestrator) RunFinalVerification(ctx context.Context, input FinalVerifyInput) (VerificationGateDecision, error) {
	results := make([]VerificationResult, 0, len(o.Verifiers))
	waitingExternal := false
	for _, verifier := range o.Verifiers {
		if verifier == nil {
			continue
		}
		result, err := verifier.VerifyFinal(ctx, input)
		if err != nil {
			result = VerificationResult{
				Name:       verifier.Name(),
				Status:     VerificationFail,
				Summary:    err.Error(),
				Reason:     "verifier execution error",
				ErrorClass: ErrorClassUnknown,
			}
		}
		result = NormalizeResult(result)
		if result.WaitingExternal {
			waitingExternal = true
		}
		results = append(results, result)
	}

	decision := VerificationGateDecision{
		Passed:  true,
		Reason:  controlplane.StopReasonAccepted,
		Results: results,
	}
	for _, result := range results {
		switch result.Status {
		case VerificationFail:
			decision.Passed = false
			decision.Reason = controlplane.StopReasonVerificationFailed
			return decision, nil
		case VerificationHardBlock:
			decision.Passed = false
			if waitingExternal {
				decision.Reason = controlplane.StopReasonTodoWaitingExternal
			} else {
				decision.Reason = controlplane.StopReasonTodoNotConverged
			}
		case VerificationSoftBlock:
			decision.Passed = false
			if decision.Reason == controlplane.StopReasonAccepted {
				decision.Reason = controlplane.StopReasonTodoNotConverged
			}
		}
	}
	return decision, nil
}
