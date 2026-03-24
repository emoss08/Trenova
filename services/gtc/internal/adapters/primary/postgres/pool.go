package postgres

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewPool(ctx context.Context, databaseURL string, logger *zap.Logger) (*pgxpool.Pool, error) {
	normalizedURL, err := normalizeDatabaseURL(databaseURL)
	if err != nil {
		return nil, err
	}

	cfg, err := pgxpool.ParseConfig(normalizedURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres pool: %w", err)
	}

	logger.Info("connected to postgres snapshot/checkpoint pool")
	return pool, nil
}

func normalizeDatabaseURL(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse database url: %w", err)
	}

	query := parsed.Query()
	query.Del("replication")
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}
