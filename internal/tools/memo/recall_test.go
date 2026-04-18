package memo

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"neo-code/internal/memo"
	"neo-code/internal/tools"
)

func TestRecallToolName(t *testing.T) {
	tool := NewRecallTool(nil)
	if tool.Name() != tools.ToolNameMemoRecall {
		t.Fatalf("Name() = %q, want %q", tool.Name(), tools.ToolNameMemoRecall)
	}
}

func TestRecallToolExecuteSuccess(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeUser,
		Title:   "prefer chinese comments",
		Content: "prefer chinese comments",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	tool := NewRecallTool(svc)
	args, _ := json.Marshal(recallInput{Keyword: "chinese"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError || !strings.Contains(result.Content, "Found 1 memory topic") {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.Contains(result.Content, "[user]") {
		t.Fatalf("expected scoped header, got %q", result.Content)
	}
}

func TestRecallToolExecuteNoMatch(t *testing.T) {
	tool := NewRecallTool(newTestService(t))
	args, _ := json.Marshal(recallInput{Keyword: "missing"})

	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError || !strings.Contains(result.Content, "No memories found") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRecallToolExecuteBadInput(t *testing.T) {
	tool := NewRecallTool(newTestService(t))

	if _, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: []byte("not json")}); err == nil {
		t.Fatal("expected invalid JSON error")
	}
	args, _ := json.Marshal(recallInput{Keyword: ""})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err == nil || !result.IsError {
		t.Fatalf("expected empty keyword error, got result=%+v err=%v", result, err)
	}
}

func TestRecallToolExecuteNilService(t *testing.T) {
	tool := NewRecallTool(nil)
	args, _ := json.Marshal(recallInput{Keyword: "x"})

	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err == nil || !result.IsError {
		t.Fatalf("expected nil service error, got result=%+v err=%v", result, err)
	}
}

func TestRecallToolDescriptionAndSchema(t *testing.T) {
	tool := NewRecallTool(nil)
	if tool.Description() == "" {
		t.Fatal("Description() should not be empty")
	}
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Schema() should not be nil")
	}
	if tool.MicroCompactPolicy() != tools.MicroCompactPolicyPreserveHistory {
		t.Fatalf("MicroCompactPolicy() = %v, want PreserveHistory", tool.MicroCompactPolicy())
	}
}

func TestRecallToolExecuteWithScopeFilter(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeUser,
		Title:   "user pref",
		Content: "user pref content",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("Add user: %v", err)
	}
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeFeedback,
		Title:   "feedback pref",
		Content: "feedback pref content",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("Add feedback: %v", err)
	}

	tool := NewRecallTool(svc)
	args, _ := json.Marshal(recallInput{Keyword: "pref", Scope: "user"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError || !strings.Contains(result.Content, "Found 1 memory topic") {
		t.Fatalf("expected 1 user result, got: %s", result.Content)
	}
	if strings.Contains(result.Content, "feedback") {
		t.Fatalf("should not contain feedback entry: %s", result.Content)
	}
}

func TestRecallToolExecuteInvalidScope(t *testing.T) {
	tool := NewRecallTool(newTestService(t))
	args, _ := json.Marshal(recallInput{Keyword: "test", Scope: "badscope"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err == nil || !result.IsError {
		t.Fatalf("expected bad scope error, got result=%+v err=%v", result, err)
	}
}

func TestRecallToolExecuteAppliesOutputLimit(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeReference,
		Title:   "long memory",
		Content: strings.Repeat("x", tools.DefaultOutputLimitBytes+1024),
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	tool := NewRecallTool(svc)
	args, _ := json.Marshal(recallInput{Keyword: "long"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError || !strings.Contains(result.Content, "...[truncated]") {
		t.Fatalf("unexpected truncated result: %+v", result)
	}
}
