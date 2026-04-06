package infra

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestShellArgs(t *testing.T) {
	if got := ShellArgs("bash", "pwd"); len(got) != 3 || got[0] != "bash" || got[2] != "pwd" {
		t.Fatalf("unexpected bash args: %+v", got)
	}
	if got := ShellArgs("sh", "pwd"); len(got) != 3 || got[0] != "sh" || got[2] != "pwd" {
		t.Fatalf("unexpected sh args: %+v", got)
	}
	if got := ShellArgs("unknown", "git status"); len(got) != 4 || got[0] != "powershell" {
		t.Fatalf("expected powershell fallback, got %+v", got)
	}
}

func TestSanitizeWorkspaceOutput(t *testing.T) {
	raw := []byte("\x1b[31mERROR\x1b[0m\r\nok\t\x00")
	got := SanitizeWorkspaceOutput(raw)
	if strings.Contains(got, "\x1b[31m") {
		t.Fatalf("expected ansi removed, got %q", got)
	}
	if !strings.Contains(got, "ERROR") || !strings.Contains(got, "ok") {
		t.Fatalf("expected content preserved, got %q", got)
	}
}

func TestDecodeWorkspaceOutputUTF16LE(t *testing.T) {
	utf16Data := utf16.Encode([]rune("PowerShell 输出"))
	buf := make([]byte, 2+len(utf16Data)*2)
	buf[0], buf[1] = 0xFF, 0xFE
	for i, word := range utf16Data {
		binary.LittleEndian.PutUint16(buf[2+i*2:], word)
	}

	got := DecodeWorkspaceOutput(buf)
	if !strings.Contains(got, "PowerShell") {
		t.Fatalf("expected decoded utf16 content, got %q", got)
	}
}

func TestCollectWorkspaceFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(rel), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	mustWrite("README.md")
	mustWrite("internal/tui/update.go")
	mustWrite(".git/config")
	mustWrite("node_modules/skip.js")

	files, err := CollectWorkspaceFiles(root, 10)
	if err != nil {
		t.Fatalf("CollectWorkspaceFiles() error = %v", err)
	}
	got := strings.Join(files, ",")
	if strings.Contains(got, ".git") || strings.Contains(got, "node_modules") {
		t.Fatalf("expected ignored dirs skipped, got %v", files)
	}
	if !strings.Contains(got, "README.md") || !strings.Contains(got, "internal/tui/update.go") {
		t.Fatalf("expected workspace files included, got %v", files)
	}
}

func TestCopyTextUsesInjectedWriter(t *testing.T) {
	original := clipboardWriteAll
	t.Cleanup(func() { clipboardWriteAll = original })

	captured := ""
	clipboardWriteAll = func(text string) error {
		captured = text
		return nil
	}
	if err := CopyText("hello"); err != nil {
		t.Fatalf("CopyText() error = %v", err)
	}
	if captured != "hello" {
		t.Fatalf("expected captured clipboard text, got %q", captured)
	}
}

func TestCachedMarkdownRendererBasic(t *testing.T) {
	renderer := NewCachedMarkdownRenderer("dark", 4, "(empty)")

	empty, err := renderer.Render(" \n\t ", 20)
	if err != nil {
		t.Fatalf("Render(empty) error = %v", err)
	}
	if empty != "(empty)" {
		t.Fatalf("expected empty placeholder, got %q", empty)
	}

	out, err := renderer.Render("# Title\n\n- one", 40)
	if err != nil {
		t.Fatalf("Render(markdown) error = %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected non-empty rendered markdown")
	}
	if renderer.RendererCount() != 1 || renderer.CacheCount() != 1 {
		t.Fatalf("expected renderer and cache entries, got renderers=%d cache=%d", renderer.RendererCount(), renderer.CacheCount())
	}
}

func TestCachedMarkdownRendererCacheEviction(t *testing.T) {
	renderer := NewCachedMarkdownRenderer("dark", 1, "(empty)")

	if _, err := renderer.Render("first", 20); err != nil {
		t.Fatalf("Render(first) error = %v", err)
	}
	if _, err := renderer.Render("second", 20); err != nil {
		t.Fatalf("Render(second) error = %v", err)
	}
	if renderer.CacheOrderCount() != 1 || renderer.CacheCount() != 1 {
		t.Fatalf("expected single cache entry after eviction, got order=%d cache=%d", renderer.CacheOrderCount(), renderer.CacheCount())
	}
}
