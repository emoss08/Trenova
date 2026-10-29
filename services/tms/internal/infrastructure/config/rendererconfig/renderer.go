package rendererconfig

import (
	"strings"
	"time"
)

// RendererConfig governs HTML-to-PDF rendering and the limits applied to
// organization-authored templates.
//
// GotenbergURL points at a sidecar that owns the browser. It is deliberately not
// reachable from the host and is never proxied: it renders untrusted markup, so
// the only thing that should be able to reach it is this process.
type Config struct {
	GotenbergURL   string        `mapstructure:"gotenbergUrl"   validate:"omitempty,url"`
	RequestTimeout time.Duration `mapstructure:"requestTimeout"`
	HealthTimeout  time.Duration `mapstructure:"healthTimeout"`
	RenderTimeout  time.Duration `mapstructure:"renderTimeout"`
	MaxConcurrent  int           `mapstructure:"maxConcurrent"  validate:"min=0,max=32"`
	MaxRetries     int           `mapstructure:"maxRetries"     validate:"min=0,max=5"`
	MaxSourceBytes int64         `mapstructure:"maxSourceBytes" validate:"min=0"`
	MaxHTMLBytes   int64         `mapstructure:"maxHtmlBytes"   validate:"min=0"`
	MaxTextBytes   int64         `mapstructure:"maxTextBytes"   validate:"min=0"`
	MaxAssetBytes  int64         `mapstructure:"maxAssetBytes"  validate:"min=0"`
	MaxAssets      int           `mapstructure:"maxAssets"      validate:"min=0,max=64"`
	MaxAssetBudget int64         `mapstructure:"maxAssetBudget" validate:"min=0"`
	MaxPDFBytes    int64         `mapstructure:"maxPdfBytes"    validate:"min=0"`
}

// Enabled reports whether a rendering backend is configured. When it is not,
// PDF generation surfaces a business error instead of failing startup — the
// same treatment web push and search get.
func (c *Config) Enabled() bool { return c.GotenbergURL != "" }

func (c *Config) GetGotenbergURL() string {
	return strings.TrimSuffix(c.GotenbergURL, "/")
}

func (c *Config) GetRequestTimeout() time.Duration {
	if c.RequestTimeout == 0 {
		return 45 * time.Second
	}
	return c.RequestTimeout
}

func (c *Config) GetHealthTimeout() time.Duration {
	if c.HealthTimeout == 0 {
		return 5 * time.Second
	}
	return c.HealthTimeout
}

func (c *Config) GetRenderTimeout() time.Duration {
	if c.RenderTimeout == 0 {
		return 2 * time.Second
	}
	return c.RenderTimeout
}

func (c *Config) GetMaxConcurrent() int {
	if c.MaxConcurrent == 0 {
		return 4
	}
	return c.MaxConcurrent
}

func (c *Config) GetMaxRetries() int {
	if c.MaxRetries == 0 {
		return 3
	}
	return c.MaxRetries
}

func (c *Config) GetMaxSourceBytes() int64 {
	if c.MaxSourceBytes == 0 {
		return 256 << 10
	}
	return c.MaxSourceBytes
}

func (c *Config) GetMaxHTMLBytes() int64 {
	if c.MaxHTMLBytes == 0 {
		return 2 << 20
	}
	return c.MaxHTMLBytes
}

func (c *Config) GetMaxTextBytes() int64 {
	if c.MaxTextBytes == 0 {
		return 64 << 10
	}
	return c.MaxTextBytes
}

func (c *Config) GetMaxAssetBytes() int64 {
	if c.MaxAssetBytes == 0 {
		return 1 << 20
	}
	return c.MaxAssetBytes
}

func (c *Config) GetMaxAssets() int {
	if c.MaxAssets == 0 {
		return 10
	}
	return c.MaxAssets
}

func (c *Config) GetMaxAssetBudget() int64 {
	if c.MaxAssetBudget == 0 {
		return 4 << 20
	}
	return c.MaxAssetBudget
}

func (c *Config) GetMaxPDFBytes() int64 {
	if c.MaxPDFBytes == 0 {
		return 20 << 20
	}
	return c.MaxPDFBytes
}
