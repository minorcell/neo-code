package tui

import (
	"strings"
	"testing"
)

func TestCommandMenuItem(t *testing.T) {
	item := commandMenuItem{
		title:           "Test Command",
		description:     "Test description",
		filter:          "test",
		highlight:       false,
		replacement:     "/test",
		useReplaceRange: false,
		replaceStart:    0,
		replaceEnd:      0,
		openFileBrowser: false,
	}

	if item.Title() != "Test Command" {
		t.Errorf("Title() = %v, want Test Command", item.Title())
	}
	if item.Description() != "Test description" {
		t.Errorf("Description() = %v, want Test description", item.Description())
	}
	if item.FilterValue() != "test" {
		t.Errorf("FilterValue() = %v, want test", item.FilterValue())
	}
}

func TestCommandMenuItemWithEmptyFilter(t *testing.T) {
	item := commandMenuItem{
		title:       "Command",
		description: "Description",
		filter:      "",
	}

	if item.FilterValue() != "command description" {
		t.Errorf("FilterValue() = %v, want command description", item.FilterValue())
	}
}

func TestCommandMenuItemFilterValueCase(t *testing.T) {
	item := commandMenuItem{
		title:       "UPPERCASE",
		description: "Description",
		filter:      "lowercase",
	}

	if !strings.Contains(item.FilterValue(), "lowercase") {
		t.Errorf("FilterValue() should contain lowercase, got %v", item.FilterValue())
	}
}

func TestSelectionItem(t *testing.T) {
	item := selectionItem{
		id:          "test-id",
		name:        "Test Name",
		description: "Test description",
	}

	if item.Title() != "Test Name" {
		t.Errorf("Title() = %v, want Test Name", item.Title())
	}
	if item.Description() != "Test description" {
		t.Errorf("Description() = %v, want Test description", item.Description())
	}
	if !strings.Contains(item.FilterValue(), "test-id") {
		t.Errorf("FilterValue() should contain test-id, got %v", item.FilterValue())
	}
}

func TestCommandMenuView(t *testing.T) {
	styles := newStyles()
	model := newCommandMenuModel(styles)

	v := model.View()
	if v == "" {
		t.Error("View() returned empty string")
	}
}
