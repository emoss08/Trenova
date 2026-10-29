package documentconversionconfig

import (
	"strings"
	"time"
)

type Config struct {
	DoclingURL        string        `mapstructure:"doclingUrl"        validate:"omitempty,url"`
	APIKey            string        `mapstructure:"apiKey"`
	RequestTimeout    time.Duration `mapstructure:"requestTimeout"`
	ConversionTimeout time.Duration `mapstructure:"conversionTimeout"`
	HealthTimeout     time.Duration `mapstructure:"healthTimeout"`
	MaxConcurrent     int           `mapstructure:"maxConcurrent"     validate:"min=0,max=32"`
	MaxRetries        int           `mapstructure:"maxRetries"        validate:"min=0,max=5"`
	MaxFileBytes      int64         `mapstructure:"maxFileBytes"      validate:"min=0"`
	MaxPages          int           `mapstructure:"maxPages"          validate:"min=0,max=500"`
	OCREngine         string        `mapstructure:"ocrEngine"         validate:"omitempty,oneof=rapidocr easyocr tesseract tesserocr auto"`
	OCRLanguages      []string      `mapstructure:"ocrLanguages"      validate:"dive,min=2,max=16"`
	TableMode         string        `mapstructure:"tableMode"         validate:"omitempty,oneof=accurate fast"`
	VLMEndpoint       string        `mapstructure:"vlmEndpoint"       validate:"omitempty,url"`
	VLMModel          string        `mapstructure:"vlmModel"`
}

func (c *Config) Enabled() bool {
	return c.DoclingURL != "" && c.APIKey != ""
}

func (c *Config) MissingAPIKey() bool {
	return c.DoclingURL != "" && c.APIKey == ""
}

func (c *Config) GetDoclingURL() string {
	return strings.TrimSuffix(c.DoclingURL, "/")
}

func (c *Config) GetRequestTimeout() time.Duration {
	if c.RequestTimeout == 0 {
		return 30 * time.Second
	}
	return c.RequestTimeout
}

func (c *Config) GetConversionTimeout() time.Duration {
	if c.ConversionTimeout == 0 {
		return 10 * time.Minute
	}
	return c.ConversionTimeout
}

func (c *Config) GetHealthTimeout() time.Duration {
	if c.HealthTimeout == 0 {
		return 5 * time.Second
	}
	return c.HealthTimeout
}

func (c *Config) GetMaxConcurrent() int {
	if c.MaxConcurrent == 0 {
		return 2
	}
	return c.MaxConcurrent
}

func (c *Config) GetMaxRetries() int {
	if c.MaxRetries == 0 {
		return 3
	}
	return c.MaxRetries
}

func (c *Config) GetMaxFileBytes() int64 {
	if c.MaxFileBytes == 0 {
		return 50 << 20
	}
	return c.MaxFileBytes
}

func (c *Config) GetMaxPages() int {
	if c.MaxPages == 0 {
		return 50
	}
	return c.MaxPages
}

func (c *Config) GetOCREngine() string {
	if c.OCREngine == "" {
		return "rapidocr"
	}
	return c.OCREngine
}

func (c *Config) GetOCRLanguages() []string {
	if len(c.OCRLanguages) == 0 {
		return []string{"en"}
	}
	return c.OCRLanguages
}

func (c *Config) GetTableMode() string {
	if c.TableMode == "" {
		return "accurate"
	}
	return c.TableMode
}

func (c *Config) GetVLMEndpoint() string {
	return strings.TrimSuffix(c.VLMEndpoint, "/")
}

func (c *Config) GetVLMModel() string {
	if c.VLMModel == "" {
		return "ibm-granite/granite-docling-258M"
	}
	return c.VLMModel
}
