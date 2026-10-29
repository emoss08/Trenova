package storageconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStorageConfig_GetMaxFileSize(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, int64(52428800), c.GetMaxFileSize())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{MaxFileSize: 104857600}
		assert.Equal(t, int64(104857600), c.GetMaxFileSize())
	})
}

func TestStorageConfig_GetPresignedURLExpiry(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, 15*time.Minute, c.GetPresignedURLExpiry())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{PresignedURLExpiry: 30 * time.Minute}
		assert.Equal(t, 30*time.Minute, c.GetPresignedURLExpiry())
	})
}

func TestStorageConfig_GetAllowedMIMETypes(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		result := c.GetAllowedMIMETypes()
		assert.Len(t, result, 14)
		assert.Contains(t, result, "application/pdf")
		assert.Contains(t, result, "image/jpeg")
		assert.Contains(t, result, "image/heic", "phone photos from Dash")
		assert.Contains(t, result, "image/heif")
		assert.Contains(t, result, "text/csv")
		assert.NotContains(t, result, "text/html")
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		custom := []string{"image/png", "application/pdf"}
		c := &Config{AllowedMIMETypes: custom}
		assert.Equal(t, custom, c.GetAllowedMIMETypes())
	})
}

func TestStorageConfig_GetProvider(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, ProviderMinio, c.GetProvider())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{Provider: ProviderR2}
		assert.Equal(t, ProviderR2, c.GetProvider())
	})
}

func TestStorageConfig_ShouldAutoCreateBucket(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.True(t, c.ShouldAutoCreateBucket())
	})

	t.Run("custom false", func(t *testing.T) {
		t.Parallel()
		autoCreateBucket := false
		c := &Config{AutoCreateBucket: &autoCreateBucket}
		assert.False(t, c.ShouldAutoCreateBucket())
	})
}
