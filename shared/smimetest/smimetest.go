package smimetest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type Identity struct {
	Certificate *x509.Certificate
	Key         *rsa.PrivateKey
}

func NewIdentity(t *testing.T, commonName string) *Identity {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment |
			x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return &Identity{Certificate: certificate, Key: key}
}

const SamplePayload = "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       " +
	"*260107*1200*^*00401*000000001*0*P*>~GS*SM*SENDER*RECEIVER*20260107*1200*1*X*004010~" +
	"ST*204*0001~B2**SCAC**SHIP1**PP~SE*3*0001~GE*1*1~IEA*1*000000001~"
