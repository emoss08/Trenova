// Package capturereleaseservice serves the current Trenova Capture release.
//
// The release build signs a manifest with an ed25519 key and publishes it,
// by default as an asset of the repository's rolling capture-stable release.
// This service reads it, checks the signature against the configured public
// key and the release it describes, caches it, and hands out the signed
// manifest unchanged: the companion checks the same signature itself against
// a key built into it, so the server is a convenient mirror, never a party
// the companion has to trust.
package capturereleaseservice

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/versionutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	// Product is what every manifest names.
	Product = "trenova-capture"
	// MaxSignedBytes bounds a manifest read from anywhere.
	MaxSignedBytes = 64 << 10
	// MaxInstallerBytes bounds the installer a manifest may describe.
	MaxInstallerBytes = 512 << 20

	maxNotesRunes    = 4000
	maxURLLength     = 2048
	maxFileNameBytes = 128
	fetchTimeout     = 15 * time.Second
	freshFor         = 15 * time.Minute
	staleFor         = 24 * time.Hour
	failureBackoff   = time.Minute
	maxRedirects     = 5
)

var (
	sha256Hex     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	plainMSIName  = regexp.MustCompile(`^[A-Za-z0-9._-]+\.[mM][sS][iI]$`)
	errBadRelease = errors.New("the capture release manifest is invalid")
)

// SignedRelease is the manifest as published: its exact bytes, base64, and
// their signature.
type SignedRelease struct {
	Manifest  string `json:"manifest"`
	Signature string `json:"signature"`
}

type Installer struct {
	FileName string `json:"fileName"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

// Release mirrors capture_protocol::release::Release.
type Release struct {
	Product             string    `json:"product"`
	Version             string    `json:"version"`
	PublishedAt         int64     `json:"publishedAt"`
	MinimumWindowsBuild int       `json:"minimumWindowsBuild"`
	Installer           Installer `json:"installer"`
	Notes               string    `json:"notes"`
}

// Latest is the current release and the signed manifest it came from.
type Latest struct {
	Release *Release
	Signed  *SignedRelease
}

type Params struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
}

type Service struct {
	l       *zap.Logger
	url     string
	key     ed25519.PublicKey
	offline bool
	client  *http.Client
	flights singleflight.Group
	now     func() time.Time

	mu        sync.Mutex
	cached    *Latest
	fetchedAt time.Time
	failedAt  time.Time
}

// ParsePublicKey reads a base64 ed25519 public key.
func ParsePublicKey(value string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, errors.New("the capture release key is not a base64 ed25519 public key")
	}

	return ed25519.PublicKey(raw), nil
}

func newClient(proxy string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("parse update proxy: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &http.Client{
		Timeout:   fetchTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "https" {
				return errors.New("refusing a redirect away from HTTPS")
			}

			return nil
		},
	}, nil
}

func New(p Params) (*Service, error) {
	update := p.Config.Update
	s := &Service{
		l:       p.Logger.Named("service.capture-release"),
		url:     update.GetCaptureManifestURL(),
		offline: update.OfflineMode,
		now:     time.Now,
	}

	client, err := newClient(update.ProxyURL)
	if err != nil {
		return nil, err
	}
	s.client = client

	if strings.TrimSpace(update.CapturePublicKey) == "" {
		s.l.Info("no capture release key is configured; no Trenova Capture release will be served")

		return s, nil
	}
	if s.key, err = ParsePublicKey(update.CapturePublicKey); err != nil {
		return nil, err
	}

	return s, nil
}

// Enabled reports whether this server can serve a release at all.
func (s *Service) Enabled() bool {
	return s.key != nil && !s.offline
}

// Latest returns the current release, or nil when none is published or this
// server is not set up to serve one. A failed fetch falls back to the last
// good manifest for a day, and is not retried for a minute.
func (s *Service) Latest(ctx context.Context) (*Latest, error) {
	if !s.Enabled() {
		return nil, nil //nolint:nilnil // no release is a normal answer
	}

	s.mu.Lock()
	now := s.now()
	if s.fetchedAt.After(now.Add(-freshFor)) || s.failedAt.After(now.Add(-failureBackoff)) {
		cached, fetched := s.cached, s.fetchedAt
		s.mu.Unlock()
		if cached == nil || fetched.Before(now.Add(-staleFor)) {
			return nil, nil //nolint:nilnil // nothing usable is cached
		}

		return cached, nil
	}
	s.mu.Unlock()

	result, err, _ := s.flights.Do("latest", func() (any, error) {
		return s.fetch(context.WithoutCancel(ctx))
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	now = s.now()
	if err != nil {
		s.failedAt = now
		s.l.Warn("could not read the capture release manifest",
			zap.String("url", s.url), zap.Error(err))
		if s.cached != nil && s.fetchedAt.After(now.Add(-staleFor)) {
			return s.cached, nil
		}

		return nil, err
	}

	latest, _ := result.(*Latest)
	s.cached = latest
	s.fetchedAt = now

	return latest, nil
}

func (s *Service) fetch(ctx context.Context) (*Latest, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build the manifest request: %w", err)
	}
	req.Header.Set("Accept", "application/json, application/octet-stream")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch the manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil //nolint:nilnil // nothing published yet
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch the manifest: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxSignedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read the manifest: %w", err)
	}
	if len(body) > MaxSignedBytes {
		return nil, errors.New("the manifest is larger than any manifest should be")
	}

	return Verify(body, s.key)
}

// Verify checks a signed manifest's signature and the release it describes.
func Verify(body []byte, key ed25519.PublicKey) (*Latest, error) {
	var signed SignedRelease
	if err := sonic.Unmarshal(body, &signed); err != nil {
		return nil, fmt.Errorf("%w: %w", errBadRelease, err)
	}

	manifest, err := base64.StdEncoding.DecodeString(signed.Manifest)
	if err != nil {
		return nil, fmt.Errorf("%w: the manifest is not base64", errBadRelease)
	}
	signature, err := base64.StdEncoding.DecodeString(signed.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("%w: the signature is malformed", errBadRelease)
	}
	if !ed25519.Verify(key, manifest, signature) {
		return nil, fmt.Errorf("%w: the signature does not verify", errBadRelease)
	}

	var release Release
	if err = sonic.Unmarshal(manifest, &release); err != nil {
		return nil, fmt.Errorf("%w: %w", errBadRelease, err)
	}
	if err = release.Validate(); err != nil {
		return nil, err
	}

	return &Latest{Release: &release, Signed: &signed}, nil
}

// Validate checks everything a reader acts on, as the companion does.
func (r *Release) Validate() error {
	invalid := func(reason string) error {
		return fmt.Errorf("%w: %s", errBadRelease, reason)
	}

	switch {
	case r.Product != Product:
		return invalid("the manifest is for another product")
	case strings.HasPrefix(r.Version, "v") || !versionutils.IsValid(r.Version):
		return invalid("the version is not MAJOR.MINOR.PATCH")
	case !strings.HasPrefix(r.Installer.URL, "https://") || len(r.Installer.URL) > maxURLLength:
		return invalid("the installer must come over HTTPS")
	case !sha256Hex.MatchString(r.Installer.SHA256):
		return invalid("the installer checksum is not SHA-256")
	case r.Installer.Size <= 0 || r.Installer.Size > MaxInstallerBytes:
		return invalid("the installer size is out of range")
	case len(r.Installer.FileName) > maxFileNameBytes ||
		!plainMSIName.MatchString(r.Installer.FileName):
		return invalid("the installer file name is not a plain .msi name")
	case utf8.RuneCountInString(r.Notes) > maxNotesRunes:
		return invalid("the release notes are too long")
	}

	if _, err := url.Parse(r.Installer.URL); err != nil {
		return invalid("the installer URL does not parse")
	}

	return nil
}
