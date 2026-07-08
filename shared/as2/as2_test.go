package as2

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testIdentity struct {
	certificate *x509.Certificate
	key         *rsa.PrivateKey
}

func newTestIdentity(t *testing.T, commonName string) *testIdentity {
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
	return &testIdentity{certificate: certificate, key: key}
}

const testPayload = "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       " +
	"*260107*1200*^*00401*000000001*0*P*>~GS*SM*SENDER*RECEIVER*20260107*1200*1*X*004010~" +
	"ST*204*0001~B2**SCAC**SHIP1**PP~SE*3*0001~GE*1*1~IEA*1*000000001~"

func TestBuildAndParseSignedEncryptedMessage(t *testing.T) {
	t.Parallel()

	sender := newTestIdentity(t, "sender.example")
	receiver := newTestIdentity(t, "receiver.example")

	built, err := BuildMessage(&BuildMessageOptions{
		From:                  "SENDER-AS2",
		To:                    "RECEIVER-AS2",
		FileName:              "shipment.edi",
		Payload:               []byte(testPayload),
		SigningCertificate:    sender.certificate,
		SigningKey:            sender.key,
		EncryptionCertificate: receiver.certificate,
		RequestMDN:            true,
		RequestSignedMDN:      true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, built.MIC)
	require.NotEmpty(t, built.MessageID)
	require.Contains(t, built.ContentType, "application/pkcs7-mime")
	require.Contains(t, built.ContentType, "enveloped-data")
	require.Equal(t, "SENDER-AS2", built.Headers.Get(HeaderAS2From))
	require.Equal(t, "RECEIVER-AS2", built.Headers.Get(HeaderAS2To))
	require.NotEmpty(t, built.Headers.Get(HeaderDispositionNotificationTo))
	require.NotContains(t, string(built.Body), "ISA*00")

	parsed, err := ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		DecryptionCertificate: receiver.certificate,
		DecryptionKey:         receiver.key,
		PartnerCertificate:    sender.certificate,
		TransferEncoding:      built.Headers.Get("Content-Transfer-Encoding"),
		RequireSignature:      true,
		RequireEncryption:     true,
	})
	require.NoError(t, err)
	require.True(t, parsed.Signed)
	require.True(t, parsed.Encrypted)
	require.Equal(t, testPayload, string(parsed.Payload))
	require.Equal(t, "shipment.edi", parsed.FileName)
	require.True(t, MICMatches(built.MIC, parsed.MIC))
}

func TestBuildAndParseCompressedSignedEncryptedMessage(t *testing.T) {
	t.Parallel()

	sender := newTestIdentity(t, "sender.example")
	receiver := newTestIdentity(t, "receiver.example")

	built, err := BuildMessage(&BuildMessageOptions{
		From:                  "SENDER-AS2",
		To:                    "RECEIVER-AS2",
		Payload:               []byte(testPayload),
		SigningCertificate:    sender.certificate,
		SigningKey:            sender.key,
		EncryptionCertificate: receiver.certificate,
		Compress:              true,
	})
	require.NoError(t, err)

	parsed, err := ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		DecryptionCertificate: receiver.certificate,
		DecryptionKey:         receiver.key,
		PartnerCertificate:    sender.certificate,
		TransferEncoding:      built.Headers.Get("Content-Transfer-Encoding"),
	})
	require.NoError(t, err)
	require.True(t, parsed.Compressed)
	require.Equal(t, testPayload, string(parsed.Payload))
}

func TestParseMessageRejectsTamperedBody(t *testing.T) {
	t.Parallel()

	sender := newTestIdentity(t, "sender.example")

	built, err := BuildMessage(&BuildMessageOptions{
		From:               "SENDER-AS2",
		To:                 "RECEIVER-AS2",
		Payload:            []byte(testPayload),
		SigningCertificate: sender.certificate,
		SigningKey:         sender.key,
	})
	require.NoError(t, err)

	tampered := []byte(string(built.Body))
	tampered = []byte(replaceOnce(string(tampered), "SHIP1", "SHIP2"))

	_, err = ParseMessage(built.ContentType, tampered, &ParseMessageOptions{
		PartnerCertificate: sender.certificate,
	})
	require.Error(t, err)
}

func TestParseMessageRejectsWrongSigner(t *testing.T) {
	t.Parallel()

	sender := newTestIdentity(t, "sender.example")
	impostor := newTestIdentity(t, "impostor.example")

	built, err := BuildMessage(&BuildMessageOptions{
		From:               "SENDER-AS2",
		To:                 "RECEIVER-AS2",
		Payload:            []byte(testPayload),
		SigningCertificate: sender.certificate,
		SigningKey:         sender.key,
	})
	require.NoError(t, err)

	_, err = ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		PartnerCertificate: impostor.certificate,
	})
	require.Error(t, err)
}

func TestParseMessageRequiresConfiguredLayers(t *testing.T) {
	t.Parallel()

	built, err := BuildMessage(&BuildMessageOptions{
		From:    "SENDER-AS2",
		To:      "RECEIVER-AS2",
		Payload: []byte(testPayload),
	})
	require.NoError(t, err)

	_, err = ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		RequireSignature: true,
	})
	require.ErrorIs(t, err, ErrSignatureRequired)

	_, err = ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		RequireEncryption: true,
	})
	require.ErrorIs(t, err, ErrEncryptionRequired)

	parsed, err := ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{})
	require.NoError(t, err)
	require.Equal(t, testPayload, string(parsed.Payload))
	require.NotEmpty(t, parsed.MIC)
}

func TestComputeMICIsDeterministic(t *testing.T) {
	t.Parallel()

	first, err := ComputeMIC([]byte(testPayload), MICAlgorithmSHA256)
	require.NoError(t, err)
	second, err := ComputeMIC([]byte(testPayload), MICAlgorithmSHA256)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Contains(t, first, ", sha256")

	different, err := ComputeMIC([]byte(testPayload+"~"), MICAlgorithmSHA256)
	require.NoError(t, err)
	require.False(t, MICMatches(first, different))
}

func TestCompressRoundTrip(t *testing.T) {
	t.Parallel()

	compressed, err := Compress([]byte(testPayload))
	require.NoError(t, err)
	require.True(t, IsCompressedData(compressed))

	decompressed, err := Decompress(compressed)
	require.NoError(t, err)
	require.Equal(t, testPayload, string(decompressed))
}

func TestBuildAndParseSignedMDN(t *testing.T) {
	t.Parallel()

	receiver := newTestIdentity(t, "receiver.example")

	built, err := BuildMDN(&BuildMDNOptions{
		From:               "RECEIVER-AS2",
		To:                 "SENDER-AS2",
		OriginalMessageID:  "<abc123@trenova.as2>",
		ReceivedContentMIC: "q1w2e3r4, sha256",
		SigningCertificate: receiver.certificate,
		SigningKey:         receiver.key,
	})
	require.NoError(t, err)
	require.True(t, IsMDNContentType(built.ContentType) ||
		builtMDNIsSigned(built.ContentType))

	parsed, err := ParseMDN(built.ContentType, built.Body, receiver.certificate)
	require.NoError(t, err)
	require.True(t, parsed.Signed)
	require.True(t, parsed.Processed())
	require.Equal(t, "<abc123@trenova.as2>", parsed.OriginalMessageID)
	require.Equal(t, "q1w2e3r4, sha256", parsed.ReceivedContentMIC)
}

func TestBuildAndParseErrorMDN(t *testing.T) {
	t.Parallel()

	built, err := BuildMDN(&BuildMDNOptions{
		From:              "RECEIVER-AS2",
		To:                "SENDER-AS2",
		OriginalMessageID: "<abc123@trenova.as2>",
		ErrorText:         "decryption failed",
	})
	require.NoError(t, err)
	require.True(t, IsMDNContentType(built.ContentType))

	parsed, err := ParseMDN(built.ContentType, built.Body, nil)
	require.NoError(t, err)
	require.False(t, parsed.Processed())
	require.Contains(t, parsed.FailureText(), "decryption failed")
}

func TestParsePEMHelpers(t *testing.T) {
	t.Parallel()

	identity := newTestIdentity(t, "pem.example")
	certificatePEM := encodeCertificatePEM(t, identity.certificate)
	keyPEM := encodePKCS8PEM(t, identity.key)

	certificate, err := ParseCertificate(certificatePEM)
	require.NoError(t, err)
	require.True(t, certificate.Equal(identity.certificate))

	key, err := ParsePrivateKey(keyPEM)
	require.NoError(t, err)
	require.IsType(t, &rsa.PrivateKey{}, key)

	_, err = ParseCertificate([]byte("not pem"))
	require.ErrorIs(t, err, ErrNoPEMBlock)
	_, err = ParsePrivateKey([]byte("not pem"))
	require.ErrorIs(t, err, ErrUnsupportedKeyType)
}

func builtMDNIsSigned(contentType string) bool {
	return len(contentType) >= len(contentTypeMultipartSigned) &&
		contentType[:len(contentTypeMultipartSigned)] == contentTypeMultipartSigned
}

func replaceOnce(input, old, replacement string) string {
	for index := 0; index+len(old) <= len(input); index++ {
		if input[index:index+len(old)] == old {
			return input[:index] + replacement + input[index+len(old):]
		}
	}
	return input
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
