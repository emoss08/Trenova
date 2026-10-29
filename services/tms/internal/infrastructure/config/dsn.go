package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
)

func (c *Config) GetDSN(password string) string {
	return c.postgresDSN(c.Database.User, password)
}

func (c *Config) postgresDSN(user, password string) string {
	return c.postgresDSNAt(c.Database.Host, c.Database.Port, user, password)
}

func (c *Config) postgresDSNAt(host string, port int, user, password string) string {
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   "/" + c.Database.Name,
		RawQuery: "sslmode=" + url.QueryEscape(c.Database.SSLMode) +
			"&application_name=" + url.QueryEscape(c.App.Name) +
			"&connect_timeout=10",
	}

	return dsn.String()
}

func (c *Config) GetDSNMasked() string {
	return fmt.Sprintf(
		"postgres://%s:****@%s/%s?sslmode=%s",
		c.Database.User,
		net.JoinHostPort(c.Database.Host, strconv.Itoa(c.Database.Port)),
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func (c *Config) GetDSNForRole(role databaseconfig.Role) string {
	if !role.Configured() {
		return c.GetDSN(c.Database.Password)
	}

	return c.postgresDSN(role.User, role.Password)
}

func (c *Config) GetMigrationDSN() string {
	return c.GetDSNForRole(c.Database.Migrator)
}

func (c *Config) GetCacheListenDSN() string {
	host := strings.TrimSpace(c.Database.Cache.ListenHost)
	if host == "" {
		return ""
	}

	port := c.Database.Cache.ListenPort
	if port <= 0 {
		port = c.Database.Port
	}

	return c.postgresDSNAt(host, port, c.Database.User, c.Database.Password)
}
