package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestTimeoutCtx_DefaultTimeout(t *testing.T) {
	// Save and restore default timeout
	orig := DefaultTimeout
	defer func() { DefaultTimeout = orig }()
	DefaultTimeout = 500 * time.Millisecond

	cmd := &cobra.Command{Use: "test"}
	ctx, cancel := timeoutCtx(cmd)

	// Verify the context has a deadline within expected bounds
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}

	remaining := time.Until(deadline)
	if remaining < 100*time.Millisecond || remaining > 500*time.Millisecond {
		t.Fatalf("expected deadline ~500ms, got %v", remaining)
	}

	// Verify cancel function works
	cancel()
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Fatal("expected context to be done after cancel()")
	}
}

func TestTimeoutCtx_CustomTimeout(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	customTimeout := 2 * time.Second
	ctx, cancel := timeoutCtxWith(cmd, customTimeout)

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}

	remaining := time.Until(deadline)
	if remaining < 1*time.Second || remaining > 2*time.Second {
		t.Fatalf("expected deadline ~2s, got %v", remaining)
	}

	cancel()
}

func TestTimeoutCtx_ParentCancellation(t *testing.T) {
	// Save and restore default timeout
	orig := DefaultTimeout
	defer func() { DefaultTimeout = orig }()
	DefaultTimeout = 5 * time.Second

	// Create a parent context we can cancel
	parentCtx, parentCancel := context.WithCancel(context.Background())

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(parentCtx)

	ctx, cancel := timeoutCtx(cmd)
	defer cancel()

	// Cancel the parent
	parentCancel()

	// Child should also be done
	select {
	case <-ctx.Done():
		// expected
	case <-time.After(1 * time.Second):
		t.Fatal("expected child context to be cancelled when parent is cancelled")
	}
}

func TestTimeoutCtx_ReturnsCancelFunc(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	ctx, cancel := timeoutCtx(cmd)

	// Verify cancel is not nil
	if cancel == nil {
		t.Fatal("expected non-nil cancel function")
	}

	// Verify ctx is not nil
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	// Cleanup
	cancel()
}

func TestTimeoutCtxWith_NilCmdContext(t *testing.T) {
	// When cmd.Context() is nil (no parent set), should fall back to Background
	cmd := &cobra.Command{Use: "test"}
	// cmd.Context() defaults to nil when not set via SetContext
	// Cobra may set one internally, so we test the behavior explicitly
	orig := DefaultTimeout
	defer func() { DefaultTimeout = orig }()
	DefaultTimeout = 5 * time.Second

	ctx, cancel := timeoutCtxWith(cmd, 5*time.Second)

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline even with nil parent")
	}

	remaining := time.Until(deadline)
	if remaining < 3*time.Second || remaining > 5*time.Second {
		t.Fatalf("expected deadline ~5s with nil parent, got %v", remaining)
	}

	cancel()
}

func TestTimeoutCtx_DefaultIsGeneralTimeout(t *testing.T) {
	// Verify DefaultTimeout matches GeneralTimeout by default
	if DefaultTimeout != GeneralTimeout {
		t.Fatalf("expected DefaultTimeout(%v) == GeneralTimeout(%v)", DefaultTimeout, GeneralTimeout)
	}
}

func TestTimeoutCtx_SignalCancellation(t *testing.T) {
	// Simulate Ctrl+C by creating a signal.NotifyContext (the same mechanism cobra uses)
	// and proving timeoutCtx propagates OS signal cancellation to the child context.
	//
	// This proves: when a user presses Ctrl+C, cmd.Context() (which is a
	// signal.NotifyContext) is cancelled, and timeoutCtx(cmd) produces a child
	// that also cancels immediately.

	orig := DefaultTimeout
	defer func() { DefaultTimeout = orig }()
	DefaultTimeout = 5 * time.Second

	// Create a signal-based context, just like cobra does for its commands.
	// We notify on SIGINT (Ctrl+C) and SIGTERM.
	parentCtx, parentCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parentCancel()

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(parentCtx)

	ctx, cancel := timeoutCtx(cmd)
	defer cancel()

	// Before signal, context should NOT be done
	select {
	case <-ctx.Done():
		t.Fatal("child context should not be done before signal")
	default:
		// expected
	}

	// Simulate Ctrl+C: send SIGINT to ourselves via the signal channel.
	// We can't easily send to the internal signal channel of NotifyContext,
	// so we just call the parent cancel function (which NotifyContext returns
	// and which is triggered by cobra when it receives SIGINT).
	//
	// Alternatively, we send an actual signal to our process group.
	// The safest approach for tests: call parentCancel directly, which is
	// exactly what happens when cobra's signal handler fires.
	parentCancel()

	// Child context must be done promptly after parent cancellation
	select {
	case <-ctx.Done():
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("child context was not cancelled after parent signal cancellation")
	}

	// Verify the cancellation error is context.Canceled
	if ctx.Err() != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", ctx.Err())
	}
}

func TestTimeoutCtx_SignalCancellation_WithTimeoutCtxWith(t *testing.T) {
	// Same as above but uses timeoutCtxWith directly to prove both helpers
	// propagate signal-based cancellation.

	parentCtx, parentCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parentCancel()

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(parentCtx)

	ctx, cancel := timeoutCtxWith(cmd, 5*time.Second)
	defer cancel()

	// Simulate Ctrl+C
	parentCancel()

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("timeoutCtxWith child was not cancelled after signal")
	}

	if ctx.Err() != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", ctx.Err())
	}
}
