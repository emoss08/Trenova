package managedconfig_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	token := strings.Repeat("a", 32)
	tests := []struct {
		name            string
		cfg             managedconfig.Config
		withoutInstance bool
		err             error
	}{
		{name: "off needs nothing", cfg: managedconfig.Config{}, withoutInstance: true},
		{name: "on with a token", cfg: managedconfig.Config{Enabled: true, Token: token}},
		{
			name: "dedicated tenancy",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Tenancy: managedconfig.TenancyDedicated,
			},
		},
		{
			name: "shared tenancy on a managed server",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Tenancy: managedconfig.TenancyShared,
			},
		},
		{
			name:            "shared tenancy without managed",
			cfg:             managedconfig.Config{Tenancy: managedconfig.TenancyShared},
			withoutInstance: true,
			err:             managedconfig.ErrSharedTenancyNotManaged,
		},
		{
			name: "a tenancy that is neither",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Tenancy: managedconfig.Tenancy("pooled"),
			},
			err: managedconfig.ErrTenancyInvalid,
		},
		{
			name:            "on without the instance's ID",
			cfg:             managedconfig.Config{Enabled: true, Token: token},
			withoutInstance: true,
			err:             managedconfig.ErrInstanceIDInvalid,
		},
		{
			name: "a shared server without an instance ID",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Tenancy: managedconfig.TenancyShared,
			},
			withoutInstance: true,
		},
		{
			name: "a shared server whose ID is not a UUID",
			cfg: managedconfig.Config{
				Enabled:    true,
				Token:      token,
				Tenancy:    managedconfig.TenancyShared,
				InstanceID: "server-1",
			},
			err: managedconfig.ErrInstanceIDInvalid,
		},
		{
			name: "a shared server without a token",
			cfg: managedconfig.Config{
				Enabled: true,
				Tenancy: managedconfig.TenancyShared,
			},
			withoutInstance: true,
			err:             managedconfig.ErrTokenRequired,
		},
		{
			name: "on with an instance ID that is not a UUID",
			cfg: managedconfig.Config{
				Enabled:    true,
				Token:      token,
				InstanceID: "inst_01JTEST",
			},
			err: managedconfig.ErrInstanceIDInvalid,
		},
		{
			name: "on with an uppercase instance ID",
			cfg: managedconfig.Config{
				Enabled:    true,
				Token:      token,
				InstanceID: "0192F3A4-5B6C-7D8E-9F01-23456789ABCD",
			},
			err: managedconfig.ErrInstanceIDInvalid,
		},
		{
			name: "on without a token",
			cfg:  managedconfig.Config{Enabled: true},
			err:  managedconfig.ErrTokenRequired,
		},
		{
			name: "on with a short token",
			cfg:  managedconfig.Config{Enabled: true, Token: "short"},
			err:  managedconfig.ErrTokenRequired,
		},
		{
			name: "on with a bad port",
			cfg:  managedconfig.Config{Enabled: true, Token: token, Port: 70000},
			err:  managedconfig.ErrPortInvalid,
		},
		{
			name: "on with the embedding service",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Embedding: managedconfig.Embedding{
					URL:   "http://trenova-embedding:11434",
					Model: "nomic-embed-text:v1.5",
				},
			},
		},
		{
			name: "an embedding URL with a path",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Embedding: managedconfig.Embedding{
					URL:   "http://trenova-embedding:11434/v1",
					Model: "nomic-embed-text:v1.5",
				},
			},
			err: managedconfig.ErrEmbeddingURLInvalid,
		},
		{
			name: "an embedding URL that is not http",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Embedding: managedconfig.Embedding{
					URL:   "file:///models",
					Model: "nomic-embed-text:v1.5",
				},
			},
			err: managedconfig.ErrEmbeddingURLInvalid,
		},
		{
			name: "an embedding URL with credentials",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Embedding: managedconfig.Embedding{
					URL:   "http://user:pass@trenova-embedding:11434",
					Model: "nomic-embed-text:v1.5",
				},
			},
			err: managedconfig.ErrEmbeddingURLInvalid,
		},
		{
			name: "an embedding URL without a model",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Embedding: managedconfig.Embedding{
					URL: "http://trenova-embedding:11434",
				},
			},
			err: managedconfig.ErrEmbeddingModelMissing,
		},
		{
			name: "an embedding model without a URL",
			cfg: managedconfig.Config{
				Enabled: true,
				Token:   token,
				Embedding: managedconfig.Embedding{
					Model: "nomic-embed-text:v1.5",
				},
			},
			err: managedconfig.ErrEmbeddingModelWithoutURL,
		},
		{
			name: "with the console's support links",
			cfg: managedconfig.Config{
				Enabled:              true,
				Token:                token,
				SupportURL:           "https://console.example.com/support",
				NewSupportRequestURL: "https://console.example.com/support/new?from=trenova&instance=1",
			},
		},
		{
			name: "a relative support link",
			cfg:  managedconfig.Config{Enabled: true, Token: token, SupportURL: "/support"},
			err:  managedconfig.ErrSupportURLInvalid,
		},
		{
			name: "a support link that is not http",
			cfg: managedconfig.Config{
				Enabled:              true,
				Token:                token,
				NewSupportRequestURL: "javascript:alert(1)",
			},
			err: managedconfig.ErrSupportURLInvalid,
		},
		{
			name: "a support link with credentials",
			cfg: managedconfig.Config{
				Enabled:    true,
				Token:      token,
				SupportURL: "https://a:b@console.example.com/support",
			},
			err: managedconfig.ErrSupportURLInvalid,
		},
		{
			name: "a support link with a space",
			cfg: managedconfig.Config{
				Enabled:    true,
				Token:      token,
				SupportURL: "https://console.example.com/sup port",
			},
			err: managedconfig.ErrSupportURLInvalid,
		},
		{
			name: "a support link too long",
			cfg: managedconfig.Config{
				Enabled:    true,
				Token:      token,
				SupportURL: "https://console.example.com/" + strings.Repeat("a", 2048),
			},
			err: managedconfig.ErrSupportURLInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := tt.cfg
			if !tt.withoutInstance && cfg.InstanceID == "" {
				cfg.InstanceID = "0192f3a4-5b6c-7d8e-9f01-23456789abcd"
			}
			require.ErrorIs(t, cfg.Validate(), tt.err)
		})
	}
}

func TestDefaults(t *testing.T) {
	t.Parallel()

	var c managedconfig.Config
	require.Equal(t, "127.0.0.1", c.GetHost())
	require.Equal(t, 8090, c.GetPort())
	require.Contains(t, managedconfig.Defaults(), "embedding.url",
		"the loader applies TRENOVA_MANAGED_EMBEDDING_URL only to a key it knows")
	require.Contains(t, managedconfig.Defaults(), "embedding.model")
	require.Contains(t, managedconfig.Defaults(), "instanceId",
		"the loader applies TRENOVA_MANAGED_INSTANCEID only to a key it knows")
	require.Contains(t, managedconfig.Defaults(), "supportUrl")
	require.Contains(t, managedconfig.Defaults(), "newSupportRequestUrl")
}

func TestSharedTenancyIsOnlyAManagedServers(t *testing.T) {
	t.Parallel()

	var none *managedconfig.Config
	require.False(t, none.SharedTenancy())
	require.False(t, (&managedconfig.Config{Tenancy: managedconfig.TenancyShared}).SharedTenancy())
	require.False(t, (&managedconfig.Config{Enabled: true}).SharedTenancy())
	require.False(t, (&managedconfig.Config{
		Enabled: true,
		Tenancy: managedconfig.TenancyDedicated,
	}).SharedTenancy())
	require.True(t, (&managedconfig.Config{
		Enabled: true,
		Tenancy: managedconfig.TenancyShared,
	}).SharedTenancy())
	require.Contains(t, managedconfig.Defaults(), "tenancy")
}

func TestSupportIsOnlyAManagedInstances(t *testing.T) {
	t.Parallel()

	links := managedconfig.Config{
		SupportURL:           " https://console.example.com/support ",
		NewSupportRequestURL: "https://console.example.com/support/new",
	}
	require.Equal(t, managedconfig.Support{}, links.Support(), "self-hosted")
	var none *managedconfig.Config
	require.Equal(t, managedconfig.Support{}, none.Support())

	links.Enabled = true
	require.Equal(t, managedconfig.Support{
		RequestsURL:   "https://console.example.com/support",
		NewRequestURL: "https://console.example.com/support/new",
	}, links.Support())
}

func TestBuiltInEmbedding(t *testing.T) {
	t.Parallel()

	service := managedconfig.Embedding{
		URL:   " http://trenova-embedding:11434/ ",
		Model: " nomic-embed-text:v1.5 ",
	}
	tests := []struct {
		name string
		cfg  *managedconfig.Config
		ok   bool
	}{
		{name: "no config", cfg: nil},
		{name: "self-hosted, even when the service is named", cfg: &managedconfig.Config{
			Embedding: service,
		}},
		{name: "managed without the service", cfg: &managedconfig.Config{Enabled: true}},
		{name: "managed with the service", cfg: &managedconfig.Config{
			Enabled:   true,
			Embedding: service,
		}, ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := tt.cfg.BuiltInEmbedding()
			require.Equal(t, tt.ok, ok)
			if tt.ok {
				require.Equal(t, managedconfig.Embedding{
					URL:   "http://trenova-embedding:11434",
					Model: "nomic-embed-text:v1.5",
				}, got)
			}
		})
	}
}
