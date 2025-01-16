package db

import (
	"context"

	"github.com/uptrace/bun"
)

// Connection is a wrapper around the bun.DB type that provides a way to get a database connection
// and close the connection.
type Connection interface {
	// DB returns a database connection.
	DB(ctx context.Context) (*bun.DB, error)

	// Close closes the database connection.
	Close() error
}
