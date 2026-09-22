package development

import (
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inboundRefs(t *testing.T) (*inboundSeedRefs, *inboundmessage.Mailbox) {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipments := make([]*shipment.Shipment, 0, 3)
	for i := range 3 {
		shipments = append(shipments, &shipment.Shipment{
			ID:        pulid.MustNew("shp_"),
			ProNumber: "S0000" + string(rune('1'+i)),
		})
	}

	refs := &inboundSeedRefs{
		org:   orgStub(orgID, buID),
		admin: userStub(),
		customers: []*customer.Customer{
			{ID: pulid.MustNew("cus_"), Name: "Northwind Foods"},
			{ID: pulid.MustNew("cus_"), Name: "Harborline Grocers"},
		},
		shipments: shipments,
		now:       1_800_000_000,
	}

	mailbox := &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "Intake",
		Address:        SeedInboundMailboxAddress,
		Provider:       inboundmessage.ProviderResend,
		TokenHash:      hashutils.SHA256Hex(SeedInboundMailboxToken),
		ReviewPolicy:   inboundmessage.ReviewBelowConfidence,
		MinConfidence:  0.75,
		Status:         inboundmessage.MailboxActive,
	}

	return refs, mailbox
}

func TestInboundMessageSeed_EveryMessageIsValid(t *testing.T) {
	refs, mailbox := inboundRefs(t)

	multiErr := errortypes.NewMultiError()
	mailbox.Validate(multiErr)
	require.Falsef(t, multiErr.HasErrors(), "mailbox is not valid: %v", multiErr)

	for _, seeded := range NewInboundMessageSeed().messages(refs, mailbox) {
		messageErr := errortypes.NewMultiError()
		seeded.message.Validate(messageErr)
		assert.Falsef(t, messageErr.HasErrors(),
			"message %q is not valid: %v", seeded.message.Subject, messageErr)
	}
}

// The inbox splits on lanes, so the seed is only useful if it fills them.
func TestInboundMessageSeed_FillsEveryLane(t *testing.T) {
	refs, mailbox := inboundRefs(t)

	seen := map[inboundmessage.Status]bool{}
	for _, seeded := range NewInboundMessageSeed().messages(refs, mailbox) {
		seen[seeded.message.Status] = true
	}

	for _, status := range []inboundmessage.Status{
		inboundmessage.StatusInReview,
		inboundmessage.StatusActioned,
		inboundmessage.StatusQuarantined,
	} {
		assert.Truef(t, seen[status], "no seeded message is %s", status)
	}
}

// A match without a reason is the thing the inbox was built not to show. Any
// seeded message that names a record has to say why it named it.
func TestInboundMessageSeed_EveryMatchCarriesItsReason(t *testing.T) {
	refs, mailbox := inboundRefs(t)

	for _, seeded := range NewInboundMessageSeed().messages(refs, mailbox) {
		message := seeded.message
		matched := message.MatchedShipmentID.IsNotNil() ||
			message.MatchedCustomerID.IsNotNil() ||
			message.MatchedCarrierID.IsNotNil()
		if !matched {
			continue
		}
		assert.NotEmptyf(t, message.MatchReason,
			"%q names a record with no reason given", message.Subject)
	}
}

// A file the pipeline could not read has to say so on its own row: its
// absence is what makes the message incomplete, and a blank row hides that.
func TestInboundMessageSeed_HasAnAttachmentThatCouldNotBeRead(t *testing.T) {
	refs, mailbox := inboundRefs(t)

	var refused, read int
	for _, seeded := range NewInboundMessageSeed().messages(refs, mailbox) {
		for _, attachment := range seeded.attachments {
			if attachment.FailureText != "" {
				refused++

				continue
			}
			read++
			assert.NotEqualf(t, inboundmessage.AttachmentUnknown, attachment.Kind,
				"%q was read but has no kind", attachment.FileName)
		}
	}

	assert.Positive(t, read, "no attachment was read, so the good path is not shown")
	assert.Positive(t, refused, "no attachment failed, so the failure path is not shown")
}

// The webhook token is stored hashed, never in the clear. A seed that wrote
// the plain token would teach the wrong thing about how the table works.
func TestInboundMessageSeed_StoresTheTokenHashed(t *testing.T) {
	_, mailbox := inboundRefs(t)

	assert.NotEqual(t, SeedInboundMailboxToken, mailbox.TokenHash)
	assert.Equal(t, hashutils.SHA256Hex(SeedInboundMailboxToken), mailbox.TokenHash)
}

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

	sealed, err := mailboxSigningSecret(enc)
	require.NoError(t, err)
	require.NotEmpty(t, sealed)
	assert.NotEqual(t, SeedInboundMailboxSigningSecret, sealed, "the secret is stored sealed, not in the clear")

	opened, err := enc.DecryptString(sealed)
	require.NoError(t, err)
	assert.Equal(t, SeedInboundMailboxSigningSecret, opened)

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
	sealed, err := mailboxSigningSecret(encryptionservice.NewWithKeyManager(encryptionservice.DisabledKeyManager{}))

	require.NoError(t, err)
	assert.Empty(t, sealed)
}
