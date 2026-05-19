package minio

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeStorageEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		endpoint   string
		defaultSSL bool
		want       string
		wantSSL    bool
		wantErr    bool
	}{
		{
			name:     "host port",
			endpoint: "localhost:9000",
			want:     "localhost:9000",
		},
		{
			name:       "host port uses configured ssl",
			endpoint:   "localhost:9000",
			defaultSSL: true,
			want:       "localhost:9000",
			wantSSL:    true,
		},
		{
			name:     "http endpoint",
			endpoint: "http://localhost:9000",
			want:     "localhost:9000",
		},
		{
			name:     "host port with trailing slash",
			endpoint: "localhost:9000/",
			want:     "localhost:9000",
		},
		{
			name:     "https endpoint",
			endpoint: "https://account-id.r2.cloudflarestorage.com",
			want:     "account-id.r2.cloudflarestorage.com",
			wantSSL:  true,
		},
		{
			name:     "https endpoint with trailing slash",
			endpoint: "https://account-id.r2.cloudflarestorage.com/",
			want:     "account-id.r2.cloudflarestorage.com",
			wantSSL:  true,
		},
		{
			name:     "unsupported scheme",
			endpoint: "ftp://storage.example.com",
			wantErr:  true,
		},
		{
			name:     "path rejected",
			endpoint: "https://storage.example.com/path",
			wantErr:  true,
		},
		{
			name:     "schemeless path rejected",
			endpoint: "storage.example.com/path",
			wantErr:  true,
		},
		{
			name:     "query rejected",
			endpoint: "https://storage.example.com?debug=true",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, gotSSL, err := normalizeStorageEndpoint(tt.endpoint, tt.defaultSSL)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantSSL, gotSSL)
		})
	}
}

func TestNewStorageClientConfig(t *testing.T) {
	t.Parallel()

	t.Run("minio defaults", func(t *testing.T) {
		t.Parallel()

		got, err := newStorageClientConfig(&config.StorageConfig{
			Endpoint: "localhost:9000",
		})

		require.NoError(t, err)
		assert.Equal(t, "localhost:9000", got.endpoint)
		assert.False(t, got.secure)
		assert.Empty(t, got.region)
		assert.Equal(t, miniogo.BucketLookupAuto, got.bucketLookup)
		assert.True(t, got.autoCreateBucket)
	})

	t.Run("r2 defaults", func(t *testing.T) {
		t.Parallel()

		got, err := newStorageClientConfig(&config.StorageConfig{
			Provider: config.StorageProviderR2,
			Endpoint: "account-id.r2.cloudflarestorage.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "account-id.r2.cloudflarestorage.com", got.endpoint)
		assert.True(t, got.secure)
		assert.Equal(t, "auto", got.region)
		assert.Equal(t, miniogo.BucketLookupPath, got.bucketLookup)
		assert.True(t, got.autoCreateBucket)
	})

	t.Run("r2 keeps configured region", func(t *testing.T) {
		t.Parallel()

		got, err := newStorageClientConfig(&config.StorageConfig{
			Provider: config.StorageProviderR2,
			Endpoint: "https://account-id.r2.cloudflarestorage.com",
			Region:   "wnam",
		})

		require.NoError(t, err)
		assert.Equal(t, "wnam", got.region)
	})

	t.Run("r2 rejects public endpoint", func(t *testing.T) {
		t.Parallel()

		_, err := newStorageClientConfig(&config.StorageConfig{
			Provider:       config.StorageProviderR2,
			Endpoint:       "https://account-id.r2.cloudflarestorage.com",
			PublicEndpoint: "https://files.example.com",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "public endpoint cannot be set for R2")
	})

	t.Run("auto create bucket false", func(t *testing.T) {
		t.Parallel()

		autoCreateBucket := false
		got, err := newStorageClientConfig(&config.StorageConfig{
			Endpoint:         "localhost:9000",
			AutoCreateBucket: &autoCreateBucket,
		})

		require.NoError(t, err)
		assert.False(t, got.autoCreateBucket)
	})
}
