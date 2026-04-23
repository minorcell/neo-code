package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"neo-code/internal/updater"
)

func TestVersionCommandPassesPrereleaseFlag(t *testing.T) {
	originalRunner := runVersionCommand
	originalPreload := runGlobalPreload
	originalSilentCheck := runSilentUpdateCheck
	t.Cleanup(func() { runVersionCommand = originalRunner })
	t.Cleanup(func() { runGlobalPreload = originalPreload })
	t.Cleanup(func() { runSilentUpdateCheck = originalSilentCheck })

	runGlobalPreload = func(context.Context) error { return nil }
	runSilentUpdateCheck = func(context.Context) {}

	var received versionCommandOptions
	runVersionCommand = func(_ context.Context, options versionCommandOptions) (versionCommandResult, error) {
		received = options
		return versionCommandResult{
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.0.0",
			Comparable:     true,
			HasUpdate:      false,
		}, nil
	}

	command := NewRootCommand()
	command.SetArgs([]string{"version", "--prerelease"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if !received.IncludePrerelease {
		t.Fatal("expected IncludePrerelease to be true")
	}
	if stdout.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestVersionCommandShowsUpdateAvailable(t *testing.T) {
	originalRunner := runVersionCommand
	originalPreload := runGlobalPreload
	originalSilentCheck := runSilentUpdateCheck
	t.Cleanup(func() { runVersionCommand = originalRunner })
	t.Cleanup(func() { runGlobalPreload = originalPreload })
	t.Cleanup(func() { runSilentUpdateCheck = originalSilentCheck })

	runGlobalPreload = func(context.Context) error { return nil }
	runSilentUpdateCheck = func(context.Context) {}
	runVersionCommand = func(context.Context, versionCommandOptions) (versionCommandResult, error) {
		return versionCommandResult{
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.2.0",
			Comparable:     true,
			HasUpdate:      true,
		}, nil
	}

	command := NewRootCommand()
	command.SetArgs([]string{"version"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Current version: v1.0.0") {
		t.Fatalf("output = %q, want current version", output)
	}
	if !strings.Contains(output, "Latest stable version: v1.2.0") {
		t.Fatalf("output = %q, want latest version", output)
	}
	if !strings.Contains(output, "Update available: run neocode update") {
		t.Fatalf("output = %q, want update guidance", output)
	}
}

func TestVersionCommandShowsUpToDate(t *testing.T) {
	originalRunner := runVersionCommand
	originalPreload := runGlobalPreload
	originalSilentCheck := runSilentUpdateCheck
	t.Cleanup(func() { runVersionCommand = originalRunner })
	t.Cleanup(func() { runGlobalPreload = originalPreload })
	t.Cleanup(func() { runSilentUpdateCheck = originalSilentCheck })

	runGlobalPreload = func(context.Context) error { return nil }
	runSilentUpdateCheck = func(context.Context) {}
	runVersionCommand = func(context.Context, versionCommandOptions) (versionCommandResult, error) {
		return versionCommandResult{
			CurrentVersion: "v1.2.0",
			LatestVersion:  "v1.2.0",
			Comparable:     true,
			HasUpdate:      false,
		}, nil
	}

	command := NewRootCommand()
	command.SetArgs([]string{"version"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "You are on the latest version.") {
		t.Fatalf("output = %q, want up-to-date message", stdout.String())
	}
}

func TestVersionCommandCheckFailureDoesNotFailCommand(t *testing.T) {
	originalRunner := runVersionCommand
	originalPreload := runGlobalPreload
	originalSilentCheck := runSilentUpdateCheck
	t.Cleanup(func() { runVersionCommand = originalRunner })
	t.Cleanup(func() { runGlobalPreload = originalPreload })
	t.Cleanup(func() { runSilentUpdateCheck = originalSilentCheck })

	runGlobalPreload = func(context.Context) error { return nil }
	runSilentUpdateCheck = func(context.Context) {}
	runVersionCommand = func(context.Context, versionCommandOptions) (versionCommandResult, error) {
		return versionCommandResult{
			CurrentVersion: "v1.2.0",
			CheckErr:       errors.New("network down"),
		}, nil
	}

	command := NewRootCommand()
	command.SetArgs([]string{"version"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "check failed") {
		t.Fatalf("output = %q, want check failure", stdout.String())
	}
}

func TestVersionCommandSkipsComparisonForNonSemver(t *testing.T) {
	originalRunner := runVersionCommand
	originalPreload := runGlobalPreload
	originalSilentCheck := runSilentUpdateCheck
	t.Cleanup(func() { runVersionCommand = originalRunner })
	t.Cleanup(func() { runGlobalPreload = originalPreload })
	t.Cleanup(func() { runSilentUpdateCheck = originalSilentCheck })

	runGlobalPreload = func(context.Context) error { return nil }
	runSilentUpdateCheck = func(context.Context) {}
	runVersionCommand = func(context.Context, versionCommandOptions) (versionCommandResult, error) {
		return versionCommandResult{
			CurrentVersion: "dev",
			LatestVersion:  "v1.2.0",
			Comparable:     false,
		}, nil
	}

	command := NewRootCommand()
	command.SetArgs([]string{"version"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Comparison skipped: current build is non-semver.") {
		t.Fatalf("output = %q, want non-semver message", stdout.String())
	}
}

func TestDefaultVersionCommandRunnerUsesProbeOptions(t *testing.T) {
	originalProbe := runReleaseProbe
	originalReader := readCurrentVersion
	originalTimeout := versionProbeTimeout
	t.Cleanup(func() { runReleaseProbe = originalProbe })
	t.Cleanup(func() { readCurrentVersion = originalReader })
	t.Cleanup(func() { versionProbeTimeout = originalTimeout })

	readCurrentVersion = func() string { return "v1.0.0" }
	versionProbeTimeout = 2 * time.Second

	var capturedCurrent string
	var capturedIncludePrerelease bool
	var capturedTimeout time.Duration
	runReleaseProbe = func(ctx context.Context, currentVersion string, includePrerelease bool, timeout time.Duration) (updater.CheckResult, error) {
		capturedCurrent = currentVersion
		capturedIncludePrerelease = includePrerelease
		capturedTimeout = timeout
		return updater.CheckResult{
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.1.0",
			HasUpdate:      true,
		}, nil
	}

	result, err := defaultVersionCommandRunner(context.Background(), versionCommandOptions{IncludePrerelease: true})
	if err != nil {
		t.Fatalf("defaultVersionCommandRunner() error = %v", err)
	}
	if capturedCurrent != "v1.0.0" {
		t.Fatalf("captured current version = %q, want %q", capturedCurrent, "v1.0.0")
	}
	if !capturedIncludePrerelease {
		t.Fatal("expected include prerelease to be forwarded")
	}
	if capturedTimeout != 2*time.Second {
		t.Fatalf("captured timeout = %s, want %s", capturedTimeout, 2*time.Second)
	}
	if !result.HasUpdate || result.LatestVersion != "v1.1.0" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
