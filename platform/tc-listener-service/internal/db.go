// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
