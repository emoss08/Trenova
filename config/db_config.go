package config

import (
	"fmt"
	"sort"
	"strings"
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

// ConnectionString generates a connection string to be passed to sql.Open or equivalents, assuming Postgres syntax
func (c Database) ConnectionString() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", c.Host, c.Port, c.Username, c.Password, c.Database))

	if _, ok := c.AdditionalParams["sslmode"]; !ok {
		b.WriteString(" sslmode=disable")
	}

	if len(c.AdditionalParams) > 0 {
		params := make([]string, 0, len(c.AdditionalParams))
		for param := range c.AdditionalParams {
			params = append(params, param)
		}

		sort.Strings(params)

		for _, param := range params {
			fmt.Fprintf(&b, " %s=%s", param, c.AdditionalParams[param])
		}
	}

	return b.String()
}

func (c Database) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", c.Username, c.Password, c.Host, c.Port, c.Database)
}
