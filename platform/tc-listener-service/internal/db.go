package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq" // db dependency
)

// InitDB initializes a connection to the PostgreSQL database using the connection string
// specified in the environment variable "DB_CONNECTION_STRING".
//
// Parameters:
//
//	ctx - context for managing the lifecycle of the database connection
//
// Returns:
//
//	*sql.DB - a new database connection
//	error - an error if the connection initialization fails
func InitDB(ctx context.Context) (*sql.DB, error) {
	dbConnectionString := EnvVar("DB_CONNECTION_STRING")
	if dbConnectionString == "" {
		return nil, errors.New("DB_CONNECTION_STRING environment variable is not set")
	}

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
