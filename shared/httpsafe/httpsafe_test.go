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
		{"carrier grade nat top", "100.127.255.254", true},
		{"this network", "0.1.2.3", true},
		{"this network top", "0.255.255.255", true},
		{"ietf protocol assignments", "192.0.0.8", true},
		{"benchmarking", "198.18.0.1", true},
		{"benchmarking top", "198.19.255.254", true},
		{"documentation 1", "192.0.2.10", true},
		{"documentation 2", "198.51.100.10", true},
		{"documentation 3", "203.0.113.10", true},
		{"reserved class e", "240.0.0.1", true},
		{"broadcast", "255.255.255.255", true},
		{"ipv4 mapped loopback", "::ffff:127.0.0.1", true},
		{"ipv4 mapped private", "::ffff:10.0.0.1", true},
		{"ipv4 mapped metadata", "::ffff:169.254.169.254", true},
		{"ipv4 compatible loopback", "::127.0.0.1", true},
		{"nat64 loopback", "64:ff9b::7f00:1", true},
		{"nat64 private", "64:ff9b::a00:1", true},
		{"nat64 metadata", "64:ff9b::a9fe:a9fe", true},
		{"nat64 local use", "64:ff9b:1::808:808", true},
		{"six to four private", "2002:a00:1::1", true},
		{"six to four loopback", "2002:7f00:1::1", true},
		{"teredo", "2001:0:4136:e378:8000:63bf:3fff:fdd2", true},
		{"site local", "fec0::1", true},
		{"discard only", "100::1", true},
		{"documentation v6", "2001:db8::1", true},
		{"public v4", "8.8.8.8", false},
		{"public just below benchmarking", "198.17.255.255", false},
		{"public just above benchmarking", "198.20.0.1", false},
		{"public just above carrier grade nat", "100.128.0.1", false},
		{"public ipv4 mapped", "::ffff:8.8.8.8", false},
		{"public nat64", "64:ff9b::808:808", false},
		{"public six to four", "2002:808:808::1", false},
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

	t.Run("rejects nat64 literal wrapping metadata", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURL("http://[64:ff9b::a9fe:a9fe]/latest/meta-data/")
		require.ErrorIs(t, err, ErrBlockedAddress)
	})

	t.Run("rejects benchmarking literal", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURL("https://198.18.0.1/")
		require.ErrorIs(t, err, ErrBlockedAddress)
	})

	t.Run("rejects loopback literal", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURL("http://127.0.0.1:8080/internal")
		require.ErrorIs(t, err, ErrBlockedAddress)
	})
}

func TestIsBlockedIPWithPrivateNetworksAllowed(t *testing.T) {
	t.Parallel()

	policy := Policy{AllowPrivateNetworks: true}

	cases := []struct {
		name    string
		ip      string
		blocked bool
	}{
		{"loopback v4", "127.0.0.1", false},
		{"loopback v6", "::1", false},
		{"private 10", "10.0.0.1", false},
		{"private 172", "172.16.5.4", false},
		{"private 192", "192.168.1.1", false},
		{"private v6 ula", "fd00::1", false},
		{"carrier grade nat", "100.64.0.1", false},
		{"public v4", "8.8.8.8", false},
		// The opt-in is for an operator's own model host, never for the metadata
		// service, so link-local stays blocked even here.
		{"link local metadata", "169.254.169.254", true},
		{"link local v6", "fe80::1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"unspecified v6", "::", true},
		{"multicast", "224.0.0.1", true},
		{"this network", "0.1.2.3", true},
		{"reserved class e", "240.0.0.1", true},
		{"broadcast", "255.255.255.255", true},
		{"teredo", "2001:0:4136:e378:8000:63bf:3fff:fdd2", true},
		{"nat64 metadata", "64:ff9b::a9fe:a9fe", true},
		{"ipv4 mapped metadata", "::ffff:169.254.169.254", true},
		{"nat64 private", "64:ff9b::a00:1", false},
		{"benchmarking", "198.18.0.1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.blocked, IsBlockedIPWithPolicy(net.ParseIP(tc.ip), policy))
		})
	}
	require.True(t, IsBlockedIPWithPolicy(nil, policy))
}

func TestValidateURLWithPolicy(t *testing.T) {
	t.Parallel()

	policy := Policy{AllowPrivateNetworks: true}

	t.Run("accepts loopback model server", func(t *testing.T) {
		t.Parallel()
		parsed, err := ValidateURLWithPolicy("http://127.0.0.1:11434/v1", policy)
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1", parsed.Hostname())
	})

	t.Run("accepts private subnet model server", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURLWithPolicy("http://10.4.2.9:8000/v1", policy)
		require.NoError(t, err)
	})

	t.Run("still rejects metadata endpoint", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURLWithPolicy("http://169.254.169.254/latest/meta-data/", policy)
		require.ErrorIs(t, err, ErrBlockedAddress)
	})

	t.Run("still rejects non http scheme", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateURLWithPolicy("file:///etc/passwd", policy)
		require.ErrorIs(t, err, ErrBlockedScheme)
	})
}

func TestNewClientWithPolicyAllowsLoopbackAtDial(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	t.Cleanup(server.Close)

	client := NewClientWithPolicy(5*time.Second, Policy{AllowPrivateNetworks: true})
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusOK, resp.StatusCode)
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

func TestSameOrigin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b string
		want bool
	}{
		{"https://api.openai.com/v1", "https://api.openai.com/v2", true},
		{"https://API.openai.com", "https://api.openai.com:443/v1", true},
		{"http://localhost:11434", "http://localhost:11434/api", true},
		{"https://api.openai.com/v1", "https://attacker.example/v1", false},
		{"https://api.openai.com", "http://api.openai.com", false},
		{"http://localhost:11434", "http://localhost:8000", false},
		{"https://api.openai.com", "", false},
	}
	for _, tc := range cases {
		if got := SameOrigin(tc.a, tc.b); got != tc.want {
			t.Errorf("SameOrigin(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestValidateHost(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidateHost("sftp.partner.example", Policy{}))
	require.NoError(t, ValidateHost("8.8.8.8", Policy{}))
	require.ErrorIs(t, ValidateHost("", Policy{}), ErrMissingHost)
	require.ErrorIs(t, ValidateHost("127.0.0.1", Policy{}), ErrBlockedAddress)
	require.ErrorIs(t, ValidateHost("[::1]", Policy{}), ErrBlockedAddress)
	require.ErrorIs(t, ValidateHost("169.254.169.254", Policy{}), ErrBlockedAddress)
	require.NoError(t, ValidateHost("10.0.0.5", Policy{AllowPrivateNetworks: true}))
	require.ErrorIs(
		t,
		ValidateHost("169.254.169.254", Policy{AllowPrivateNetworks: true}),
		ErrBlockedAddress,
	)
}

func TestDialControlRefusesBlockedAddresses(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	strict := net.Dialer{Timeout: time.Second, Control: DialControl(Policy{})}
	_, err = strict.DialContext(t.Context(), "tcp", listener.Addr().String())
	require.ErrorIs(t, err, ErrBlockedAddress)

	allowed := net.Dialer{
		Timeout: time.Second,
		Control: DialControl(Policy{AllowPrivateNetworks: true}),
	}
	conn, err := allowed.DialContext(t.Context(), "tcp", listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}
