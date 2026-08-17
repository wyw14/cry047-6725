package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/config"
	"github.com/cry047/baseline/internal/domain"
	"github.com/cry047/baseline/internal/middleware"
	"github.com/cry047/baseline/internal/platform/logger"
	"github.com/cry047/baseline/internal/repository/memory"
	"github.com/cry047/baseline/internal/service/notifier"
	"github.com/cry047/baseline/internal/service/scheduler"
	"github.com/cry047/baseline/internal/service/storage"
	phttp "github.com/cry047/baseline/internal/transport/http"
	"github.com/cry047/baseline/scripts/seed"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := flag.String("config", "", "path to a config file (optional)")
	flag.Parse()
	_ = cfgPath

	cfg := config.FromEnv()

	log, err := logger.New(logger.Config{
		Level:      cfg.LogLevel,
		Encoding:   cfg.LogEncoding,
		OutputPath: cfg.LogOutputPath,
		AddCaller:  false,
	})
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync()
	log.Info(context.Background(), "starting", zap.String("config", cfg.String()))

	// Compose ports.
	ports, err := buildPorts(cfg)
	if err != nil {
		return fmt.Errorf("build ports: %w", err)
	}
	// Seed demo data into the in-memory store if requested.
	if cfg.SeedOnStart {
		actor := domain.Actor{ID: "system", Name: "系统", Role: domain.RoleAdmin}
		res, err := seed.Run(context.Background(), ports, actor)
		if err != nil {
			log.Warn(context.Background(), "seed partial", zap.String("err", err.Error()))
		} else {
			log.Info(context.Background(), "seed done",
				zap.Int("places", res.Places),
				zap.Int("facilities", res.Facilities),
				zap.Int("templates", res.Templates),
				zap.Int("plans", res.Plans))
		}
	}

	// Compose application services.
	services := &phttp.Services{
		Places:        application.NewPlaceService(ports),
		People:        application.NewPersonService(ports),
		Facilities:    application.NewFacilityService(ports),
		Plans:         application.NewPlanService(ports),
		Executions:    application.NewExecutionService(ports),
		Anomalies:     application.NewAnomalyService(ports),
		Todos:         application.NewTodoService(ports),
		Notifications: application.NewNotificationService(ports),
		Audit:         application.NewAuditService(ports),
		Timeline:      application.NewTimelineService(ports),
		Scheduler:     application.NewSchedulerService(ports, application.WithTickInterval(cfg.SchedulerTick)),
	}

	adapter := middleware.NewZapAdapter(log)
	srv, err := phttp.New(services, phttp.Config{
		Addr:            cfg.HTTPAddr,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		IdleTimeout:     cfg.IdleTimeout,
		ShutdownTimeout: cfg.ShutdownTimeout,
		Mode:            cfg.HTTPMode,
		AllowedOrigins:  cfg.AllowedOrigins,
	}, adapter)
	if err != nil {
		return fmt.Errorf("init http server: %w", err)
	}

	// Start server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		log.Info(context.Background(), "http listen", zap.String("addr", cfg.HTTPAddr))
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	// Wait for termination signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		log.Info(context.Background(), "received signal", zap.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	// Graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error(ctx, "graceful shutdown failed", err)
	}
	<-errCh
	log.Info(ctx, "bye")
	return nil
}

// buildPorts constructs a fully-wired domain.Ports. If cfg.UseMemoryStore is
// true (the default for offline runs and tests), the in-memory repository is
// used. Otherwise a PostgreSQL-backed pool is constructed.
func buildPorts(cfg config.Config) (*domain.Ports, error) {
	memStore := memory.NewStore()
	ports := memory.NewPorts(memStore)
	// Attach offline adapters.
	notifierAdapter := notifier.New(cfg.NotifierRingSize, cfg.NotifierLogPath)
	ports.Notifier = notifierAdapter
	storageAdapter, err := storage.New(storage.Config{
		RootDir:        cfg.StorageRoot,
		MaxSizeBytes:   cfg.StorageMaxSize,
		AllowedTypes:   cfg.StorageAllowedTypes,
		AllowedExts:    cfg.StorageAllowedExts,
		ForbidSymlinks: true,
	})
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}
	ports.Storage = storageAdapter
	schedAdapter := scheduler.New(
		scheduler.WithTickInterval(cfg.SchedulerTick),
		scheduler.WithErrorHandler(func(name string, err error) {
			_ = name
			_ = err
		}),
	)
	if err := schedAdapter.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("start scheduler: %w", err)
	}
	ports.Scheduler = schedAdapter
	ports.Clock = domain.SystemClock{}
	return &ports, nil
}

// silence unused import warning for time.
var _ = time.Second
