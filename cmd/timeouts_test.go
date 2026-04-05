package cmd

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// TestHungCommandTimesOut verifies that a subprocess which never exits
// is killed when its context deadline is exceeded.
func TestHungCommandTimesOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep command not available on Windows")
	}

	// Create a context that expires well before the command would finish.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// "sleep 10" will run for 10 seconds -- far longer than our 100ms timeout.
	cmd := exec.CommandContext(ctx, "sleep", "10")

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	// The command must have failed.
	if err == nil {
		t.Fatal("expected hung command to return an error, got nil")
	}

	// The command must have been killed by the context timeout, not by
	// running to completion.  Allow generous slack for CI environments.
	if elapsed > 5*time.Second {
		t.Fatalf("command took %v, expected to be killed well before 5s", elapsed)
	}

	// Verify the error is related to context cancellation / deadline exceeded.
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", ctx.Err())
	}
}

// TestHungCommandKilled verifies that os.IsProcessExit reports the process
// was killed (signal-based termination on Unix).
func TestHungCommandKilled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal-based kill not reliable on Windows")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sleep", "10")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error from killed command")
	}

	// On Unix, a context-cancelled process is killed with SIGKILL.
	// exec.ExitError is returned; check that the process did not exit cleanly.
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	// A killed process should not have exit code 0.
	if exitErr.Success() {
		t.Error("killed process reported success")
	}
}

// TestCommandCompletesWithinTimeout verifies that a fast command completes
// successfully when given adequate time.
func TestCommandCompletesWithinTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep command not available on Windows")
	}

	ctx, cancel := context.WithTimeout(context.Background(), GeneralTimeout)
	defer cancel()

	// "sleep 0.01" finishes in ~10ms, well within GeneralTimeout.
	cmd := exec.CommandContext(ctx, "sleep", "0.01")

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected quick command to succeed, got: %v", err)
	}

	// Context should not be expired.
	if ctx.Err() != nil {
		t.Errorf("context should still be valid after quick command, got: %v", ctx.Err())
	}
}

// TestTimeoutConstantsAreReasonable verifies the timeout values are positive
// and ordered as documented.
func TestTimeoutConstantsAreReasonable(t *testing.T) {
	if GeneralTimeout <= 0 {
		t.Errorf("GeneralTimeout should be positive, got %v", GeneralTimeout)
	}
	if GitTimeout <= 0 {
		t.Errorf("GitTimeout should be positive, got %v", GitTimeout)
	}
	if BuildTimeout <= 0 {
		t.Errorf("BuildTimeout should be positive, got %v", BuildTimeout)
	}
	if GitTimeout < GeneralTimeout {
		t.Errorf("GitTimeout (%v) should be >= GeneralTimeout (%v)", GitTimeout, GeneralTimeout)
	}
	if BuildTimeout < GitTimeout {
		t.Errorf("BuildTimeout (%v) should be >= GitTimeout (%v)", BuildTimeout, GitTimeout)
	}
}
