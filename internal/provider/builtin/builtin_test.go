package builtin

import (
	"testing"

	"neo-code/internal/provider/gemini"
	"neo-code/internal/provider/openai"
)

func TestDefaultConfigIncludesOpenAIAndGemini(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	if len(cfg.Providers) != 2 {
		t.Fatalf("expected 2 builtin providers, got %d", len(cfg.Providers))
	}
	if cfg.Providers[0].Name != openai.Name {
		t.Fatalf("expected first provider %q, got %q", openai.Name, cfg.Providers[0].Name)
	}
	if cfg.Providers[1].Name != gemini.Name {
		t.Fatalf("expected second provider %q, got %q", gemini.Name, cfg.Providers[1].Name)
	}
	if cfg.SelectedProvider != openai.Name {
		t.Fatalf("expected selected provider %q, got %q", openai.Name, cfg.SelectedProvider)
	}
	if cfg.CurrentModel != openai.DefaultModel {
		t.Fatalf("expected current model %q, got %q", openai.DefaultModel, cfg.CurrentModel)
	}
}
