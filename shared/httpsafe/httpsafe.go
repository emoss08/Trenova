// Package httpsafe provides SSRF-hardened helpers for issuing outbound HTTP
// requests whose target URL is derived from untrusted input. It rejects
// non-HTTP(S) schemes and refuses to connect to loopback, private, link-local
// (including cloud metadata endpoints), shared-address-space, multicast, and
// unspecified network addresses, re-checking every resolved IP at dial time to
// defeat DNS-rebinding.
package httpsafe

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

var (
	ErrBlockedScheme  = errors.New("httpsafe: url scheme is not allowed")
	ErrMissingHost    = errors.New("httpsafe: url host is missing")
	ErrBlockedAddress = errors.New("httpsafe: address resolves to a disallowed network")
)

const (
	defaultDialTimeout      = 10 * time.Second
	defaultKeepAlive        = 30 * time.Second
	defaultIdleConnTimeout  = 90 * time.Second
	defaultTLSHandshake     = 10 * time.Second
	defaultExpectContinue   = 1 * time.Second
	defaultMaxIdleConns     = 100
	defaultResponseHeaderTO = 30 * time.Second
)

// carrierGradeNAT is the RFC 6598 shared address space (100.64.0.0/10) used by
// many cloud providers for internal routing; it is not covered by the net.IP
// classification helpers.
var carrierGradeNAT = &net.IPNet{
	IP:   net.IPv4(100, 64, 0, 0),
	Mask: net.CIDRMask(10, 32),
}

// IsBlockedIP reports whether connecting to ip would reach a non-publicly
// routable destination that could be leveraged for server-side request forgery.
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return true
	}
	return carrierGradeNAT.Contains(ip)
}

// ValidateURL parses rawURL and ensures it uses an allowed scheme and, when the
// host is a literal IP address, that the address is publicly routable.
// Hostnames are resolved and re-checked at connection time by the client
// returned from NewClient, so a passing result here is a necessary but not
// sufficient guarantee on its own.
func ValidateURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("httpsafe: url is invalid: %w", err)
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return nil, fmt.Errorf("%w: %q", ErrBlockedScheme, parsed.Scheme)
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, ErrMissingHost
	}
	if ip := net.ParseIP(host); ip != nil && IsBlockedIP(ip) {
		return nil, fmt.Errorf("%w: %s", ErrBlockedAddress, host)
	}
	return parsed, nil
}

func guardedControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("httpsafe: dial address %q is invalid: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("%w: %s", ErrBlockedAddress, host)
	}
	if IsBlockedIP(ip) {
		return fmt.Errorf("%w: %s", ErrBlockedAddress, host)
	}
	return nil
}

// NewClient returns an *http.Client whose dialer refuses connections to
// disallowed network addresses, re-checking every resolved IP to defeat
// DNS-rebinding. Redirects are not followed, since a redirect target could
// otherwise escape the URL-level validation.
func NewClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
		Control:   guardedControl,
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          defaultMaxIdleConns,
			IdleConnTimeout:       defaultIdleConnTimeout,
			TLSHandshakeTimeout:   defaultTLSHandshake,
			ExpectContinueTimeout: defaultExpectContinue,
			ResponseHeaderTimeout: defaultResponseHeaderTO,
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
