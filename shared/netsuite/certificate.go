package netsuite

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/tokenutils"
)

const (
	AssertionLifetime      = 5 * time.Minute
	MaxCertificateValidity = 2 * 365 * 24 * time.Hour
	ClientAssertionType    = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	assertionAlgorithm     = "ES256"
	assertionType          = "JWT"
	maxCertificateIDLength = 100
	maxSubjectLength       = 64
	jtiBytes               = 16
	serialBits             = 128
	coordinateBytes        = 32
	pemPrivateKey          = "PRIVATE KEY"
	pemECPrivateKey        = "EC PRIVATE KEY"
	pemCertificate         = "CERTIFICATE"
)

type AssertionRequest struct {
	AccountID     string
	CertificateID string
	PrivateKey    *ecdsa.PrivateKey
	Now           time.Time
}

type CertificateGrant struct {
	AccountID     string
	CertificateID string
	PrivateKey    *ecdsa.PrivateKey
}

type CertificateRequest struct {
	Subject  string
	Now      time.Time
	Validity time.Duration
}

type Certificate struct {
	PrivateKeyPEM  []byte
	CertificatePEM []byte
	ExpiresAt      time.Time
}

type assertionHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

type assertionClaims struct {
	Iss   string `json:"iss"`
	Scope string `json:"scope"`
	Aud   string `json:"aud"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
	Jti   string `json:"jti"`
}

func (c *OAuthClient) ClientAssertion(req AssertionRequest) (string, error) {
	certificateID := strings.TrimSpace(req.CertificateID)
	if !validCertificateID(certificateID) {
		return "", ErrInvalidCertificate
	}
	if !isP256(req.PrivateKey) {
		return "", ErrInvalidPrivateKey
	}
	audience, err := c.TokenURL(req.AccountID)
	if err != nil {
		return "", err
	}
	now := req.Now
	if now.IsZero() {
		now = c.settings.now()
	}
	jti, err := tokenutils.RandomHex(jtiBytes)
	if err != nil {
		return "", fmt.Errorf("netsuite: generate assertion id: %w", err)
	}
	header, err := sonic.Marshal(assertionHeader{
		Alg: assertionAlgorithm,
		Typ: assertionType,
		Kid: certificateID,
	})
	if err != nil {
		return "", fmt.Errorf("netsuite: encode assertion header: %w", err)
	}
	claims, err := sonic.Marshal(assertionClaims{
		Iss:   c.cfg.ClientID,
		Scope: Scope,
		Aud:   audience,
		Iat:   now.Unix(),
		Exp:   now.Add(AssertionLifetime).Unix(),
		Jti:   jti,
	})
	if err != nil {
		return "", fmt.Errorf("netsuite: encode assertion claims: %w", err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(claims)
	signature, err := signES256(req.PrivateKey, signingInput)
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (c *OAuthClient) TokenFromCertificate(
	ctx context.Context,
	grant CertificateGrant,
) (*Token, error) {
	assertion, err := c.ClientAssertion(AssertionRequest{
		AccountID:     grant.AccountID,
		CertificateID: grant.CertificateID,
		PrivateKey:    grant.PrivateKey,
	})
	if err != nil {
		return nil, err
	}
	return c.requestToken(ctx, &tokenCall{
		accountID: grant.AccountID,
		endpoint:  "oauth-client-credentials",
		form: url.Values{
			formGrantType:           {grantClientCredential},
			"client_assertion_type": {ClientAssertionType},
			"client_assertion":      {assertion},
		},
		secrets: []string{assertion},
	})
}

func NewCertificate(req CertificateRequest) (*Certificate, error) {
	subject := strings.TrimSpace(req.Subject)
	if subject == "" || len(subject) > maxSubjectLength {
		return nil, ErrSubjectRequired
	}
	validity := req.Validity
	if validity == 0 {
		validity = MaxCertificateValidity
	}
	if validity < 0 || validity > MaxCertificateValidity {
		return nil, ErrInvalidValidity
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	notBefore := now.UTC().Truncate(time.Second)
	notAfter := notBefore.Add(validity)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("netsuite: generate key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), serialBits))
	if err != nil {
		return nil, fmt.Errorf("netsuite: generate serial number: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: subject},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("netsuite: create certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("netsuite: encode private key: %w", err)
	}
	return &Certificate{
		PrivateKeyPEM:  pem.EncodeToMemory(&pem.Block{Type: pemPrivateKey, Bytes: keyDER}),
		CertificatePEM: pem.EncodeToMemory(&pem.Block{Type: pemCertificate, Bytes: der}),
		ExpiresAt:      notAfter,
	}, nil
}

func ParsePrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, ErrInvalidPrivateKey
	}
	var key *ecdsa.PrivateKey
	switch block.Type {
	case pemPrivateKey:
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidPrivateKey, err)
		}
		ecKey, ok := parsed.(*ecdsa.PrivateKey)
		if !ok {
			return nil, ErrInvalidPrivateKey
		}
		key = ecKey
	case pemECPrivateKey:
		parsed, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidPrivateKey, err)
		}
		key = parsed
	default:
		return nil, ErrInvalidPrivateKey
	}
	if !isP256(key) {
		return nil, ErrInvalidPrivateKey
	}
	return key, nil
}

func signES256(key *ecdsa.PrivateKey, signingInput string) ([]byte, error) {
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		return nil, fmt.Errorf("netsuite: sign assertion: %w", err)
	}
	signature := make([]byte, 2*coordinateBytes)
	r.FillBytes(signature[:coordinateBytes])
	s.FillBytes(signature[coordinateBytes:])
	return signature, nil
}

func isP256(key *ecdsa.PrivateKey) bool {
	return key != nil && key.Curve == elliptic.P256()
}

func validCertificateID(value string) bool {
	if value == "" || len(value) > maxCertificateIDLength {
		return false
	}
	for idx := range len(value) {
		if value[idx] <= ' ' || value[idx] > '~' {
			return false
		}
	}
	return true
}
