// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package config

import (
	"fmt"
	"time"
)

type Database struct {
	// Host is the hostname or IP address of the database server
	Host string

	// Port is the port to connect to the database on
	Port int

	// Username is the username to use when connecting to the database
	Username string

	// Password is the password to use when connecting to the database
	Password string `json:"-"` // sensitive

	// Database is the name of the database to connect to
	Database string

	// AdditionalParams are additional connection parameters to be passed to the database
	AdditionalParams map[string]string `json:",omitempty"` // Optional additional connection parameters mapped into the connection string

	// MaxOpenConns is the maximum number of open connections to the database.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of connections in the idle connection pool.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	ConnMaxLifetime time.Duration

	// Debug and VerboseLogging are used to enable verbose logging for the database connection
	Debug          bool
	VerboseLogging bool
}

func (c Database) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", c.Username, c.Password, c.Host, c.Port, c.Database)
}
