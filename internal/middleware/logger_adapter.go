package middleware

import (
	"context"

	"github.com/cry047/baseline/internal/platform/logger"
)

// ZapAdapter wraps a *logger.Logger so it implements LoggerAdapter.
type ZapAdapter struct {
	z *logger.Logger
}

// NewZapAdapter returns a ZapAdapter.
func NewZapAdapter(z *logger.Logger) *ZapAdapter {
	return &ZapAdapter{z: z}
}

// Info logs an info-level message.
func (a *ZapAdapter) Info(ctx context.Context, msg string) {
	if a.z != nil {
		a.z.Info(ctx, msg)
	}
}

// Warn logs a warn-level message.
func (a *ZapAdapter) Warn(ctx context.Context, msg string) {
	if a.z != nil {
		a.z.Warn(ctx, msg)
	}
}

// Error logs an error-level message.
func (a *ZapAdapter) Error(ctx context.Context, msg string, err error) {
	if a.z != nil {
		a.z.Error(ctx, msg, err)
	}
}

// Assert that *ZapAdapter implements LoggerAdapter.
var _ LoggerAdapter = (*ZapAdapter)(nil)
