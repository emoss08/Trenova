package httpsafe

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsBlockedIP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		ip      string
		blocked bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"private 10", "10.0.0.1", true},
		{"private 172", "172.16.5.4", true},
		{"private 192", "192.168.1.1", true},
		{"private v6 ula", "fd00::1", true},
		{"link local metadata", "169.254.169.254", true},
		{"link local v6", "fe80::1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"unspecified v6", "::", true},
		{"multicast", "224.0.0.1", true},
		{"carrier grade nat", "100.64.0.1", true},
		{"public v4", "8.8.8.8", false},
		{"public v6", "2606:4700:4700::1111", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.blocked, IsBlockedIP(net.ParseIP(tc.ip)))
		})
	}
	require.True(t, IsBlockedIP(nil))
}

func TestValidateURL(t *testing.T) {
	t.Parallel()

	t.Run("accepts public https url", func(t *testing.T) {
		t.Parallel()
		parsed, err := ValidateURL("https://partner.example/as2/mdn")
		require.NoError(t, err)
		require.Equal(t, "partner.example", parsed.Hostname())
	})

	t.Run("rejects non http scheme", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURL("file:///etc/passwd")
		require.ErrorIs(t, err, ErrBlockedScheme)
	})

	t.Run("rejects gopher scheme", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURL("gopher://127.0.0.1:11211/")
		require.ErrorIs(t, err, ErrBlockedScheme)
	})

	t.Run("rejects missing host", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURL("http://")
		require.ErrorIs(t, err, ErrMissingHost)
	})

	t.Run("rejects literal metadata ip", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURL("http://169.254.169.254/latest/meta-data/")
		require.ErrorIs(t, err, ErrBlockedAddress)
	})

	t.Run("rejects loopback literal", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURL("http://127.0.0.1:8080/internal")
		require.ErrorIs(t, err, ErrBlockedAddress)
	})
}

func TestNewClientBlocksLoopbackAtDial(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	t.Cleanup(server.Close)

	client := NewClient(5 * time.Second)
	_, err := client.Get(server.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "disallowed network")
}
