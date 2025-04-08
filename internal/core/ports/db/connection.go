package db

import (
	"context"

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
	DB(ctx context.Context) (*bun.DB, error)

	// ConnectionInfo returns information about the database connection
	ConnectionInfo() (*ConnectionInfo, error)

	// Close closes the database connection.
	Close() error
}
