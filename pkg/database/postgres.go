package database

import (
	"context"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration

	// Retry configuration
	MaxRetries    int
	RetryInterval time.Duration

	// Telemetry configuration
	EnableTracing bool
	ServiceName   string
}

// DefaultPostgresConfig returns default configuration
// Note: Password must be provided via environment variable
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "", // Must be provided via environment variable
		Database:        "booking_rush",
		SSLMode:         "disable",
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		ConnectTimeout:  10 * time.Second,
		MaxRetries:      3,
		RetryInterval:   2 * time.Second,
	}
}

// DSN returns the PostgreSQL connection string
func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// PostgresDB wraps pgxpool.Pool with additional functionality
type PostgresDB struct {
	pool   *pgxpool.Pool
	config *PostgresConfig
}

// NewPostgres creates a new PostgreSQL connection pool with retry logic
func NewPostgres(ctx context.Context, cfg *PostgresConfig) (*PostgresDB, error) {
	if cfg == nil {
		cfg = DefaultPostgresConfig()
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Apply pool configuration
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	// Enable OpenTelemetry tracing if configured
	if cfg.EnableTracing {
		opts := []otelpgx.Option{
			otelpgx.WithIncludeQueryParameters(),
		}
		poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(opts...)
	}

	// Connect with retry logic
	var pool *pgxpool.Pool
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(cfg.RetryInterval)
		}

		pool, lastErr = pgxpool.NewWithConfig(ctx, poolConfig)
		if lastErr != nil {
			continue
		}

		// Test the connection
		if lastErr = pool.Ping(ctx); lastErr != nil {
			pool.Close()
			continue
		}

		// Successfully connected
		return &PostgresDB{
			pool:   pool,
			config: cfg,
		}, nil
	}

	return nil, fmt.Errorf("failed to connect to postgres after %d attempts: %w", cfg.MaxRetries+1, lastErr)
}

// Pool returns the underlying pgxpool.Pool
func (db *PostgresDB) Pool() *pgxpool.Pool {
	return db.pool
}

// Ping checks if the database connection is alive
func (db *PostgresDB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close closes all connections in the pool gracefully
func (db *PostgresDB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// Stats returns connection pool statistics
func (db *PostgresDB) Stats() *pgxpool.Stat {
	return db.pool.Stat()
}

// HealthCheck performs a health check on the database
func (db *PostgresDB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var result int
	err := db.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database health check returned unexpected result: %d", result)
	}

	return nil
}

// Exec executes a query without returning any rows
func (db *PostgresDB) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := db.pool.Exec(ctx, sql, args...)
	return err
}

// QueryRow executes a query that returns at most one row
func (db *PostgresDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// BeginTx starts a new transaction
func (db *PostgresDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

// IsConnected returns true if the database connection is alive
func (db *PostgresDB) IsConnected(ctx context.Context) bool {
	return db.Ping(ctx) == nil
}

// Acquire acquires a connection from the pool
func (db *PostgresDB) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	return db.pool.Acquire(ctx)
}
