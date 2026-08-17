package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds every external knob the application reads.
type Config struct {
	HTTPAddr        string
	HTTPMode        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	LogLevel      string
	LogEncoding   string
	LogOutputPath string

	DatabaseURL         string
	DBMaxConns          int32
	DBMinConns          int32
	DBMaxConnLifetime   time.Duration
	DBMaxConnIdleTime   time.Duration
	DBHealthCheckPeriod time.Duration
	DBQueryTimeout      time.Duration

	UseMemoryStore bool

	StorageRoot         string
	StorageMaxSize      int64
	StorageAllowedTypes []string
	StorageAllowedExts  []string

	NotifierRingSize int
	NotifierLogPath  string

	SchedulerTick time.Duration

	AllowedOrigins []string

	SeedOnStart bool
}

// Default returns a Config populated with sane defaults.
func Default() Config {
	return Config{
		HTTPAddr:            ":8080",
		HTTPMode:            "release",
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         60 * time.Second,
		ShutdownTimeout:     10 * time.Second,
		LogLevel:            "info",
		LogEncoding:         "json",
		LogOutputPath:       "stderr",
		UseMemoryStore:      true,
		StorageRoot:         "./var/attachments",
		StorageMaxSize:      8 * 1024 * 1024,
		StorageAllowedTypes: []string{"image/png", "image/jpeg", "image/webp", "image/gif", "application/pdf", "text/plain"},
		StorageAllowedExts:  []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf", ".txt"},
		NotifierRingSize:    256,
		SchedulerTick:       time.Minute,
		AllowedOrigins:      []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		SeedOnStart:         true,
	}
}

// FromEnv reads environment variables and returns a Config.
func FromEnv() Config {
	c := Default()
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		c.HTTPAddr = v
	}
	if v := os.Getenv("HTTP_MODE"); v != "" {
		c.HTTPMode = v
	}
	if v := os.Getenv("HTTP_READ_TIMEOUT"); v != "" {
		c.ReadTimeout = parseDuration(v, c.ReadTimeout)
	}
	if v := os.Getenv("HTTP_WRITE_TIMEOUT"); v != "" {
		c.WriteTimeout = parseDuration(v, c.WriteTimeout)
	}
	if v := os.Getenv("HTTP_IDLE_TIMEOUT"); v != "" {
		c.IdleTimeout = parseDuration(v, c.IdleTimeout)
	}
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		c.ShutdownTimeout = parseDuration(v, c.ShutdownTimeout)
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("LOG_ENCODING"); v != "" {
		c.LogEncoding = v
	}
	if v := os.Getenv("LOG_OUTPUT_PATH"); v != "" {
		c.LogOutputPath = v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		c.DatabaseURL = v
		c.UseMemoryStore = false
	}
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		c.DBMaxConns = parseInt32(v, c.DBMaxConns)
	}
	if v := os.Getenv("DB_MIN_CONNS"); v != "" {
		c.DBMinConns = parseInt32(v, c.DBMinConns)
	}
	if v := os.Getenv("DB_MAX_CONN_LIFETIME"); v != "" {
		c.DBMaxConnLifetime = parseDuration(v, c.DBMaxConnLifetime)
	}
	if v := os.Getenv("DB_MAX_CONN_IDLE_TIME"); v != "" {
		c.DBMaxConnIdleTime = parseDuration(v, c.DBMaxConnIdleTime)
	}
	if v := os.Getenv("DB_HEALTH_CHECK_PERIOD"); v != "" {
		c.DBHealthCheckPeriod = parseDuration(v, c.DBHealthCheckPeriod)
	}
	if v := os.Getenv("DB_QUERY_TIMEOUT"); v != "" {
		c.DBQueryTimeout = parseDuration(v, c.DBQueryTimeout)
	}
	if v := os.Getenv("USE_MEMORY_STORE"); v != "" {
		c.UseMemoryStore = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("STORAGE_ROOT"); v != "" {
		c.StorageRoot = v
	}
	if v := os.Getenv("STORAGE_MAX_SIZE"); v != "" {
		c.StorageMaxSize = parseInt64(v, c.StorageMaxSize)
	}
	if v := os.Getenv("NOTIFIER_RING_SIZE"); v != "" {
		c.NotifierRingSize = parseInt(v, c.NotifierRingSize)
	}
	if v := os.Getenv("NOTIFIER_LOG_PATH"); v != "" {
		c.NotifierLogPath = v
	}
	if v := os.Getenv("SCHEDULER_TICK"); v != "" {
		c.SchedulerTick = parseDuration(v, c.SchedulerTick)
	}
	if v := os.Getenv("ALLOWED_ORIGINS"); v != "" {
		c.AllowedOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("SEED_ON_START"); v != "" {
		c.SeedOnStart = strings.EqualFold(v, "true") || v == "1"
	}
	return c
}

// parseDuration parses a duration string with a fallback default.
func parseDuration(v string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func parseInt(v string, def int) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func parseInt32(v string, def int32) int32 {
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return int32(n)
}

func parseInt64(v string, def int64) int64 {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// String returns a debug-friendly representation.
func (c Config) String() string {
	return fmt.Sprintf("Config{HTTPAddr:%s Mode:%s UseMemory:%v StorageRoot:%s Seed:%v}",
		c.HTTPAddr, c.HTTPMode, c.UseMemoryStore, c.StorageRoot, c.SeedOnStart)
}
