package memo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	agentcontext "neo-code/internal/context"
)

func TestContextSourceEmpty(t *testing.T) {
	store := newMemoryTestStore()
	source := NewContextSource(store)

	sections, err := source.Sections(context.Background(), agentcontext.BuildInput{})
	if err != nil {
		t.Fatalf("Sections() error = %v", err)
	}
	if len(sections) != 0 {
		t.Fatalf("len(sections) = %d, want 0", len(sections))
	}
}

func TestContextSourceRendersTwoScopes(t *testing.T) {
	store := newMemoryTestStore()
	store.indexes[ScopeUser] = &Index{Entries: []Entry{{Type: TypeUser, Title: "user pref", TopicFile: "u.md"}}}
	store.indexes[ScopeProject] = &Index{Entries: []Entry{{Type: TypeProject, Title: "project fact", TopicFile: "p.md"}}}
	source := NewContextSource(store)

	sections, err := source.Sections(context.Background(), agentcontext.BuildInput{})
	if err != nil {
		t.Fatalf("Sections() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("len(sections) = %d, want 2", len(sections))
	}
	if sections[0].Title != "User Memo" || !strings.Contains(sections[0].Content, "user pref") {
		t.Fatalf("unexpected user section: %+v", sections[0])
	}
	if sections[1].Title != "Project Memo" || !strings.Contains(sections[1].Content, "project fact") {
		t.Fatalf("unexpected project section: %+v", sections[1])
	}
}

func TestContextSourceCache(t *testing.T) {
	store := newMemoryTestStore()
	store.indexes[ScopeUser] = &Index{Entries: []Entry{{Type: TypeUser, Title: "first"}}}
	source := NewContextSource(store, WithCacheTTL(10*time.Second))

	sections, err := source.Sections(context.Background(), agentcontext.BuildInput{})
	if err != nil {
		t.Fatalf("Sections() error = %v", err)
	}
	if !strings.Contains(sections[0].Content, "first") {
		t.Fatalf("expected cached content to include first, got %q", sections[0].Content)
	}

	store.indexes[ScopeUser].Entries[0].Title = "second"
	sections, err = source.Sections(context.Background(), agentcontext.BuildInput{})
	if err != nil {
		t.Fatalf("Sections() second call error = %v", err)
	}
	if !strings.Contains(sections[0].Content, "first") {
		t.Fatalf("expected cached content to stay stale, got %q", sections[0].Content)
	}
	if store.loadIndexCalls != 2 {
		t.Fatalf("LoadIndex() calls = %d, want 2 (one per scope)", store.loadIndexCalls)
	}
}

func TestContextSourceInvalidateCache(t *testing.T) {
	store := newMemoryTestStore()
	store.indexes[ScopeUser] = &Index{Entries: []Entry{{Type: TypeUser, Title: "old"}}}
	source := NewContextSource(store, WithCacheTTL(10*time.Second))

	if _, err := source.Sections(context.Background(), agentcontext.BuildInput{}); err != nil {
		t.Fatalf("Sections() warm cache error = %v", err)
	}
	store.indexes[ScopeUser].Entries[0].Title = "new"

	source.(*memoContextSource).InvalidateCache()
	sections, err := source.Sections(context.Background(), agentcontext.BuildInput{})
	if err != nil {
		t.Fatalf("Sections() after invalidation error = %v", err)
	}
	if !strings.Contains(sections[0].Content, "new") {
		t.Fatalf("expected invalidated content to include new, got %q", sections[0].Content)
	}
}

func TestContextSourceStoreErrorReturnsNil(t *testing.T) {
	store := newMemoryTestStore()
	store.err = errors.New("boom")
	source := NewContextSource(store)

	sections, err := source.Sections(context.Background(), agentcontext.BuildInput{})
	if err != nil {
		t.Fatalf("Sections() should suppress store error, got %v", err)
	}
	if sections != nil {
		t.Fatalf("sections = %+v, want nil", sections)
	}
}

func TestContextSourceReadsGlobalUserAndScopedProject(t *testing.T) {
	baseDir := t.TempDir()
	storeA := NewFileStore(baseDir, "/workspace/a")
	storeB := NewFileStore(baseDir, "/workspace/b")

	if err := storeA.SaveIndex(context.Background(), ScopeUser, &Index{
		Entries: []Entry{{Type: TypeUser, Title: "global user pref", TopicFile: "u.md"}},
	}); err != nil {
		t.Fatalf("SaveIndex(user) error = %v", err)
	}
	if err := storeA.SaveIndex(context.Background(), ScopeProject, &Index{
		Entries: []Entry{{Type: TypeProject, Title: "workspace a fact", TopicFile: "a.md"}},
	}); err != nil {
		t.Fatalf("SaveIndex(project a) error = %v", err)
	}
	if err := storeB.SaveIndex(context.Background(), ScopeProject, &Index{
		Entries: []Entry{{Type: TypeProject, Title: "workspace b fact", TopicFile: "b.md"}},
	}); err != nil {
		t.Fatalf("SaveIndex(project b) error = %v", err)
	}

	source := NewContextSource(storeB)
	sections, err := source.Sections(context.Background(), agentcontext.BuildInput{})
	if err != nil {
		t.Fatalf("Sections() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("len(sections) = %d, want 2", len(sections))
	}
	if !strings.Contains(sections[0].Content, "global user pref") {
		t.Fatalf("expected user memo to include global entry, got %q", sections[0].Content)
	}
	if strings.Contains(sections[1].Content, "workspace a fact") {
		t.Fatalf("project memo leaked another workspace: %q", sections[1].Content)
	}
	if !strings.Contains(sections[1].Content, "workspace b fact") {
		t.Fatalf("expected project memo to include current workspace entry, got %q", sections[1].Content)
	}
}
