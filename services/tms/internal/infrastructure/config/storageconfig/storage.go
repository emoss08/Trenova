package storageconfig

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrProductionTLSRequired = errors.New(
		"production and staging require TLS for non-local object storage endpoints",
	)
)

type Config struct {
	Provider           string        `mapstructure:"provider"           validate:"omitempty,oneof=minio r2"`
	Endpoint           string        `mapstructure:"endpoint"           validate:"required"`
	PublicEndpoint     string        `mapstructure:"publicEndpoint"`
	AccessKey          string        `mapstructure:"accessKey"          validate:"required"`
	SecretKey          string        `mapstructure:"secretKey"          validate:"required"`
	SessionToken       string        `mapstructure:"sessionToken"`
	Bucket             string        `mapstructure:"bucket"             validate:"required"`
	UseSSL             bool          `mapstructure:"useSSL"`
	Region             string        `mapstructure:"region"`
	AutoCreateBucket   *bool         `mapstructure:"autoCreateBucket"`
	MaxFileSize        int64         `mapstructure:"maxFileSize"`
	MaxFilesPerUpload  int           `mapstructure:"maxFilesPerUpload"  validate:"min=0,max=100"`
	PresignedURLExpiry time.Duration `mapstructure:"presignedUrlExpiry"`
	AllowedMIMETypes   []string      `mapstructure:"allowedMimeTypes"`
}

const (
	ProviderMinio = "minio"
	ProviderR2    = "r2"
)

func (c *Config) GetProvider() string {
	if c.Provider == "" {
		return ProviderMinio
	}
	return c.Provider
}

func (c *Config) ShouldAutoCreateBucket() bool {
	if c.AutoCreateBucket == nil {
		return true
	}
	return *c.AutoCreateBucket
}

func (c *Config) GetMaxFileSize() int64 {
	if c.MaxFileSize == 0 {
		return 52428800
	}
	return c.MaxFileSize
}

func (c *Config) GetMaxFilesPerUpload() int {
	if c.MaxFilesPerUpload == 0 {
		return 10
	}
	return c.MaxFilesPerUpload
}

func (c *Config) GetPresignedURLExpiry() time.Duration {
	if c.PresignedURLExpiry == 0 {
		return 15 * time.Minute
	}
	return c.PresignedURLExpiry
}

func (c *Config) GetAllowedMIMETypes() []string {
	if len(c.AllowedMIMETypes) == 0 {
		return []string{
			"application/pdf",
			"image/jpeg",
			"image/png",
			"image/webp",
			"image/gif",
			"image/tiff",
			"image/heic",
			"image/heif",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"text/plain",
			"text/csv",
		}
	}

	return c.AllowedMIMETypes
}

func Defaults() map[string]any {
	return map[string]any{
		"provider":         ProviderMinio,
		"autoCreateBucket": true,
	}
}

func (c *Config) ValidateProduction() error {
	if c.GetProvider() == ProviderR2 &&
		(!c.UseSSL || !strings.HasPrefix(c.Endpoint, "https://")) {
		return ErrProductionTLSRequired
	}

	return nil
}

func (c *Config) Validate() error {
	if c.GetProvider() == ProviderR2 && c.PublicEndpoint != "" {
		return errors.New(
			"storage.publicEndpoint cannot be set when storage.provider is r2; private R2 presigned URLs must use the R2 S3 API endpoint",
		)
	}

	return nil
}
