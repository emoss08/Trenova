package smime

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/emoss08/trenova/shared/smimetest"
	"github.com/stretchr/testify/require"
)

func TestCompressRoundTrip(t *testing.T) {
	t.Parallel()

	compressed, err := Compress([]byte(smimetest.SamplePayload))
	require.NoError(t, err)
	require.True(t, IsCompressedData(compressed))

	decompressed, err := Decompress(compressed, int64(len(smimetest.SamplePayload)))
	require.NoError(t, err)
	require.Equal(t, smimetest.SamplePayload, string(decompressed))
}

func TestDecompressRefusesContentOverTheLimit(t *testing.T) {
	t.Parallel()

	compressed, err := Compress(bytes.Repeat([]byte("x"), 4096))
	require.NoError(t, err)

	_, err = Decompress(compressed, 1024)
	require.ErrorIs(t, err, ErrDecompressedTooLarge)
}

func TestImplicitRejectionHidesWhetherTheKeyTransportWasWellFormed(t *testing.T) {
	t.Parallel()

	identity := smimetest.NewIdentity(t, "receiver.example")
	key := implicitRejectionKey{Decrypter: identity.Key, sessionKeyLen: 32}
	options := &rsa.PKCS1v15DecryptOptions{}
	sessionKey := bytes.Repeat([]byte{7}, 32)

	wellFormed, err := rsa.EncryptPKCS1v15(rand.Reader, &identity.Key.PublicKey, sessionKey)
	require.NoError(t, err)
	wrongLength, err := rsa.EncryptPKCS1v15(rand.Reader, &identity.Key.PublicKey, []byte("short"))
	require.NoError(t, err)
	notPadded := bytes.Repeat([]byte{1}, identity.Key.Size())

	recovered, err := key.Decrypt(rand.Reader, wellFormed, options)
	require.NoError(t, err)
	require.Equal(t, sessionKey, recovered)

	for _, ciphertext := range [][]byte{wrongLength, notPadded} {
		substituted, decryptErr := key.Decrypt(rand.Reader, ciphertext, options)
		require.NoError(t, decryptErr, "bad padding must not be an error anyone can observe")
		require.Len(t, substituted, 32)
		require.NotEqual(t, sessionKey, substituted)
	}
}

func TestImplicitRejectionAnswersARepeatedCiphertextTheSameWay(t *testing.T) {
	t.Parallel()

	identity := smimetest.NewIdentity(t, "receiver.example")
	key := implicitRejectionKey{Decrypter: identity.Key, sessionKeyLen: 32}
	options := &rsa.PKCS1v15DecryptOptions{}
	notPadded := bytes.Repeat([]byte{1}, identity.Key.Size())
	otherNotPadded := bytes.Repeat([]byte{2}, identity.Key.Size())

	first, err := key.Decrypt(rand.Reader, notPadded, options)
	require.NoError(t, err)
	second, err := key.Decrypt(rand.Reader, notPadded, options)
	require.NoError(t, err)
	other, err := key.Decrypt(rand.Reader, otherNotPadded, options)
	require.NoError(t, err)

	require.Equal(t, first, second, "a varying answer would mark the ciphertext as badly padded")
	require.NotEqual(t, first, other)
}

func TestImplicitRejectionDependsOnThePrivateKey(t *testing.T) {
	t.Parallel()

	one := smimetest.NewIdentity(t, "one.example")
	two := smimetest.NewIdentity(t, "two.example")
	options := &rsa.PKCS1v15DecryptOptions{}
	notPadded := bytes.Repeat([]byte{1}, one.Key.Size())

	first, err := implicitRejectionKey{Decrypter: one.Key, sessionKeyLen: 32}.
		Decrypt(rand.Reader, notPadded, options)
	require.NoError(t, err)
	second, err := implicitRejectionKey{Decrypter: two.Key, sessionKeyLen: 32}.
		Decrypt(rand.Reader, notPadded, options)
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestDecryptReportsTheSameErrorForAKeyThatDoesNotOpenTheMessage(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	stranger := smimetest.NewIdentity(t, "stranger.example")
	ciphertext, err := Encrypt(
		[]byte(smimetest.SamplePayload),
		receiver.Certificate,
		EncryptionAlgorithmAES256GCM,
	)
	require.NoError(t, err)

	_, err = Decrypt(ciphertext, receiver.Certificate, stranger.Key)

	require.Error(t, err)
	require.NotContains(t, err.Error(), "crypto/rsa")
}

func TestDecryptSurvivesAKeyThatTurnsCBCContentToGarbage(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	stranger := smimetest.NewIdentity(t, "stranger.example")
	ciphertext, err := Encrypt([]byte("x"), receiver.Certificate, EncryptionAlgorithmAES256CBC)
	require.NoError(t, err)

	for range 64 {
		require.NotPanics(t, func() {
			_, _ = Decrypt(ciphertext, receiver.Certificate, stranger.Key)
		})
	}
}

func TestParsePEMHelpers(t *testing.T) {
	t.Parallel()

	identity := smimetest.NewIdentity(t, "pem.example")
	certificatePEM := encodeCertificatePEM(t, identity.Certificate)
	keyPEM := encodePKCS8PEM(t, identity.Key)

	certificate, err := ParseCertificate(certificatePEM)
	require.NoError(t, err)
	require.True(t, certificate.Equal(identity.Certificate))

	key, err := ParsePrivateKey(keyPEM)
	require.NoError(t, err)
	require.IsType(t, &rsa.PrivateKey{}, key)

	_, err = ParseCertificate([]byte("not pem"))
	require.ErrorIs(t, err, ErrNoPEMBlock)
	_, err = ParsePrivateKey([]byte("not pem"))
	require.ErrorIs(t, err, ErrUnsupportedKeyType)
}

func encodeCertificatePEM(t *testing.T, certificate *x509.Certificate) []byte {
	t.Helper()
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
}

func encodePKCS8PEM(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}
