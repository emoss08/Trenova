package as2

import (
	"testing"

	"github.com/emoss08/trenova/shared/smimetest"
	"github.com/stretchr/testify/require"
)

func signedMessage(t *testing.T, signer, receiver *smimetest.Identity) *BuiltMessage {
	t.Helper()

	built, err := BuildMessage(&BuildMessageOptions{
		From:                  "SENDER-AS2",
		To:                    "RECEIVER-AS2",
		Payload:               []byte(smimetest.SamplePayload),
		SigningCertificate:    signer.Certificate,
		SigningKey:            signer.Key,
		EncryptionCertificate: receiver.Certificate,
	})
	require.NoError(t, err)

	return built
}

func TestParseMessageRefusesASignatureItHasNoCertificateToVerify(t *testing.T) {
	t.Parallel()

	impostor := smimetest.NewIdentity(t, "impostor.example")
	receiver := smimetest.NewIdentity(t, "receiver.example")
	built := signedMessage(t, impostor, receiver)

	_, err := ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		DecryptionCertificate: receiver.Certificate,
		DecryptionKey:         receiver.Key,
		TransferEncoding:      built.Headers.Get("Content-Transfer-Encoding"),
		RequireSignature:      true,
	})

	require.ErrorIs(t, err, ErrNoPartnerCertificate)
}

func TestParseMessageDoesNotCallAnUnverifiedSignatureSigned(t *testing.T) {
	t.Parallel()

	impostor := smimetest.NewIdentity(t, "impostor.example")
	receiver := smimetest.NewIdentity(t, "receiver.example")
	built := signedMessage(t, impostor, receiver)

	parsed, err := ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		DecryptionCertificate: receiver.Certificate,
		DecryptionKey:         receiver.Key,
		TransferEncoding:      built.Headers.Get("Content-Transfer-Encoding"),
	})

	require.NoError(t, err)
	require.False(t, parsed.Signed)
}

func TestParseMessageRefusesASignerOtherThanThePartner(t *testing.T) {
	t.Parallel()

	partner := smimetest.NewIdentity(t, "partner.example")
	impostor := smimetest.NewIdentity(t, "impostor.example")
	receiver := smimetest.NewIdentity(t, "receiver.example")
	built := signedMessage(t, impostor, receiver)

	_, err := ParseMessage(built.ContentType, built.Body, &ParseMessageOptions{
		DecryptionCertificate: receiver.Certificate,
		DecryptionKey:         receiver.Key,
		PartnerCertificate:    partner.Certificate,
		TransferEncoding:      built.Headers.Get("Content-Transfer-Encoding"),
		RequireSignature:      true,
	})

	require.Error(t, err)
}
