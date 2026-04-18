package memo

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"neo-code/internal/memo"
	"neo-code/internal/tools"
)

func TestRemoveToolName(t *testing.T) {
	tool := NewRemoveTool(nil)
	if tool.Name() != tools.ToolNameMemoRemove {
		t.Fatalf("Name() = %q, want %q", tool.Name(), tools.ToolNameMemoRemove)
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

func TestRemoveToolExecuteSuccess(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeUser,
		Title:   "prefer chinese comments",
		Content: "always write comments in chinese",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeUser,
		Title:   "prefer short functions",
		Content: "keep functions under 30 lines",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	tool := NewRemoveTool(svc)
	args, _ := json.Marshal(removeInput{Keyword: "prefer"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}
	if !strings.Contains(result.Content, "Removed 2 memo(s)") {
		t.Fatalf("unexpected content: %q", result.Content)
	}

	// 验证记忆已被真正删除
	entries, err := svc.List(context.Background(), memo.ScopeUser)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries after removal, got %d", len(entries))
	}
}

func TestRemoveToolExecuteNoMatch(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeUser,
		Title:   "prefer tabs",
		Content: "use tabs for indentation",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	tool := NewRemoveTool(svc)
	args, _ := json.Marshal(removeInput{Keyword: "nonexistent"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}
	if !strings.Contains(result.Content, "No memos matching") {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

func TestRemoveToolExecuteWithScopeFilter(t *testing.T) {
	svc := newTestService(t)

	// 添加 user 作用域的记忆（TypeUser -> ScopeUser）
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeUser,
		Title:   "prefer dark theme",
		Content: "dark theme preference",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	// 添加 project 作用域的记忆（TypeProject -> ScopeProject）
	if err := svc.Add(context.Background(), memo.Entry{
		Type:    memo.TypeProject,
		Title:   "prefer dark theme for project",
		Content: "dark theme for this project",
		Source:  memo.SourceUserManual,
	}); err != nil {
		t.Fatalf("seed Add() error = %v", err)
	}

	tool := NewRemoveTool(svc)
	// 只删除 user 作用域的匹配条目
	args, _ := json.Marshal(removeInput{Keyword: "dark theme", Scope: "user"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}
	if !strings.Contains(result.Content, "Removed 1 memo(s)") {
		t.Fatalf("unexpected content: %q", result.Content)
	}

	// 验证 user 作用域已清空
	userEntries, err := svc.List(context.Background(), memo.ScopeUser)
	if err != nil {
		t.Fatalf("List(user) error = %v", err)
	}
	if len(userEntries) != 0 {
		t.Fatalf("expected 0 user entries, got %d", len(userEntries))
	}

	// 验证 project 作用域保留
	projEntries, err := svc.List(context.Background(), memo.ScopeProject)
	if err != nil {
		t.Fatalf("List(project) error = %v", err)
	}
	if len(projEntries) != 1 {
		t.Fatalf("expected 1 project entry, got %d", len(projEntries))
	}
}

func TestRemoveToolExecuteEmptyKeyword(t *testing.T) {
	svc := newTestService(t)
	tool := NewRemoveTool(svc)

	args, _ := json.Marshal(removeInput{Keyword: "   "})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err == nil || !result.IsError {
		t.Fatalf("expected empty keyword error, got result=%+v err=%v", result, err)
	}
}

func TestRemoveToolExecuteNilService(t *testing.T) {
	tool := NewRemoveTool(nil)
	args, _ := json.Marshal(removeInput{Keyword: "test"})

	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err == nil || !result.IsError {
		t.Fatalf("expected nil service error, got result=%+v err=%v", result, err)
	}
}

func TestRemoveToolExecuteInvalidJSON(t *testing.T) {
	tool := NewRemoveTool(nil)
	if _, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: []byte("not json")}); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestRemoveToolExecuteInvalidScope(t *testing.T) {
	svc := newTestService(t)
	tool := NewRemoveTool(svc)

	args, _ := json.Marshal(removeInput{Keyword: "test", Scope: "invalid"})
	result, err := tool.Execute(context.Background(), tools.ToolCallInput{Arguments: args})
	if err == nil || !result.IsError {
		t.Fatalf("expected invalid scope error, got result=%+v err=%v", result, err)
	}
}

func TestParseMemoScope(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		allowAll bool
		want     memo.Scope
		wantErr  bool
	}{
		{"空字符串 allowAll=true 默认 ScopeAll", "", true, memo.ScopeAll, false},
		{"空字符串 allowAll=false 返回错误", "", false, "", true},
		{"user", "user", true, memo.ScopeUser, false},
		{"USER 大写", "USER", true, memo.ScopeUser, false},
		{"project", "project", true, memo.ScopeProject, false},
		{"PROJECT 大写", "PROJECT", true, memo.ScopeProject, false},
		{"all allowAll=true", "all", true, memo.ScopeAll, false},
		{"all allowAll=false 返回错误", "all", false, "", true},
		{"不合法的 scope", "invalid", true, "", true},
		{"带空格的 user", " user ", true, memo.ScopeUser, false},
		{"不合法 scope allowAll=false", "badscope", false, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMemoScope(tt.raw, tt.allowAll)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseMemoScope(%q, %v) expected error, got scope=%q", tt.raw, tt.allowAll, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseMemoScope(%q, %v) unexpected error: %v", tt.raw, tt.allowAll, err)
			}
			if got != tt.want {
				t.Fatalf("parseMemoScope(%q, %v) = %q, want %q", tt.raw, tt.allowAll, got, tt.want)
			}
		})
	}
}
