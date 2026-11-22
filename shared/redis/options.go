package redis

import (
	"crypto/tls"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultAddr            = "localhost:6379"
	defaultDialTimeout     = 5 * time.Second
	defaultReadTimeout     = 5 * time.Second
	defaultConnMaxIdleTime = 30 * time.Minute
	defaultMaxRetries      = 3
	defaultMinRetryBackoff = 10 * time.Millisecond
	defaultMaxRetryBackoff = time.Second
	defaultPoolTimeout     = 30 * time.Second
)

type Options struct {
	Addr            string
	Username        string
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	PoolTimeout     time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
	TLSConfig       *tls.Config
	TracerProvider  trace.TracerProvider
	MeterProvider   metric.MeterProvider
}

func (opt *Options) init() {
	if opt.Addr == "" {
		opt.Addr = defaultAddr
	}
	opt.DialTimeout = timeoutOrDefault(opt.DialTimeout, defaultDialTimeout)
	if opt.PoolSize == 0 {
		opt.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}
	opt.ReadTimeout = timeoutOrDefault(opt.ReadTimeout, defaultReadTimeout)
	opt.WriteTimeout = timeoutOrDefault(opt.WriteTimeout, opt.ReadTimeout)
	if opt.PoolTimeout == 0 {
		if opt.ReadTimeout > 0 {
			opt.PoolTimeout = opt.ReadTimeout + time.Second
		} else {
			opt.PoolTimeout = defaultPoolTimeout
		}
	}
	opt.ConnMaxIdleTime = timeoutOrDefault(opt.ConnMaxIdleTime, defaultConnMaxIdleTime)
	switch opt.MaxRetries {
	case -1:
		opt.MaxRetries = 0
	case 0:
		opt.MaxRetries = defaultMaxRetries
	}
	opt.MinRetryBackoff = timeoutOrDefault(opt.MinRetryBackoff, defaultMinRetryBackoff)
	opt.MaxRetryBackoff = timeoutOrDefault(opt.MaxRetryBackoff, defaultMaxRetryBackoff)
}

func timeoutOrDefault(d, fallback time.Duration) time.Duration {
	switch {
	case d == 0:
		return fallback
	case d < 0:
		return 0
	default:
		return d
	}
}
