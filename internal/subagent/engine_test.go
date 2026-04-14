package subagent

import (
	"context"
	"testing"
)

func TestDefaultEngineRunStep(t *testing.T) {
	t.Parallel()

	engine := defaultEngine{}

	t.Run("uses expected output as summary", func(t *testing.T) {
		t.Parallel()

		out, err := engine.RunStep(context.Background(), StepInput{
			Task: Task{
				Goal:           "goal",
				ExpectedOutput: "expected",
			},
		})
		if err != nil {
			t.Fatalf("RunStep() error = %v", err)
		}
		if !out.Done {
			t.Fatalf("expected done output")
		}
		if out.Output.Summary != "expected" {
			t.Fatalf("summary = %q, want %q", out.Output.Summary, "expected")
		}
	})

	t.Run("falls back to goal", func(t *testing.T) {
		t.Parallel()

		out, err := engine.RunStep(context.Background(), StepInput{
			Task: Task{
				Goal:           "goal-value",
				ExpectedOutput: " ",
			},
		})
		if err != nil {
			t.Fatalf("RunStep() error = %v", err)
		}
		if out.Output.Summary != "goal-value" {
			t.Fatalf("summary = %q, want %q", out.Output.Summary, "goal-value")
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := engine.RunStep(ctx, StepInput{Task: Task{Goal: "g"}}); err == nil {
			t.Fatalf("expected context error")
		}
	})
}
