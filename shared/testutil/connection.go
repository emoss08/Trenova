package testutil

import (
	"context"

	"github.com/uptrace/bun"
)

type TestConnection struct {
	db *bun.DB
}

func NewTestConnection(db *bun.DB) *TestConnection {
	return &TestConnection{db: db}
}

func (c *TestConnection) DB() *bun.DB {
	return c.db
}

func (c *TestConnection) HealthCheck(ctx context.Context) error {
	if c.db == nil {
		return nil
	}
	return c.db.PingContext(ctx)
}

func (c *TestConnection) IsHealthy(ctx context.Context) bool {
	return c.HealthCheck(ctx) == nil
}

func (c *TestConnection) Close() error {
	return nil
}
