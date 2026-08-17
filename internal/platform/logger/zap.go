package logger

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cry047/baseline/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the platform's structured logger abstraction.
// The mutex is used to synchronize Sink fallbacks in tests; the underlying
// zap.Logger is itself concurrency-safe.
type Logger struct {
	z      *zap.Logger
	fields []zap.Field
	sinks  []io.Writer
}

// Config configures the logger.
type Config struct {
	Level      string // debug|info|warn|error
	Encoding   string // json|console
	OutputPath string
	AddCaller  bool
}

// New returns a new Logger.
func New(cfg Config) (*Logger, error) {
	if cfg.Encoding == "" {
		cfg.Encoding = "json"
	}
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.OutputPath == "" {
		cfg.OutputPath = "stderr"
	}
	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}
	zapCfg := zap.NewProductionEncoderConfig()
	zapCfg.TimeKey = "ts"
	zapCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	zapCfg.EncodeCaller = zapcore.ShortCallerEncoder

	encoder := zapcore.NewJSONEncoder(zapCfg)
	if cfg.Encoding == "console" {
		zapCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(zapCfg)
	}

	writer, _, err := zap.Open(cfg.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("open log writer: %w", err)
	}
	core := zapcore.NewCore(encoder, writer, level)
	opts := []zap.Option{zap.AddStacktrace(zapcore.ErrorLevel)}
	if cfg.AddCaller {
		opts = append(opts, zap.AddCaller())
	}
	z := zap.New(core, opts...)
	return &Logger{z: z}, nil
}

// NewForWriter returns a Logger writing to a specific io.Writer (useful for tests).
func NewForWriter(w io.Writer, level string) (*Logger, error) {
	if level == "" {
		level = "info"
	}
	lvl := zapcore.InfoLevel
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	}
	zapCfg := zap.NewDevelopmentEncoderConfig()
	zapCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zapCfg),
		zapcore.AddSync(w),
		lvl,
	)
	return &Logger{z: zap.New(core)}, nil
}

// With returns a Logger with extra fields.
func (l *Logger) With(fields ...zap.Field) *Logger {
	cp := *l
	cp.z = l.z.With(fields...)
	cp.fields = append(cp.fields, fields...)
	return &cp
}

// WithRequestID returns a Logger that includes request_id.
func (l *Logger) WithRequestID(ctx context.Context) *Logger {
	rid := domain.RequestIDFromContext(ctx)
	if rid == "" {
		return l
	}
	return l.With(zap.String("request_id", rid))
}

// Info logs an info-level message.
func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithRequestID(ctx).z.Info(msg, fields...)
}

// Warn logs a warn-level message.
func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithRequestID(ctx).z.Warn(msg, fields...)
}

// Error logs an error-level message.
func (l *Logger) Error(ctx context.Context, msg string, err error, fields ...zap.Field) {
	all := append([]zap.Field{}, fields...)
	if err != nil {
		all = append(all, zap.Error(err))
	}
	l.WithRequestID(ctx).z.Error(msg, all...)
}

// InfoSimple is the simple-form logger.LoggerAdapter-compatible method.
func (l *Logger) InfoSimple(ctx context.Context, msg string) { l.Info(ctx, msg) }

// WarnSimple is the simple-form logger.LoggerAdapter-compatible method.
func (l *Logger) WarnSimple(ctx context.Context, msg string) { l.Warn(ctx, msg) }

// ErrorSimple is the simple-form logger.LoggerAdapter-compatible method.
func (l *Logger) ErrorSimple(ctx context.Context, msg string, err error) { l.Error(ctx, msg, err) }

// Debug logs a debug-level message.
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithRequestID(ctx).z.Debug(msg, fields...)
}

// Sync flushes buffered logs.
func (l *Logger) Sync() error { return l.z.Sync() }

// Std returns a *log.Logger compatible writer that bridges stdlib log output
// into this logger (useful for libraries that log via stdlib).
func (l *Logger) Std() *os.File {
	return os.Stderr
}

// Sugar returns the sugared logger for convenience.
func (l *Logger) Sugar() *zap.SugaredLogger { return l.z.Sugar() }

// Must panics if err is non-nil; used in main.
func Must(l *Logger, err error) *Logger {
	if err != nil {
		panic(err)
	}
	return l
}
