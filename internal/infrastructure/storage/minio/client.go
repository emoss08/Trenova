package minio

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ClientParams struct {
	fx.In

	Logger *logger.Logger
	Config *config.Manager
}

// Client wraps the Minio client with additional functionality
type Client struct {
	*minio.Client
	config *config.MinioConfig
	l      *zerolog.Logger

	// Connection pool
	transport *http.Transport
}

func NewClient(p ClientParams) (*Client, error) {
	log := p.Logger.With().Str("component", "storage.minio").Logger()
	cfg := p.Config.Minio()

	if err := validateConfig(cfg); err != nil {
		return nil, eris.Wrap(err, "invalid configuration")
	}

	// Set default configuration
	setDefaultConfig(cfg)

	// Configure transport
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.ConnectionTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		DisableCompression: true,
	}

	// Configure Minio client options
	opts := &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:    cfg.UseSSL,
		Region:    cfg.Region,
		Transport: transport,
	}

	// Create Minio client
	minioClient, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create minio client")
	}

	client := &Client{
		Client:    minioClient,
		config:    cfg,
		l:         &log,
		transport: transport,
	}

	// Test connection
	if err = client.ping(context.Background()); err != nil {
		return nil, eris.Wrap(err, "failed to ping minio server")
	}

	return client, nil
}

// ping verifies the connection to the Minio server
func (c *Client) ping(ctx context.Context) error {
	_, err := c.Client.ListBuckets(ctx)
	if err != nil {
		c.l.Error().Err(err).Msg("failed to ping minio server")
		return err
	}
	return nil
}

// WithRetry wraps an operation with retry logic
func (c *Client) WithRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	for i := 0; i < c.config.MaxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := operation(); err != nil {
				lastErr = err
				c.l.Warn().Err(err).Int("attempt", i+1).
					Int("maxRetries", c.config.MaxRetries).
					Msg("operation failed, retrying")
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return nil
		}
	}
	return eris.Wrap(lastErr, "operation failed after max retries")
}

// WithTimeout wraps an operation with a timeout
func (c *Client) WithTimeout(ctx context.Context, timeout time.Duration, operation func() error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- operation()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return eris.Wrap(services.ErrOperationTimeout, ctx.Err().Error())
	}
}

func validateConfig(cfg *config.MinioConfig) error {
	if cfg.Endpoint == "" {
		return eris.Wrap(services.ErrInvalidConfiguration, "endpoint is required")
	}
	if cfg.AccessKey == "" {
		return eris.Wrap(services.ErrInvalidConfiguration, "access key is required")
	}
	if cfg.SecretKey == "" {
		return eris.Wrap(services.ErrInvalidConfiguration, "secret key is required")
	}
	return nil
}

func setDefaultConfig(cfg *config.MinioConfig) {
	if cfg.ConnectionTimeout == 0 {
		cfg.ConnectionTimeout = 10 * time.Second
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 100
	}
	if cfg.MaxConnsPerHost == 0 {
		cfg.MaxConnsPerHost = 100
	}
	if cfg.IdleConnTimeout == 0 {
		cfg.IdleConnTimeout = 90 * time.Second
	}
}
