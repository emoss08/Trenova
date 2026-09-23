// Package httpsafe provides SSRF-hardened helpers for issuing outbound HTTP
// requests whose target URL is derived from untrusted input. It rejects
// non-HTTP(S) schemes and refuses to connect to loopback, private, link-local
// (including cloud metadata endpoints), shared-address-space, multicast, and
// unspecified network addresses, re-checking every resolved IP at dial time to
// defeat DNS-rebinding.
//
// A caller that must reach an operator-designated host on its own network — a
// self-hosted model server, for instance — opts in with Policy.AllowPrivateNetworks.
// That relaxation is deliberately partial: loopback and private ranges open up,
// while link-local stays blocked, because 169.254.169.254 is the cloud metadata
// endpoint and reaching it is never the intent behind "this host is internal".
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

// Policy controls how permissive the address checks are. The zero value is the
// strict default that every existing caller gets.
type Policy struct {
	// AllowPrivateNetworks permits loopback, RFC 1918 / RFC 4193 private, and
	// carrier-grade-NAT destinations. Link-local, multicast, and unspecified
	// addresses remain blocked regardless, so cloud metadata endpoints stay
	// unreachable. Only set this for a host an operator explicitly designated.
	AllowPrivateNetworks bool

	// ResponseHeaderTimeout overrides how long the transport waits for response
	// headers. A model server that holds the connection open while it generates
	// sends nothing until the first token, which on a loaded self-hosted GPU can
	// outlast the 30s default. Zero keeps that default.
	ResponseHeaderTimeout time.Duration
}

// IsBlockedIP reports whether connecting to ip would reach a non-publicly
// routable destination that could be leveraged for server-side request forgery.
func IsBlockedIP(ip net.IP) bool {
	return IsBlockedIPWithPolicy(ip, Policy{})
}

// IsBlockedIPWithPolicy reports whether connecting to ip is disallowed under p.
func IsBlockedIPWithPolicy(ip net.IP, p Policy) bool {
	if ip == nil {
		return true
	}
	// Never reachable: an unspecified or multicast target is not a model server,
	// and link-local covers the metadata service that SSRF most often targets.
	if ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if p.AllowPrivateNetworks {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() {
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
	return ValidateURLWithPolicy(rawURL, Policy{})
}

// ValidateURLWithPolicy is ValidateURL under an explicit Policy.
func ValidateURLWithPolicy(rawURL string, p Policy) (*url.URL, error) {
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
	if ip := net.ParseIP(host); ip != nil && IsBlockedIPWithPolicy(ip, p) {
		return nil, fmt.Errorf("%w: %s", ErrBlockedAddress, host)
	}
	return parsed, nil
}

func guardedControl(p Policy) func(string, string, syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("httpsafe: dial address %q is invalid: %w", address, err)
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("%w: %s", ErrBlockedAddress, host)
		}
		if IsBlockedIPWithPolicy(ip, p) {
			return fmt.Errorf("%w: %s", ErrBlockedAddress, host)
		}
		return nil
	}
}

// NewClient returns an *http.Client whose dialer refuses connections to
// disallowed network addresses, re-checking every resolved IP to defeat
// DNS-rebinding. Redirects are not followed, since a redirect target could
// otherwise escape the URL-level validation.
func NewClient(timeout time.Duration) *http.Client {
	return NewClientWithPolicy(timeout, Policy{})
}

// NewClientWithPolicy is NewClient under an explicit Policy. The policy is bound
// into the dialer, so a client built for a private-network destination cannot be
// reused to reach one that was never vetted.
func NewClientWithPolicy(timeout time.Duration, p Policy) *http.Client {
	dialer := &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
		Control:   guardedControl(p),
	}
	responseHeaderTO := p.ResponseHeaderTimeout
	if responseHeaderTO <= 0 {
		responseHeaderTO = defaultResponseHeaderTO
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
			ResponseHeaderTimeout: responseHeaderTO,
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// NewStreamingClientWithPolicy builds a client for responses read incrementally.
//
// It is NewClientWithPolicy without the whole-request timeout, and that absence
// is the point. http.Client.Timeout covers reading the response body, so on a
// streamed reply it is a deadline for the entire exchange: a server-sent-event
// stream that is delivering perfectly well is cut off the moment it elapses,
// mid-token, with an error that reads like a network fault. Streams are kept
// honest by bounding silence between reads instead, which is the caller's job
// because only the caller knows what a reasonable gap is.
//
// The egress guard, the header timeout and the redirect policy are unchanged:
// nothing about streaming loosens where a request may go.
func NewStreamingClientWithPolicy(p Policy) *http.Client {
	client := NewClientWithPolicy(0, p)
	client.Timeout = 0

	return client
}

// SameOrigin reports whether two URLs reach the same server: the same scheme,
// host and port. A credential stored for one origin must not be sent to
// another, so a change of origin is a change of who receives it. An address
// that does not parse is an origin of its own.
func SameOrigin(a, b string) bool {
	left, err := url.Parse(strings.TrimSpace(a))
	if err != nil {
		return false
	}
	right, err := url.Parse(strings.TrimSpace(b))
	if err != nil {
		return false
	}

	return strings.EqualFold(left.Scheme, right.Scheme) &&
		strings.EqualFold(left.Hostname(), right.Hostname()) &&
		effectivePort(left) == effectivePort(right)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}
