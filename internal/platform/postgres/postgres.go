package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds PostgreSQL connection settings.
type Config struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	QueryTimeout      time.Duration
}

// Pool wraps a pgx pool to provide controlled I/O with timeouts.
type Pool struct {
	pool  *pgxpool.Pool
	query time.Duration
}

// New returns a connection pool using the given config. The pool is immediately
// ready for use after a successful Ping.
func New(ctx context.Context, cfg Config) (*Pool, error) {
	if cfg.URL == "" {
		return nil, errors.New("postgres URL is required")
	}
	pcfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres url: %w", err)
	}
	if cfg.MaxConns > 0 {
		pcfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		pcfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		pcfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		pcfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheckPeriod > 0 {
		pcfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	}
	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	q := cfg.QueryTimeout
	if q == 0 {
		q = 5 * time.Second
	}
	return &Pool{pool: pool, query: q}, nil
}

// Acquire returns a connection from the pool with a context-bound timeout.
func (p *Pool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, p.query)
	defer cancel()
	return p.pool.Acquire(ctx)
}

// BeginTx starts a new transaction with the default isolation level.
func (p *Pool) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	ctx, cancel := context.WithTimeout(ctx, p.query)
	defer cancel()
	return p.pool.BeginTx(ctx, opts)
}

// Exec executes a statement within a context-bound timeout.
func (p *Pool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := context.WithTimeout(ctx, p.query)
	defer cancel()
	return p.pool.Exec(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row.
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	ctx, cancel := context.WithTimeout(ctx, p.query)
	defer cancel()
	return p.pool.QueryRow(ctx, sql, args...)
}

// Query executes a query that returns multiple rows.
func (p *Pool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, p.query)
	defer cancel()
	return p.pool.Query(ctx, sql, args...)
}

// Close releases all pool resources.
func (p *Pool) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// Ping verifies the database is reachable.
func (p *Pool) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, p.query)
	defer cancel()
	return p.pool.Ping(ctx)
}
