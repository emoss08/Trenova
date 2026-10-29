package as2

import (
	"testing"

	"github.com/emoss08/trenova/shared/smime"
	"github.com/emoss08/trenova/shared/smimetest"
	"github.com/stretchr/testify/require"
)

func TestBuildAndParseMessageWithEveryEncryptionAlgorithm(t *testing.T) {
	t.Parallel()

	sender := smimetest.NewIdentity(t, "sender.example")
	receiver := smimetest.NewIdentity(t, "receiver.example")
	for _, algorithm := range []string{
		smime.EncryptionAlgorithmAES128CBC,
		smime.EncryptionAlgorithmAES128GCM,
		smime.EncryptionAlgorithmAES256CBC,
		smime.EncryptionAlgorithmAES256GCM,
		smime.EncryptionAlgorithmTripleDES,
	} {
		t.Run(algorithm, func(t *testing.T) {
			t.Parallel()

			built, err := BuildMessage(&BuildMessageOptions{
				From:                  "SENDER-AS2",
				To:                    "RECEIVER-AS2",
				FileName:              "shipment.edi",
				Payload:               []byte(smimetest.SamplePayload),
				SigningCertificate:    sender.Certificate,
				SigningKey:            sender.Key,
				EncryptionCertificate: receiver.Certificate,
				EncryptionAlgorithm:   algorithm,
			})
			require.NoError(t, err)

			parsed, err := ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
				DecryptionCertificate: receiver.Certificate,
				DecryptionKey:         receiver.Key,
				PartnerCertificate:    sender.Certificate,
				TransferEncoding:      built.Headers.Get("Content-Transfer-Encoding"),
				RequireSignature:      true,
				RequireEncryption:     true,
			})
			require.NoError(t, err)
			require.True(t, parsed.Encrypted)
			require.Equal(t, smimetest.SamplePayload, string(parsed.Payload))
		})
	}
}
