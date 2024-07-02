package config

import (
	"fmt"
	"time"
)

type Database struct {
	Host             string
	Port             int
	Username         string
	Password         string `json:"-"` // sensitive
	Database         string
	AdditionalParams map[string]string `json:",omitempty"` // Optional additional connection parameters mapped into the connection string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
}

func (c Database) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", c.Username, c.Password, c.Host, c.Port, c.Database)
}
