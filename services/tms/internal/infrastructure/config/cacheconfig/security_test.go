package cacheconfig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_NeedsBothHalvesOfAClientCertificate(t *testing.T) {
	t.Parallel()

	c := &Config{TLS: TLSConfig{Enabled: true, CertFile: "/certs/client.crt"}}

	require.ErrorIs(t, c.Validate(), ErrTLSClientCertificatePair)
}

func TestValidate_RefusesTLSSettingsWhileTLSIsOff(t *testing.T) {
	t.Parallel()

	c := &Config{TLS: TLSConfig{CAFile: "/certs/ca.crt"}}

	require.ErrorIs(t, c.Validate(), ErrTLSSettingsWithoutTLS)
}

func TestValidate_AcceptsPlainAndVerifiedConnections(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&Config{}).Validate())
	require.NoError(t, (&Config{TLS: TLSConfig{Enabled: true, CAFile: "/certs/ca.crt"}}).Validate())
}

func TestValidateProduction_RequiresARealPassword(t *testing.T) {
	t.Parallel()

	for _, password := range []string{"", "   ", "change-me", "my-secret-password"} {
		c := &Config{Password: password}
		require.ErrorIs(t, c.ValidateProduction(), ErrProductionRedisPasswordRequired, password)
	}
	require.NoError(t, (&Config{Password: "9f8e7d6c5b4a39281706f5e4d3c2b1a0"}).ValidateProduction())
}

func TestTLSClientConfig_VerifiesTheHostByDefault(t *testing.T) {
	t.Parallel()

	cfg, err := (&Config{Host: "redis.internal", TLS: TLSConfig{Enabled: true}}).TLSClientConfig()

	require.NoError(t, err)
	assert.Equal(t, "redis.internal", cfg.ServerName)
	assert.False(t, cfg.InsecureSkipVerify)
	assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
}

func TestTLSClientConfig_TrustsTheConfiguredAuthority(t *testing.T) {
	t.Parallel()

	caFile := writeSelfSignedCA(t)
	cfg, err := (&Config{
		Host: "redis",
		TLS:  TLSConfig{Enabled: true, ServerName: "cache.trenova", CAFile: caFile},
	}).TLSClientConfig()

	require.NoError(t, err)
	assert.Equal(t, "cache.trenova", cfg.ServerName)
	require.NotNil(t, cfg.RootCAs)
}

func TestTLSClientConfig_RefusesAnAuthorityFileWithoutACertificate(t *testing.T) {
	t.Parallel()

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, []byte("not a certificate"), 0o600))

	_, err := (&Config{TLS: TLSConfig{Enabled: true, CAFile: caFile}}).TLSClientConfig()

	require.Error(t, err)
}

func writeSelfSignedCA(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(
		path,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		0o600,
	))
	return path
}
