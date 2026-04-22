package tools

import "testing"

func TestToolResultMetadataMarksFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata map[string]any
		want     bool
	}{
		{name: "empty", metadata: nil, want: false},
		{name: "ok true bool", metadata: map[string]any{"ok": true}, want: false},
		{name: "ok false bool", metadata: map[string]any{"ok": false}, want: true},
		{name: "ok false string", metadata: map[string]any{"ok": "false"}, want: true},
		{name: "ok one number", metadata: map[string]any{"ok": 1}, want: false},
		{name: "ok zero number", metadata: map[string]any{"ok": 0}, want: true},
		{name: "ok invalid string with exit code", metadata: map[string]any{"ok": "unknown", "exit_code": 2}, want: true},
		{name: "exit code zero", metadata: map[string]any{"exit_code": 0}, want: false},
		{name: "exit code string non-zero", metadata: map[string]any{"exit_code": "3"}, want: true},
		{name: "exit code tiny positive float", metadata: map[string]any{"exit_code": 0.1}, want: true},
		{name: "exit code tiny negative float", metadata: map[string]any{"exit_code": -0.1}, want: true},
		{name: "exit code invalid string", metadata: map[string]any{"exit_code": "x"}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ToolResultMetadataMarksFailure(tt.metadata); got != tt.want {
				t.Fatalf("ToolResultMetadataMarksFailure(%v) = %v, want %v", tt.metadata, got, tt.want)
			}
		})
	}
}
