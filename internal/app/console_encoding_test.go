package app

import (
	"errors"
	"testing"
)

func TestEnsureConsoleUTF8SetsOutputThenInput(t *testing.T) {
	originalOutput := setConsoleOutputCodePage
	originalInput := setConsoleInputCodePage
	t.Cleanup(func() {
		setConsoleOutputCodePage = originalOutput
		setConsoleInputCodePage = originalInput
	})

	calls := make([]string, 0, 2)
	setConsoleOutputCodePage = func(codePage uint32) error {
		if codePage != utf8CodePage {
			t.Fatalf("expected utf8 code page %d, got %d", utf8CodePage, codePage)
		}
		calls = append(calls, "output")
		return nil
	}
	setConsoleInputCodePage = func(codePage uint32) error {
		if codePage != utf8CodePage {
			t.Fatalf("expected utf8 code page %d, got %d", utf8CodePage, codePage)
		}
		calls = append(calls, "input")
		return nil
	}

	ensureConsoleUTF8()

	if len(calls) != 2 || calls[0] != "output" || calls[1] != "input" {
		t.Fatalf("expected output->input order, got %+v", calls)
	}
}

func TestEnsureConsoleUTF8SkipsInputWhenOutputFails(t *testing.T) {
	originalOutput := setConsoleOutputCodePage
	originalInput := setConsoleInputCodePage
	t.Cleanup(func() {
		setConsoleOutputCodePage = originalOutput
		setConsoleInputCodePage = originalInput
	})

	outputErr := errors.New("output failed")
	setConsoleOutputCodePage = func(codePage uint32) error {
		return outputErr
	}
	inputCalled := false
	setConsoleInputCodePage = func(codePage uint32) error {
		inputCalled = true
		return nil
	}

	ensureConsoleUTF8()

	if inputCalled {
		t.Fatalf("expected input code page setup to be skipped when output setup fails")
	}
}
