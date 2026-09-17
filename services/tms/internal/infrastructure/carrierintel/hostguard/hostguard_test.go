package hostguard

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testAllowed = []string{"api.carrierok.com", "Mobile.FMCSA.dot.gov"}

func TestValidateBaseURL_Accepts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		raw      string
		fallback string
		want     string
	}{
		{
			name:     "empty uses fallback",
			raw:      "",
			fallback: "https://api.carrierok.com",
			want:     "https://api.carrierok.com",
		},
		{
			name: "trailing slash trimmed",
			raw:  "https://api.carrierok.com/",
			want: "https://api.carrierok.com",
		},
		{
			name: "path kept",
			raw:  "https://mobile.fmcsa.dot.gov/qc/services/",
			want: "https://mobile.fmcsa.dot.gov/qc/services",
		},
		{
			name: "host case insensitive",
			raw:  "HTTPS://API.CarrierOk.com",
			want: "https://api.carrierok.com",
		},
		{
			name: "explicit port kept",
			raw:  "https://api.carrierok.com:8443",
			want: "https://api.carrierok.com:8443",
		},
		{name: "localhost http", raw: "http://localhost:8080/", want: "http://localhost:8080"},
		{name: "loopback ip http", raw: "http://127.0.0.1:39211", want: "http://127.0.0.1:39211"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateBaseURL(tc.raw, tc.fallback, testAllowed)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidateBaseURL_Rejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raw     string
		message string
	}{
		{
			name:    "not allowed host",
			raw:     "https://evil.example.com",
			message: "Base URL host is not an approved carrier intelligence host",
		},
		{
			name:    "http public host",
			raw:     "http://api.carrierok.com",
			message: "Base URL must use HTTPS",
		},
		{name: "ftp scheme", raw: "ftp://api.carrierok.com", message: "Base URL must use HTTPS"},
		{
			name:    "private ip",
			raw:     "https://10.0.0.5",
			message: "Base URL host is not an approved carrier intelligence host",
		},
		{name: "metadata ip", raw: "http://169.254.169.254", message: "Base URL must use HTTPS"},
		{
			name:    "credentials",
			raw:     "https://user:secret@api.carrierok.com",
			message: "Base URL must not include credentials",
		},
		{
			name:    "query",
			raw:     "https://api.carrierok.com?x=1",
			message: "Base URL must not include a query string or fragment",
		},
		{name: "relative", raw: "/v2/profile", message: "Base URL is not a valid URL"},
		{
			name:    "suffix trick",
			raw:     "https://api.carrierok.com.evil.example",
			message: "Base URL host is not an approved carrier intelligence host",
		},
		{name: "empty without fallback", raw: "  ", message: "Base URL is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateBaseURL(tc.raw, "", testAllowed)
			require.Error(t, err)

			var guardErr *Error
			require.ErrorAs(t, err, &guardErr)
			assert.Equal(t, tc.message, err.Error())
			assert.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestResolve_BlocksLoopbackWhenSandboxDisallowed(t *testing.T) {
	t.Parallel()

	disallowed := false
	cfg := &config.Config{}
	cfg.CarrierIntelligence.SandboxAllowed = &disallowed

	_, err := Resolve("http://127.0.0.1:9000", "", cfg)
	require.Error(t, err)
	assert.Equal(t, "Local base URLs are not allowed in this environment", err.Error())

	target, err := Resolve("", "https://api.carrierok.com", cfg)
	require.NoError(t, err)
	assert.Equal(t, "https://api.carrierok.com", target.BaseURL)
	assert.False(t, target.Loopback)
}

func TestResolve_AllowsLoopbackOutsideProduction(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	target, err := Resolve("http://localhost:9000/", "", cfg)
	require.NoError(t, err)
	assert.True(t, target.Loopback)
	assert.Equal(t, "http://localhost:9000", target.BaseURL)
	assert.NotNil(t, target.HTTPClient(0))
}
