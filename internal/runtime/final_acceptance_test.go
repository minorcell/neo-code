package runtime

import (
	"context"
	"testing"

	"neo-code/internal/config"
	providertypes "neo-code/internal/provider/types"
	agentsession "neo-code/internal/session"
)

func TestBeforeAcceptFinalDecisionPaths(t *testing.T) {
	t.Parallel()

	service := &Service{}
	baseCfg := config.StaticDefaults().Clone()
	baseCfg.Runtime.Verification.Enabled = boolPtr(true)
	baseCfg.Runtime.Verification.FinalIntercept = boolPtr(true)
	snapshot := TurnBudgetSnapshot{
		Config:  baseCfg,
		Workdir: t.TempDir(),
	}

	t.Run("pending required todo -> continue", func(t *testing.T) {
		t.Parallel()
		state := newRunState("run-continue", agentsession.New("continue"))
		required := true
		state.session.Todos = []agentsession.TodoItem{
			{
				ID:       "todo-1",
				Content:  "do work",
				Status:   agentsession.TodoStatusPending,
				Required: &required,
			},
		}
		decision, err := service.beforeAcceptFinal(context.Background(), &state, snapshot, providertypes.Message{
			Role:  providertypes.RoleAssistant,
			Parts: []providertypes.ContentPart{providertypes.NewTextPart("done")},
		}, true)
		if err != nil {
			t.Fatalf("beforeAcceptFinal() error = %v", err)
		}
		if decision.Status != "continue" {
			t.Fatalf("status = %q, want continue", decision.Status)
		}
	})

	t.Run("all converged -> accepted", func(t *testing.T) {
		t.Parallel()
		state := newRunState("run-accepted", agentsession.New("accepted"))
		decision, err := service.beforeAcceptFinal(context.Background(), &state, snapshot, providertypes.Message{
			Role:  providertypes.RoleAssistant,
			Parts: []providertypes.ContentPart{providertypes.NewTextPart("done")},
		}, true)
		if err != nil {
			t.Fatalf("beforeAcceptFinal() error = %v", err)
		}
		if decision.Status != "accepted" {
			t.Fatalf("status = %q, want accepted", decision.Status)
		}
	})

	t.Run("verification disabled -> compatibility fallback", func(t *testing.T) {
		t.Parallel()
		state := newRunState("run-fallback", agentsession.New("fallback"))
		cfg := snapshot.Config
		cfg.Runtime.Verification.Enabled = boolPtr(false)
		decision, err := service.beforeAcceptFinal(context.Background(), &state, TurnBudgetSnapshot{
			Config:  cfg,
			Workdir: snapshot.Workdir,
		}, providertypes.Message{}, true)
		if err != nil {
			t.Fatalf("beforeAcceptFinal() error = %v", err)
		}
		if decision.StopReason != "compatibility_fallback" {
			t.Fatalf("stop_reason = %q, want compatibility_fallback", decision.StopReason)
		}
	})

	t.Run("final intercept disabled -> compatibility fallback", func(t *testing.T) {
		t.Parallel()
		state := newRunState("run-no-intercept", agentsession.New("no-intercept"))
		cfg := snapshot.Config
		cfg.Runtime.Verification.FinalIntercept = boolPtr(false)
		decision, err := service.beforeAcceptFinal(context.Background(), &state, TurnBudgetSnapshot{
			Config:  cfg,
			Workdir: snapshot.Workdir,
		}, providertypes.Message{}, true)
		if err != nil {
			t.Fatalf("beforeAcceptFinal() error = %v", err)
		}
		if decision.StopReason != "compatibility_fallback" {
			t.Fatalf("stop_reason = %q, want compatibility_fallback", decision.StopReason)
		}
	})
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}
