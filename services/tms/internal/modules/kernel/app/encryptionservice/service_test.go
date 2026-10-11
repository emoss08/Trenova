package encryptionservice

import (
	"context"
	"strings"
	"testing"
	"time"

	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	gax "github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/require"
)

func TestService_EnvelopeRoundTrip(t *testing.T) {
	t.Parallel()

	svc := testService()
	aad := testAAD()

	ciphertext, err := svc.EncryptBytesWithAAD([]byte("sensitive document bytes"), aad)
	require.NoError(t, err)
	require.True(t, IsEnvelope(ciphertext))
	require.NotContains(t, ciphertext, "sensitive document bytes")

	plaintext, err := svc.DecryptBytesWithAAD(ciphertext, aad)
	require.NoError(t, err)
	require.Equal(t, []byte("sensitive document bytes"), plaintext)
}

func TestService_EnvelopeRejectsWrongTenantAAD(t *testing.T) {
	t.Parallel()

	svc := testService()
	aad := testAAD()

	ciphertext, err := svc.EncryptStringWithAAD("client-secret", aad)
	require.NoError(t, err)

	wrongAAD := aad
	wrongAAD.OrganizationID = pulid.MustNew("org_")
	_, err = svc.DecryptStringWithAAD(ciphertext, wrongAAD)
	require.Error(t, err)
}

func TestService_EnvelopeRejectsTamper(t *testing.T) {
	t.Parallel()

	svc := testService()
	ciphertext, err := svc.EncryptStringWithAAD("client-secret", testAAD())
	require.NoError(t, err)

	replacement := "A"
	if strings.HasSuffix(ciphertext, "A") {
		replacement = "B"
	}
	tampered := ciphertext[:len(ciphertext)-1] + replacement
	_, err = svc.DecryptStringWithAAD(tampered, testAAD())
	require.Error(t, err)
}

func TestService_EnvelopeUsesRandomDEKAndNonce(t *testing.T) {
	t.Parallel()

	svc := testService()
	aad := testAAD()

	first, err := svc.EncryptStringWithAAD("same plaintext", aad)
	require.NoError(t, err)
	second, err := svc.EncryptStringWithAAD("same plaintext", aad)
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestService_RejectsUnencryptedCiphertext(t *testing.T) {
	t.Parallel()

	svc := testService()
	_, err := svc.DecryptStringWithAAD("legacy secret", testAAD())
	require.ErrorIs(t, err, ErrInvalidEnvelope)
}

func TestService_RewrapEnvelopeKeepsPlaintext(t *testing.T) {
	t.Parallel()

	svc := testService()
	aad := testAAD()
	ciphertext, err := svc.EncryptStringWithAAD("client-secret", aad)
	require.NoError(t, err)

	rewrapped, err := svc.RewrapEnvelopeWithAAD(ciphertext, aad)
	require.NoError(t, err)
	require.NotEqual(t, ciphertext, rewrapped)

	plaintext, err := svc.DecryptStringWithAAD(rewrapped, aad)
	require.NoError(t, err)
	require.Equal(t, "client-secret", plaintext)
}

func TestGCPAutokeyManagerUsesEnvelopeAAD(t *testing.T) {
	t.Parallel()

	client := &fakeKMSClient{}
	manager := NewGCPAutokeyManager(
		client,
		"projects/test/locations/us/keyRings/autokey/cryptoKeys/trenova",
		time.Second,
	)
	svc := NewWithKeyManager(manager)
	aad := testAAD()

	ciphertext, err := svc.EncryptStringWithAAD("client-secret", aad)
	require.NoError(t, err)
	require.True(t, IsEnvelope(ciphertext))
	require.Equal(t, aad.Bytes(), client.encryptAAD)

	plaintext, err := svc.DecryptStringWithAAD(ciphertext, aad)
	require.NoError(t, err)
	require.Equal(t, "client-secret", plaintext)
	require.Equal(t, aad.Bytes(), client.decryptAAD)
}

func testService() *Service {
	return New(Params{
		Config: &config.Config{
			Security: config.SecurityConfig{
				Encryption: config.EncryptionConfig{
					Key: "unit-test-encryption-key-with-at-least-32-bytes",
				},
			},
		},
	})
}

func testAAD() AAD {
	return AAD{
		Purpose:        PurposeDocument,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ResourceID:     "documents/test.pdf",
	}
}

type fakeKMSClient struct {
	encryptAAD []byte
	decryptAAD []byte
}

func (c *fakeKMSClient) Encrypt(
	_ context.Context,
	req *kmspb.EncryptRequest,
	_ ...gax.CallOption,
) (*kmspb.EncryptResponse, error) {
	c.encryptAAD = append([]byte(nil), req.AdditionalAuthenticatedData...)
	ciphertext := append([]byte("wrapped:"), req.Plaintext...)
	return &kmspb.EncryptResponse{Ciphertext: ciphertext}, nil
}

func (c *fakeKMSClient) Decrypt(
	_ context.Context,
	req *kmspb.DecryptRequest,
	_ ...gax.CallOption,
) (*kmspb.DecryptResponse, error) {
	c.decryptAAD = append([]byte(nil), req.AdditionalAuthenticatedData...)
	return &kmspb.DecryptResponse{
		Plaintext: []byte(strings.TrimPrefix(string(req.Ciphertext), "wrapped:")),
	}, nil
}

func (c *fakeKMSClient) Close() error {
	return nil
}
