package inboundfixtures

import (
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/seeds/fixtures/triagefixtures"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The seeded mailbox used to be written without a signing secret, and a mailbox
with no secret refuses every delivery — so the one mailbox a developer has could
never receive mail. The seed now seals the published development secret with
the configured key, and a delivery signed with that secret verifies.
*/
func TestMailboxSigningSecret_SealsTheDevelopmentSecretSoASignedDeliveryVerifies(t *testing.T) {
	enc := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("a-development-encryption-key-of-32+-chars"),
	)

	mailbox := &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	sealed, err := mailboxSigningSecret(enc, mailbox)
	require.NoError(t, err)
	require.NotEmpty(t, sealed)
	assert.NotEqual(
		t,
		triagefixtures.SeedInboundMailboxSigningSecret,
		sealed,
		"the secret is stored sealed, not in the clear",
	)

	opened, err := enc.DecryptStringWithAAD(sealed, encryptionservice.InboundMailboxSecretAAD(
		mailbox.OrganizationID,
		mailbox.BusinessUnitID,
		mailbox.ID,
		inboundmessage.SigningSecretField,
	))
	require.NoError(t, err, "the secret is bound to the mailbox the seed wrote it on")
	assert.Equal(t, triagefixtures.SeedInboundMailboxSigningSecret, opened)

	now := time.Unix(1_790_000_000, 0)
	stamp := strconv.FormatInt(now.Unix(), 10)
	body := []byte(`{"type":"email.received"}`)
	signature, err := webhooksig.SignSvix(opened, "msg_seed", stamp, body)
	require.NoError(t, err)
	assert.NoError(t, webhooksig.VerifySvix(webhooksig.SvixParams{
		Secret: opened, ID: "msg_seed", Timestamp: stamp, Signature: signature, Body: body, Now: now,
	}))
}

// With no key configured there is nothing to seal with. The mailbox is still
// seeded, and still refuses deliveries — the safe default — rather than the
// seed failing or storing the secret in the clear.
func TestMailboxSigningSecret_LeavesItUnsetWhenEncryptionIsOff(t *testing.T) {
	sealed, err := mailboxSigningSecret(
		encryptionservice.NewWithKeyManager(encryptionservice.DisabledKeyManager{}),
		&inboundmessage.Mailbox{ID: pulid.MustNew("imbx_")},
	)

	require.NoError(t, err)
	assert.Empty(t, sealed)
}
