package memo

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"neo-code/internal/memo"
	"neo-code/internal/tools"
)

func TestListToolName(t *testing.T) {
	tool := NewListTool(nil)
	if tool.Name() != tools.ToolNameMemoList {
		t.Fatalf("Name() = %q, want %q", tool.Name(), tools.ToolNameMemoList)
	}
	if tool.Description() == "" {
		t.Fatal("Description() should not be empty")
	}
	if tool.Schema() == nil {
		t.Fatal("Schema() should not be nil")
	}
	if tool.MicroCompactPolicy() != tools.MicroCompactPolicyPreserveHistory {
		t.Fatalf("MicroCompactPolicy() = %v, want PreserveHistory", tool.MicroCompactPolicy())
	}
}

func TestListToolExecuteEmpty(t *testing.T) {
	svc := newTestService(t)
	tool := NewListTool(svc)

	args, _ := json.Marshal(listInput{})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}
	if result.Content != "No memos stored yet." {
		t.Fatalf("Content = %q, want %q", result.Content, "No memos stored yet.")
	}
}

func TestListToolExecuteWithUserMemos(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeUser,
		Title:   "prefer go style",
		Content: "prefer go style",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	tool := NewListTool(svc)
	args, _ := json.Marshal(listInput{Scope: "user"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}
	if !strings.Contains(result.Content, "User Memo:") {
		t.Fatalf("expected 'User Memo:' header, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "[user] prefer go style") {
		t.Fatalf("expected user entry line, got %q", result.Content)
	}
	if strings.Contains(result.Content, "Project Memo:") {
		t.Fatalf("scope=user should not include Project Memo section, got %q", result.Content)
	}
}

func TestListToolExecuteWithProjectMemos(t *testing.T) {
	svc := newTestService(t)
	for _, e := range []memo.Entry{
		{Type: memo.TypeFeedback, Title: "fix logging", Content: "fix logging", Source: memo.SourceToolInitiated},
		{Type: memo.TypeProject, Title: "use grpc", Content: "use grpc", Source: memo.SourceToolInitiated},
		{Type: memo.TypeReference, Title: "design doc", Content: "design doc", Source: memo.SourceToolInitiated},
	} {
		if err := svc.Add(context.Background(), e); err != nil {
			t.Fatalf("seed Add() error = %v", err)
		}
	}

	tool := NewListTool(svc)
	args, _ := json.Marshal(listInput{Scope: "project"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}
	if !strings.Contains(result.Content, "Project Memo:") {
		t.Fatalf("expected 'Project Memo:' header, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "[feedback] fix logging") {
		t.Fatalf("expected feedback entry, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "[project] use grpc") {
		t.Fatalf("expected project entry, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "[reference] design doc") {
		t.Fatalf("expected reference entry, got %q", result.Content)
	}
	if strings.Contains(result.Content, "User Memo:") {
		t.Fatalf("scope=project should not include User Memo section, got %q", result.Content)
	}
}

func TestListToolExecuteAllScopes(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeUser,
		Title:   "dark mode",
		Content: "dark mode",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeProject,
		Title:   "use sqlite",
		Content: "use sqlite",
		Source:  memo.SourceToolInitiated,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	tool := NewListTool(svc)

	// scope="all" 应包含两层
	args, _ := json.Marshal(listInput{Scope: "all"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}
	if !strings.Contains(result.Content, "User Memo:") {
		t.Fatalf("expected 'User Memo:' header, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "Project Memo:") {
		t.Fatalf("expected 'Project Memo:' header, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "[user] dark mode") {
		t.Fatalf("expected user entry, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "[project] use sqlite") {
		t.Fatalf("expected project entry, got %q", result.Content)
	}

	// 默认空 scope 等价于 all
	args, _ = json.Marshal(listInput{})
	result2, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result2.Content != result.Content {
		t.Fatalf("default scope content = %q, want same as all = %q", result2.Content, result.Content)
	}
}

func TestListToolExecuteAllScopesPartialEmpty(t *testing.T) {
	svc := newTestService(t)
	// 只有 project 条目，user 层为空
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeProject,
		Title:   "use grpc",
		Content: "use grpc",
		Source:  memo.SourceToolInitiated,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	tool := NewListTool(svc)
	args, _ := json.Marshal(listInput{Scope: "all"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(result.Content, "User Memo:") {
		t.Fatalf("expected 'User Memo:' header for empty section, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "<empty>") {
		t.Fatalf("expected '<empty>' for user section, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "Project Memo:") {
		t.Fatalf("expected 'Project Memo:' header, got %q", result.Content)
	}
}

func TestListToolExecuteNilService(t *testing.T) {
	tool := NewListTool(nil)
	args, _ := json.Marshal(listInput{})

	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err == nil || !result.IsError {
		t.Fatalf("expected nil service error, got result=%+v err=%v", result, err)
	}
}

func TestListToolExecuteInvalidJSON(t *testing.T) {
	tool := NewListTool(nil)
	if _, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: []byte("not json")}); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestListToolExecuteInvalidScope(t *testing.T) {
	svc := newTestService(t)
	tool := NewListTool(svc)

	args, _ := json.Marshal(listInput{Scope: "badscope"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err == nil || !result.IsError {
		t.Fatalf("expected invalid scope error, got result=%+v err=%v", result, err)
	}
}
