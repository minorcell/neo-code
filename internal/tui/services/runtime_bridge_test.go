package services

import (
	"testing"
	"time"

	tuistate "neo-code/internal/tui/state"
)

func TestRuntimeBridgeMappings(t *testing.T) {
	context := MapRunContextPayload("run-1", "session-1", RuntimeRunContextPayload{
		Provider: "openai",
		Model:    "gpt-5.4",
		Workdir:  "/repo",
		Mode:     "act",
	})
	if context.RunID != "run-1" || context.Provider != "openai" {
		t.Fatalf("unexpected context mapping: %+v", context)
	}

	tool := MapToolStatusPayload(RuntimeToolStatusPayload{
		ToolCallID: "call-1",
		ToolName:   "filesystem_edit",
		Status:     "succeeded",
		Message:    "ok",
		DurationMS: 120,
	})
	if tool.Status != tuistate.ToolLifecycleSucceeded || tool.DurationMS != 120 {
		t.Fatalf("unexpected tool mapping: %+v", tool)
	}

	usage := MapUsagePayload(RuntimeUsagePayload{
		Run:     RuntimeUsageSnapshot{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
		Session: RuntimeUsageSnapshot{InputTokens: 40, OutputTokens: 50, TotalTokens: 90},
	})
	if usage.RunTotalTokens != 30 || usage.SessionTotalTokens != 90 {
		t.Fatalf("unexpected usage mapping: %+v", usage)
	}
}

func TestRuntimeBridgeParsers(t *testing.T) {
	ctx, ok := ParseRunContextPayload(map[string]any{
		"Provider": "openai",
		"Model":    "gpt-5.4",
		"Workdir":  "/repo",
		"Mode":     "act",
	})
	if !ok || ctx.Provider != "openai" {
		t.Fatalf("expected run_context payload to parse, got %+v", ctx)
	}

	tool, ok := ParseToolStatusPayload(map[string]any{
		"ToolCallID": "call-1",
		"ToolName":   "filesystem_edit",
		"Status":     "running",
		"DurationMS": int64(99),
	})
	if !ok || tool.ToolCallID != "call-1" || tool.DurationMS != 99 {
		t.Fatalf("expected tool_status payload to parse, got %+v", tool)
	}

	usage, ok := ParseUsagePayload(map[string]any{
		"Run": map[string]any{
			"InputTokens":  1,
			"OutputTokens": 2,
			"TotalTokens":  3,
		},
		"Session": map[string]any{
			"InputTokens":  10,
			"OutputTokens": 20,
			"TotalTokens":  30,
		},
	})
	if !ok || usage.Run.TotalTokens != 3 || usage.Session.TotalTokens != 30 {
		t.Fatalf("expected usage payload to parse, got %+v", usage)
	}
}

func TestMergeToolStatesHandlesDuplicateAndLimit(t *testing.T) {
	now := time.Now()
	states := []ToolStateVM{
		{
			ToolCallID: "call-1",
			ToolName:   "tool-a",
			Status:     tuistate.ToolLifecycleRunning,
			UpdatedAt:  now,
		},
	}

	updated := MergeToolStates(states, ToolStateVM{
		ToolCallID: "call-1",
		ToolName:   "tool-a",
		Status:     tuistate.ToolLifecycleSucceeded,
		UpdatedAt:  now.Add(time.Second),
	}, 2)
	if len(updated) != 1 || updated[0].Status != tuistate.ToolLifecycleSucceeded {
		t.Fatalf("expected duplicate to be replaced, got %+v", updated)
	}

	updated = MergeToolStates(updated, ToolStateVM{
		ToolCallID: "call-2",
		ToolName:   "tool-b",
		Status:     tuistate.ToolLifecycleRunning,
		UpdatedAt:  now.Add(2 * time.Second),
	}, 1)
	if len(updated) != 1 || updated[0].ToolCallID != "call-2" {
		t.Fatalf("expected limit to keep newest item, got %+v", updated)
	}
}

func TestMapRunSnapshot(t *testing.T) {
	now := time.Now()
	context, tools, usage := MapRunSnapshot(RuntimeRunSnapshot{
		RunID:     "run-2",
		SessionID: "session-2",
		Context: RuntimeRunContextSnapshot{
			RunID:     "run-2",
			SessionID: "session-2",
			Provider:  "openai",
			Model:     "gpt-5.4-mini",
			Workdir:   "/repo",
			Mode:      "act",
		},
		ToolStates: []RuntimeToolStateSnapshot{
			{
				ToolCallID: "call-1",
				ToolName:   "filesystem_read_file",
				Status:     "succeeded",
				Message:    "ok",
				DurationMS: 88,
				UpdatedAt:  now,
			},
		},
		Usage:        RuntimeUsageSnapshot{InputTokens: 3, OutputTokens: 7, TotalTokens: 10},
		SessionUsage: RuntimeUsageSnapshot{InputTokens: 30, OutputTokens: 70, TotalTokens: 100},
	})

	if context.RunID != "run-2" || len(tools) != 1 || usage.SessionTotalTokens != 100 {
		t.Fatalf("unexpected run snapshot mapping: context=%+v tools=%+v usage=%+v", context, tools, usage)
	}
}
