// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package db

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type ConnectionInfo struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
}

// Connection is a wrapper around the bun.DB type that provides a way to get a database connection
// and close the connection.
type Connection interface {
	// DB returns a database connection.
	// For backward compatibility, this returns the primary (write) connection.
	DB(ctx context.Context) (*bun.DB, error)

	// ReadDB returns a read-only database connection.
	// If read replicas are not configured, this returns the primary connection.
	ReadDB(ctx context.Context) (*bun.DB, error)

	// WriteDB returns a write database connection.
	// This always returns the primary connection.
	WriteDB(ctx context.Context) (*bun.DB, error)

	// ConnectionInfo returns information about the database connection
	ConnectionInfo() (*ConnectionInfo, error)

	// SQLDB returns a database connection.
	SQLDB(ctx context.Context) (*sql.DB, error)

	// Close closes the database connection.
	Close() error
}
