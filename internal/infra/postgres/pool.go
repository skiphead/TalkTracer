package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skiphead/go-letopis/internal/infra/config"
)

func NewPool(ctx context.Context, cfg config.DBConfig, logger *slog.Logger) (*pgxpool.Pool, error) {
	logger.Info("connecting to database",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.DBName,
		"sslmode", cfg.SSLMode,
	)

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(cfg.User),
		url.QueryEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection config: %w", err)
	}

	// Настройка пула из конфига
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	// Дополнительные настройки
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second
	poolConfig.ConnConfig.RuntimeParams = map[string]string{
		"application_name": "go-letopis",
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверка соединения с таймаутом
	pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection pool created",
		"max_conns", poolConfig.MaxConns,
		"min_conns", poolConfig.MinConns,
	)

	return pool, nil
}
