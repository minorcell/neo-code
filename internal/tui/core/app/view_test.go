package tui

import (
	"strings"
	"testing"
)

func TestRenderPickerHelpMode(t *testing.T) {
	app, _ := newTestApp(t)
	app.refreshHelpPicker()
	app.state.ActivePicker = pickerHelp

	view := app.renderPicker(48, 14)
	if !strings.Contains(view, helpPickerTitle) {
		t.Fatalf("expected help picker title in view")
	}
	if !strings.Contains(view, helpPickerSubtitle) {
		t.Fatalf("expected help picker subtitle in view")
	}
}

func TestRenderWaterfallUsesLayoutTranscriptHeight(t *testing.T) {
	app, _ := newTestApp(t)
	app.state.ActivePicker = pickerNone
	app.state.InputText = "test"
	app.input.SetValue("test")
	app.transcript.SetContent("line1\nline2")
	app.transcript.Height = 17

	view := app.renderWaterfall(80, 24)
	if strings.TrimSpace(view) == "" {
		t.Fatalf("expected non-empty waterfall view")
	}
}

func TestRenderWaterfallUsesHelpPickerDynamicHeight(t *testing.T) {
	app, _ := newTestApp(t)
	app.refreshHelpPicker()
	app.state.ActivePicker = pickerHelp
	app.helpPicker.SetSize(40, 20)

	view := app.renderWaterfall(80, 30)
	if !strings.Contains(view, helpPickerTitle) {
		t.Fatalf("expected help picker title in waterfall")
	}
}

func TestActivePickerHeightHelpUsesConfiguredHeight(t *testing.T) {
	app, _ := newTestApp(t)
	app.state.ActivePicker = pickerHelp
	app.helpPicker.SetSize(30, 18)

	if got := app.activePickerHeight(); got != 18 {
		t.Fatalf("expected help picker height 18, got %d", got)
	}
}
