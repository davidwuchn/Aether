package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

// DefaultTimeout is the default timeout for timeoutCtx.
// Exported as a var (not const) to allow test overrides.
var DefaultTimeout = GeneralTimeout

// timeoutCtx derives a timeout context from cmd.Context().
// If cmd.Context() returns nil, it falls back to context.Background().
// The returned cancel function must be called to release resources.
func timeoutCtx(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	return timeoutCtxWith(cmd, DefaultTimeout)
}

// timeoutCtxWith derives a timeout context from cmd.Context() with a custom duration.
// If cmd.Context() returns nil, it falls back to context.Background().
// The returned cancel function must be called to release resources.
func timeoutCtxWith(cmd *cobra.Command, timeout time.Duration) (context.Context, context.CancelFunc) {
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}
