package capturereleaseservice

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// testSeed is the seed the Rust fixture in testdata/signed.json was signed
// with (`trenova-capture-release sign`), so this checks the two languages
// agree on the signature, not just that Go agrees with itself.
func testKey() ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
}

func publicKeyText(key ed25519.PrivateKey) string {
	public, _ := key.Public().(ed25519.PublicKey)

	return base64.StdEncoding.EncodeToString(public)
}

func validRelease() Release {
	return Release{
		Product:             Product,
		Version:             "1.4.2",
		PublishedAt:         1_780_000_000,
		MinimumWindowsBuild: 19045,
		Installer: Installer{
			FileName: "TrenovaCapture-1.4.2-x64.msi",
			URL:      "https://example.test/TrenovaCapture-1.4.2-x64.msi",
			SHA256:   strings.Repeat("ab", 32),
			Size:     12_345_678,
		},
	}
}

func signed(t *testing.T, key ed25519.PrivateKey, release Release) []byte {
	t.Helper()
	manifest, err := sonic.Marshal(release)
	require.NoError(t, err)
	body, err := sonic.Marshal(SignedRelease{
		Manifest:  base64.StdEncoding.EncodeToString(manifest),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(key, manifest)),
	})
	require.NoError(t, err)

	return body
}

func TestVerifyReadsAManifestTheRustToolSigned(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/signed.json")
	require.NoError(t, err)
	public, _ := testKey().Public().(ed25519.PublicKey)

	latest, err := Verify(body, public)
	require.NoError(t, err)
	assert.Equal(t, "1.4.2", latest.Release.Version)
	assert.Equal(t, "TrenovaCapture-1.4.2-x64.msi", latest.Release.Installer.FileName)
	assert.EqualValues(t, 20, latest.Release.Installer.Size)
	assert.Equal(t, 19045, latest.Release.MinimumWindowsBuild)
}

func TestVerifyRefusesATamperedManifestOrAnotherKey(t *testing.T) {
	t.Parallel()

	body := signed(t, testKey(), validRelease())
	public, _ := testKey().Public().(ed25519.PublicKey)
	_, err := Verify(body, public)
	require.NoError(t, err)

	var envelope SignedRelease
	require.NoError(t, sonic.Unmarshal(body, &envelope))
	manifest, err := base64.StdEncoding.DecodeString(envelope.Manifest)
	require.NoError(t, err)
	envelope.Manifest = base64.StdEncoding.EncodeToString(
		bytes.Replace(manifest, []byte("1.4.2"), []byte("9.4.2"), 1),
	)
	tampered, err := sonic.Marshal(envelope)
	require.NoError(t, err)
	_, err = Verify(tampered, public)
	require.ErrorIs(t, err, errBadRelease)

	other := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	otherPublic, _ := other.Public().(ed25519.PublicKey)
	_, err = Verify(body, otherPublic)
	require.ErrorIs(t, err, errBadRelease)

	_, err = Verify([]byte("not json"), public)
	require.ErrorIs(t, err, errBadRelease)
}

func TestValidateRefusesASignedManifestThatPointsSomewhereUnexpected(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*Release){
		"plain http":     func(r *Release) { r.Installer.URL = "http://example.test/a.msi" },
		"path in name":   func(r *Release) { r.Installer.FileName = `..\evil.msi` },
		"not an msi":     func(r *Release) { r.Installer.FileName = "setup.exe" },
		"other product":  func(r *Release) { r.Product = "something-else" },
		"huge":           func(r *Release) { r.Installer.Size = MaxInstallerBytes + 1 },
		"empty":          func(r *Release) { r.Installer.Size = 0 },
		"upper checksum": func(r *Release) { r.Installer.SHA256 = strings.Repeat("AB", 32) },
		"short version":  func(r *Release) { r.Version = "1.4" },
		"prefixed":       func(r *Release) { r.Version = "v1.4.2" },
		"long notes":     func(r *Release) { r.Notes = strings.Repeat("x", maxNotesRunes+1) },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			release := validRelease()
			change(&release)
			require.ErrorIs(t, release.Validate(), errBadRelease)
		})
	}
}

func newService(t *testing.T, url, key string) *Service {
	t.Helper()
	svc, err := New(Params{
		Logger: zap.NewNop(),
		Config: &config.Config{Update: config.UpdateConfig{
			CaptureManifestURL: url,
			CapturePublicKey:   key,
		}},
	})
	require.NoError(t, err)

	return svc
}

func TestLatestCachesTheVerifiedManifestAndFallsBackToItWhenTheSourceFails(t *testing.T) {
	t.Parallel()

	body := signed(t, testKey(), validRelease())
	var hits atomic.Int32
	var failing atomic.Bool
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		if failing.Load() {
			w.WriteHeader(http.StatusBadGateway)

			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(source.Close)

	svc := newService(t, source.URL, publicKeyText(testKey()))
	clock := time.Unix(1_780_000_000, 0)
	svc.now = func() time.Time { return clock }

	latest, err := svc.Latest(t.Context())
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, "1.4.2", latest.Release.Version)
	assert.Equal(t, string(body), mustJSON(t, latest.Signed), "the manifest is served unchanged")

	_, err = svc.Latest(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 1, hits.Load(), "a fresh manifest is not fetched again")

	failing.Store(true)
	clock = clock.Add(freshFor + time.Second)
	stale, err := svc.Latest(t.Context())
	require.NoError(t, err)
	require.NotNil(t, stale, "the last good manifest is served while the source is down")
	assert.EqualValues(t, 2, hits.Load())

	_, err = svc.Latest(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 2, hits.Load(), "a failure is not retried at once")

	clock = clock.Add(staleFor)
	gone, err := svc.Latest(t.Context())
	require.Error(t, err)
	assert.Nil(t, gone, "a day-old manifest is no longer served")
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	out, err := sonic.Marshal(value)
	require.NoError(t, err)

	return string(out)
}

func TestLatestIsNothingWhenUnpublishedUnconfiguredOrForged(t *testing.T) {
	t.Parallel()

	missing := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(missing.Close)
	latest, err := newService(t, missing.URL, publicKeyText(testKey())).Latest(t.Context())
	require.NoError(t, err)
	assert.Nil(t, latest)

	unconfigured := newService(t, missing.URL, "")
	assert.False(t, unconfigured.Enabled())
	latest, err = unconfigured.Latest(t.Context())
	require.NoError(t, err)
	assert.Nil(t, latest)

	forger := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, ed25519.SeedSize))
	forged := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(signed(t, forger, validRelease()))
	}))
	t.Cleanup(forged.Close)
	latest, err = newService(t, forged.URL, publicKeyText(testKey())).Latest(t.Context())
	require.ErrorIs(t, err, errBadRelease)
	assert.Nil(t, latest)

	huge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), MaxSignedBytes+1))
	}))
	t.Cleanup(huge.Close)
	_, err = newService(t, huge.URL, publicKeyText(testKey())).Latest(t.Context())
	require.Error(t, err)
}

func TestTheDefaultManifestIsTheCaptureStableReleaseAndABadKeyIsFatal(t *testing.T) {
	t.Parallel()

	update := config.UpdateConfig{}
	assert.Equal(t,
		"https://github.com/emoss08/trenova/releases/download/capture-stable/trenova-capture-manifest.json",
		update.GetCaptureManifestURL())

	_, err := New(Params{
		Logger: zap.NewNop(),
		Config: &config.Config{Update: config.UpdateConfig{CapturePublicKey: "not a key"}},
	})
	require.Error(t, err)

	offline := newService(t, "", publicKeyText(testKey()))
	offline.offline = true
	assert.False(t, offline.Enabled())
}
