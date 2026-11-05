package credential

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
)

var (
	ErrBundleInvalid   = errors.New("the enrolment bundle is not valid")
	ErrIdentityInvalid = errors.New("the stored identity is not valid")
)

type Bundle struct {
	InstanceID  string    `json:"instanceId"`
	APIURL      string    `json:"apiUrl"`
	Certificate string    `json:"certificate"`
	PrivateKey  string    `json:"privateKey"`
	CABundle    string    `json:"caBundle"`
	NotAfter    time.Time `json:"notAfter"`
}

type Identity struct {
	InstanceID   uuid.UUID
	APIURL       string
	CertPEM      []byte
	KeyPEM       []byte
	CAPEM        []byte
	EnrolledFrom string
	pair         tls.Certificate
	roots        *x509.CertPool
}

func ParseBundle(body []byte) (Bundle, error) {
	var b Bundle
	if err := sonic.Unmarshal(body, &b); err != nil {
		return Bundle{}, fmt.Errorf("%w: %w", ErrBundleInvalid, err)
	}
	return b, nil
}

func (b *Bundle) Identity() (Identity, error) {
	id, err := uuid.Parse(b.InstanceID)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: instanceId: %w", ErrBundleInvalid, err)
	}
	ident, err := NewIdentity(&IdentityParams{
		InstanceID: id,
		APIURL:     b.APIURL,
		CertPEM:    []byte(b.Certificate),
		KeyPEM:     []byte(b.PrivateKey),
		CAPEM:      []byte(b.CABundle),
	})
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %w", ErrBundleInvalid, err)
	}
	ident.EnrolledFrom = ident.Fingerprint()
	return ident, nil
}

type IdentityParams struct {
	InstanceID   uuid.UUID
	APIURL       string
	CertPEM      []byte
	KeyPEM       []byte
	CAPEM        []byte
	EnrolledFrom string
}

func NewIdentity(p *IdentityParams) (Identity, error) {
	u, err := url.Parse(p.APIURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return Identity{}, fmt.Errorf("apiUrl %q must be an https URL", p.APIURL)
	}
	pair, err := tls.X509KeyPair(p.CertPEM, p.KeyPEM)
	if err != nil {
		return Identity{}, fmt.Errorf("certificate and key: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(p.CAPEM) {
		return Identity{}, errors.New("caBundle holds no certificate")
	}
	if want := "urn:trenova:instance:" + p.InstanceID.String(); !namesInstance(pair.Leaf, want) {
		return Identity{}, fmt.Errorf("the certificate does not name %s", want)
	}
	return Identity{
		InstanceID:   p.InstanceID,
		APIURL:       strings.TrimSuffix(p.APIURL, "/"),
		CertPEM:      p.CertPEM,
		KeyPEM:       p.KeyPEM,
		CAPEM:        p.CAPEM,
		EnrolledFrom: p.EnrolledFrom,
		pair:         pair,
		roots:        roots,
	}, nil
}

func namesInstance(cert *x509.Certificate, uri string) bool {
	for _, u := range cert.URIs {
		if u.String() == uri {
			return true
		}
	}
	return false
}

func (i *Identity) NotBefore() time.Time { return i.pair.Leaf.NotBefore }

func (i *Identity) NotAfter() time.Time { return i.pair.Leaf.NotAfter }

func (i *Identity) Fingerprint() string {
	sum := sha256.Sum256(i.pair.Leaf.Raw)
	return hex.EncodeToString(sum[:])
}

func (i *Identity) RenewAt() time.Time {
	life := i.NotAfter().Sub(i.NotBefore())
	return i.NotBefore().Add(life - life/3)
}

func (i *Identity) TLSConfig() (*tls.Config, error) {
	u, err := url.Parse(i.APIURL)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{i.pair},
		RootCAs:      i.roots,
		ServerName:   u.Hostname(),
	}, nil
}

type NewKey struct {
	Key    *ecdsa.PrivateKey
	KeyPEM []byte
	CSRPEM []byte
}

func GenerateKey(instanceID uuid.UUID) (NewKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return NewKey{}, fmt.Errorf("generate key: %w", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return NewKey{}, fmt.Errorf("encode key: %w", err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: "trenova-operator " + instanceID.String()},
	}, key)
	if err != nil {
		return NewKey{}, fmt.Errorf("create certificate request: %w", err)
	}
	return NewKey{
		Key:    key,
		KeyPEM: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}),
		CSRPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr}),
	}, nil
}
