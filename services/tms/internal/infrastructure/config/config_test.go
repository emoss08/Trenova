package config

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/serverconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenderingconfig"
	"github.com/stretchr/testify/assert"
)

func TestConfig_CorsEnabled(t *testing.T) {
	t.Parallel()

	t.Run("enabled", func(t *testing.T) {
		t.Parallel()
		c := &Config{Server: serverconfig.Config{CORS: serverconfig.CORSConfig{Enabled: true}}}
		assert.True(t, c.CorsEnabled())
	})

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()
		c := &Config{Server: serverconfig.Config{CORS: serverconfig.CORSConfig{Enabled: false}}}
		assert.False(t, c.CorsEnabled())
	})
}

func TestConfig_PublicLinkBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		web       string
		tendering string
		want      string
	}{
		{name: "neither set", want: ""},
		{
			name: "web base url only",
			web:  "https://tms.example.com/",
			want: "https://tms.example.com",
		},
		{
			name:      "tendering override wins",
			web:       "https://tms.example.com",
			tendering: "https://links.example.com/",
			want:      "https://links.example.com",
		},
		{
			name:      "tendering only",
			tendering: "https://links.example.com",
			want:      "https://links.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Config{
				App:       appconfig.Config{WebBaseURL: tt.web},
				Tendering: tenderingconfig.Config{PublicBaseURL: tt.tendering},
			}
			assert.Equal(t, tt.want, c.PublicLinkBaseURL())
		})
	}
}
