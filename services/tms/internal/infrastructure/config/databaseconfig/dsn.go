package databaseconfig

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func (c *Config) DSN(appName, password string) string {
	return c.dsnAt(appName, c.Host, c.Port, c.User, password)
}

func (c *Config) DSNMasked() string {
	return fmt.Sprintf(
		"postgres://%s:****@%s/%s?sslmode=%s",
		c.User,
		net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		c.Name,
		c.SSLMode,
	)
}

func (c *Config) DSNForRole(appName string, role Role) string {
	if !role.Configured() {
		return c.DSN(appName, c.Password)
	}

	return c.dsnAt(appName, c.Host, c.Port, role.User, role.Password)
}

func (c *Config) MigrationDSN(appName string) string {
	return c.DSNForRole(appName, c.Migrator)
}

func (c *Config) CacheListenDSN(appName string) string {
	host := strings.TrimSpace(c.Cache.ListenHost)
	if host == "" {
		return ""
	}

	port := c.Cache.ListenPort
	if port <= 0 {
		port = c.Port
	}

	return c.dsnAt(appName, host, port, c.User, c.Password)
}

func (c *Config) dsnAt(appName, host string, port int, user, password string) string {
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   "/" + c.Name,
		RawQuery: "sslmode=" + url.QueryEscape(c.SSLMode) +
			"&application_name=" + url.QueryEscape(appName) +
			"&connect_timeout=10",
	}

	return dsn.String()
}
