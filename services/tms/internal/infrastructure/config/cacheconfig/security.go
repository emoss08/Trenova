package cacheconfig

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
)

var (
	ErrTLSClientCertificatePair = errors.New(
		"cache.tls.certFile and cache.tls.keyFile must be set together",
	)
	ErrTLSSettingsWithoutTLS = errors.New(
		"cache.tls.serverName, caFile, certFile and keyFile need cache.tls.enabled",
	)
	ErrProductionRedisPasswordRequired = errors.New(
		"production and staging require cache.password, a real secret for Redis authentication",
	)
)

func (c *Config) Validate() error {
	t := c.TLS
	if !t.Enabled && (t.ServerName != "" || t.CAFile != "" || t.CertFile != "" || t.KeyFile != "") {
		return ErrTLSSettingsWithoutTLS
	}
	if (t.CertFile == "") != (t.KeyFile == "") {
		return ErrTLSClientCertificatePair
	}

	return nil
}

func (c *Config) ValidateProduction() error {
	password := strings.ToLower(strings.TrimSpace(c.Password))
	if password == "" {
		return ErrProductionRedisPasswordRequired
	}
	for _, placeholder := range securityconfig.InsecureDefaultValues {
		if strings.Contains(password, placeholder) {
			return ErrProductionRedisPasswordRequired
		}
	}

	return nil
}

func (c *Config) TLSClientConfig() (*tls.Config, error) {
	serverName := c.TLS.ServerName
	if serverName == "" {
		serverName = c.Host
	}
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
	}

	if c.TLS.CAFile != "" {
		pem, err := os.ReadFile(c.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read cache.tls.caFile: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("cache.tls.caFile %q holds no PEM certificate", c.TLS.CAFile)
		}
		cfg.RootCAs = pool
	}

	if c.TLS.CertFile != "" {
		pair, err := tls.LoadX509KeyPair(c.TLS.CertFile, c.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load cache.tls client certificate: %w", err)
		}
		cfg.Certificates = []tls.Certificate{pair}
	}

	return cfg, nil
}
