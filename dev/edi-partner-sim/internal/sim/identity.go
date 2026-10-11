package sim

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

type Identity struct {
	Certificate    *x509.Certificate
	Key            *rsa.PrivateKey
	CertificatePEM string
	KeyPEM         string
}

func NewIdentity(commonName string) (*Identity, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate simulator key: %w", err)
	}
	return identityFromKey(commonName, key)
}

// LoadOrCreateIdentity loads the AS2 certificate/key from dir if present, otherwise
// generates a fresh identity and persists it so restarts keep the same keypair (and
// therefore keep working against communication profiles created in an earlier run).
func LoadOrCreateIdentity(dir, commonName string) (*Identity, error) {
	if dir == "" {
		return NewIdentity(commonName)
	}
	certPath := filepath.Join(dir, "as2-cert.pem")
	keyPath := filepath.Join(dir, "as2-key.pem")

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		identity, err := identityFromPEM(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("load persisted AS2 identity: %w", err)
		}
		return identity, nil
	}

	identity, err := NewIdentity(commonName)
	if err != nil {
		return nil, err
	}
	if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
		return nil, fmt.Errorf("create identity directory: %w", mkErr)
	}
	if writeErr := os.WriteFile(certPath, []byte(identity.CertificatePEM), 0o600); writeErr != nil {
		return nil, fmt.Errorf("persist AS2 certificate: %w", writeErr)
	}
	if writeErr := os.WriteFile(keyPath, []byte(identity.KeyPEM), 0o600); writeErr != nil {
		return nil, fmt.Errorf("persist AS2 key: %w", writeErr)
	}
	return identity, nil
}

func identityFromKey(commonName string, key *rsa.PrivateKey) (*Identity, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate certificate serial: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Trenova EDI Partner Simulator"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create simulator certificate: %w", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse simulator certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal simulator key: %w", err)
	}
	return &Identity{
		Certificate:    certificate,
		Key:            key,
		CertificatePEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		KeyPEM:         string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
	}, nil
}

func identityFromPEM(certPEM, keyPEM []byte) (*Identity, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, errors.New("certificate PEM is invalid")
	}
	certificate, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, errors.New("key PEM is invalid")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("persisted key is not an RSA private key")
	}
	return &Identity{
		Certificate:    certificate,
		Key:            key,
		CertificatePEM: string(certPEM),
		KeyPEM:         string(keyPEM),
	}, nil
}
