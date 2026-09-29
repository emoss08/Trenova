package urlutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/urlutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAbsoluteHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		ok   bool
		host string
	}{
		{name: "https", raw: "https://hooks.example.com/in", ok: true, host: "hooks.example.com"},
		{name: "http with port", raw: "http://localhost:8080", ok: true, host: "localhost:8080"},
		{name: "surrounding space", raw: "  https://example.com  ", ok: true, host: "example.com"},
		{name: "empty", raw: ""},
		{name: "no scheme", raw: "example.com/in"},
		{name: "another scheme", raw: "ftp://example.com/in"},
		{name: "no host", raw: "https:///in"},
		{name: "scheme only", raw: "https:"},
		{name: "unparseable", raw: "http://[::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, ok := urlutils.ParseAbsoluteHTTP(tt.raw)
			assert.Equal(t, tt.ok, ok)
			if !tt.ok {
				assert.Nil(t, parsed)
				return
			}
			require.NotNil(t, parsed)
			assert.Equal(t, tt.host, parsed.Host)
		})
	}
}
