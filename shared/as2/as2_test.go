package as2

import (
	"testing"

	"github.com/emoss08/trenova/shared/smimetest"
	"github.com/stretchr/testify/require"
)

func TestBuildAndParseSignedEncryptedMessage(t *testing.T) {
	t.Parallel()

	sender := smimetest.NewIdentity(t, "sender.example")
	receiver := smimetest.NewIdentity(t, "receiver.example")

	built, err := BuildMessage(&BuildMessageOptions{
		From:                  "SENDER-AS2",
		To:                    "RECEIVER-AS2",
		FileName:              "shipment.edi",
		Payload:               []byte(smimetest.SamplePayload),
		SigningCertificate:    sender.Certificate,
		SigningKey:            sender.Key,
		EncryptionCertificate: receiver.Certificate,
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
		DecryptionCertificate: receiver.Certificate,
		DecryptionKey:         receiver.Key,
		PartnerCertificate:    sender.Certificate,
		TransferEncoding:      built.Headers.Get("Content-Transfer-Encoding"),
		RequireSignature:      true,
		RequireEncryption:     true,
	})
	require.NoError(t, err)
	require.True(t, parsed.Signed)
	require.True(t, parsed.Encrypted)
	require.Equal(t, smimetest.SamplePayload, string(parsed.Payload))
	require.Equal(t, "shipment.edi", parsed.FileName)
	require.True(t, MICMatches(built.MIC, parsed.MIC))
}

func TestBuildAndParseCompressedSignedEncryptedMessage(t *testing.T) {
	t.Parallel()

	sender := smimetest.NewIdentity(t, "sender.example")
	receiver := smimetest.NewIdentity(t, "receiver.example")

	built, err := BuildMessage(&BuildMessageOptions{
		From:                  "SENDER-AS2",
		To:                    "RECEIVER-AS2",
		Payload:               []byte(smimetest.SamplePayload),
		SigningCertificate:    sender.Certificate,
		SigningKey:            sender.Key,
		EncryptionCertificate: receiver.Certificate,
		Compress:              true,
	})
	require.NoError(t, err)

	parsed, err := ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		DecryptionCertificate: receiver.Certificate,
		DecryptionKey:         receiver.Key,
		PartnerCertificate:    sender.Certificate,
		TransferEncoding:      built.Headers.Get("Content-Transfer-Encoding"),
	})
	require.NoError(t, err)
	require.True(t, parsed.Compressed)
	require.Equal(t, smimetest.SamplePayload, string(parsed.Payload))
}

func TestParseMessageRejectsTamperedBody(t *testing.T) {
	t.Parallel()

	sender := smimetest.NewIdentity(t, "sender.example")

	built, err := BuildMessage(&BuildMessageOptions{
		From:               "SENDER-AS2",
		To:                 "RECEIVER-AS2",
		Payload:            []byte(smimetest.SamplePayload),
		SigningCertificate: sender.Certificate,
		SigningKey:         sender.Key,
	})
	require.NoError(t, err)

	tampered := []byte(string(built.Body))
	tampered = []byte(replaceOnce(string(tampered), "SHIP1", "SHIP2"))

	_, err = ParseMessage(built.ContentType, tampered, &ParseMessageOptions{
		PartnerCertificate: sender.Certificate,
	})
	require.Error(t, err)
}

func TestParseMessageRejectsWrongSigner(t *testing.T) {
	t.Parallel()

	sender := smimetest.NewIdentity(t, "sender.example")
	impostor := smimetest.NewIdentity(t, "impostor.example")

	built, err := BuildMessage(&BuildMessageOptions{
		From:               "SENDER-AS2",
		To:                 "RECEIVER-AS2",
		Payload:            []byte(smimetest.SamplePayload),
		SigningCertificate: sender.Certificate,
		SigningKey:         sender.Key,
	})
	require.NoError(t, err)

	_, err = ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		PartnerCertificate: impostor.Certificate,
	})
	require.Error(t, err)
}

func TestParseMessageRequiresConfiguredLayers(t *testing.T) {
	t.Parallel()

	built, err := BuildMessage(&BuildMessageOptions{
		From:    "SENDER-AS2",
		To:      "RECEIVER-AS2",
		Payload: []byte(smimetest.SamplePayload),
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
	require.Equal(t, smimetest.SamplePayload, string(parsed.Payload))
	require.NotEmpty(t, parsed.MIC)
}

func TestComputeMICIsDeterministic(t *testing.T) {
	t.Parallel()

	first, err := ComputeMIC([]byte(smimetest.SamplePayload), MICAlgorithmSHA256)
	require.NoError(t, err)
	second, err := ComputeMIC([]byte(smimetest.SamplePayload), MICAlgorithmSHA256)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Contains(t, first, ", sha256")

	different, err := ComputeMIC([]byte(smimetest.SamplePayload+"~"), MICAlgorithmSHA256)
	require.NoError(t, err)
	require.False(t, MICMatches(first, different))
}

func TestBuildAndParseSignedMDN(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")

	built, err := BuildMDN(&BuildMDNOptions{
		From:               "RECEIVER-AS2",
		To:                 "SENDER-AS2",
		OriginalMessageID:  "<abc123@trenova.as2>",
		ReceivedContentMIC: "q1w2e3r4, sha256",
		SigningCertificate: receiver.Certificate,
		SigningKey:         receiver.Key,
	})
	require.NoError(t, err)
	require.True(t, IsMDNContentType(built.ContentType) ||
		builtMDNIsSigned(built.ContentType))

	parsed, err := ParseMDN(built.ContentType, built.Body, receiver.Certificate)
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
