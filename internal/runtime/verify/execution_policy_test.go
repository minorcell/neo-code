package verify

import (
	"context"
	"errors"
	"testing"

	"neo-code/internal/config"
)

func TestPolicyCommandExecutorAllowsWhitelistedCommand(t *testing.T) {
	t.Parallel()

	executor := PolicyCommandExecutor{}
	policy := config.StaticDefaults().Runtime.Verification.ExecutionPolicy
	result, err := executor.Execute(context.Background(), CommandExecutionRequest{
		Command: "go version",
		Policy:  policy,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want 0", result.ExitCode)
	}
}

func TestPolicyCommandExecutorRejectsDeniedCommand(t *testing.T) {
	t.Parallel()

	executor := PolicyCommandExecutor{}
	policy := config.StaticDefaults().Runtime.Verification.ExecutionPolicy
	_, err := executor.Execute(context.Background(), CommandExecutionRequest{
		Command: "rm -rf .",
		Policy:  policy,
	})
	if err == nil {
		t.Fatal("expected denied command error")
	}
	if !errors.Is(err, ErrVerificationExecutionDenied) {
		t.Fatalf("error = %v, want ErrVerificationExecutionDenied", err)
	}
}

func TestPolicyCommandExecutorRejectsGitWriteSubcommand(t *testing.T) {
	t.Parallel()

	executor := PolicyCommandExecutor{}
	policy := config.StaticDefaults().Runtime.Verification.ExecutionPolicy
	_, err := executor.Execute(context.Background(), CommandExecutionRequest{
		Command: "git checkout .",
		Policy:  policy,
	})
	if err == nil {
		t.Fatal("expected git write command to be denied")
	}
	if !errors.Is(err, ErrVerificationExecutionDenied) {
		t.Fatalf("error = %v, want ErrVerificationExecutionDenied", err)
	}
}
