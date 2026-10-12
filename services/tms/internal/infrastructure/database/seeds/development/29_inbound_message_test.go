package development

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const inboundSeedTestNow = int64(1_791_900_000)

func inboundRefs(t *testing.T) (*inboundSeedRefs, *seedMailboxes) {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	person := func(name string) *tenant.User {
		return &tenant.User{ID: pulid.MustNew("usr_"), Name: name}
	}

	customers := make(map[string]*customer.Customer, 5)
	for _, code := range []string{"ACME", "FRSH", "GLBL", "RNGE", "PEAK"} {
		customers[code] = &customer.Customer{ID: pulid.MustNew("cus_"), Code: code, Name: code}
	}
	shipments := make(map[string]*shipment.Shipment, len(seedInboundShipmentPros))
	for _, pro := range seedInboundShipmentPros {
		shipments[pro] = &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: pro}
	}
	docTypes := map[string]*documenttype.DocumentType{
		"INVOICE": {ID: pulid.MustNew("dt_"), Code: "INVOICE", Name: "Invoice"},
		documenttype.CodePOD: {
			ID:   pulid.MustNew("dt_"),
			Code: documenttype.CodePOD,
			Name: "Proof of Delivery",
		},
		"RATECONF": {ID: pulid.MustNew("dt_"), Code: "RATECONF", Name: "Rate Confirmation"},
	}

	org := orgStub(orgID, buID)
	org.Timezone = "UTC"
	refs := &inboundSeedRefs{
		org:   org,
		admin: person("System Administrator"),
		team: seedTeam{
			dana:   person("Dana Whitfield"),
			marcus: person("Marcus Lee"),
			priya:  person("Priya Shah"),
		},
		customers: customers,
		shipments: shipments,
		docTypes:  docTypes,
		loc:       time.UTC,
		now:       inboundSeedTestNow,
		dayStart:  timeutils.DayStart(inboundSeedTestNow, time.UTC),
		weekStart: timeutils.WeekStart(inboundSeedTestNow, time.UTC),
	}

	return refs, seedMailboxesFor(refs, "sealed")
}

func seededTriage(
	t *testing.T,
) (*inboundSeedRefs, *seedMailboxes, []*triage.Rule, []seededMessage) {
	t.Helper()

	refs, boxes := inboundRefs(t)
	rules := seedRules(refs, boxes)
	templates := seedReplyTemplates(refs)
	messages := NewInboundMessageSeed().messages(refs, boxes, seedRuleSet(rules), templates)

	return refs, boxes, rules, messages
}

func TestInboundMessageSeed_EverythingItWritesIsValid(t *testing.T) {
	t.Parallel()

	refs, boxes, rules, messages := seededTriage(t)

	check := func(what string, validate func(*errortypes.MultiError)) {
		multiErr := errortypes.NewMultiError()
		validate(multiErr)
		assert.Falsef(t, multiErr.HasErrors(), "%s is not valid: %v", what, multiErr)
	}
	for _, mailbox := range []*inboundmessage.Mailbox{boxes.intake, boxes.billing, boxes.claims} {
		check("mailbox "+mailbox.Name, mailbox.Validate)
	}
	for _, rule := range rules {
		check("a rule", rule.Validate)
	}
	for _, template := range seedReplyTemplates(refs) {
		check("template "+template.Name, template.Validate)
	}
	for _, view := range seedViews(refs) {
		check("view "+view.Name, view.Validate)
	}
	check("the settings", triageSettingsFor(refs).Validate)

	providerIDs := make(map[string]struct{}, len(messages))
	for _, seeded := range messages {
		message := seeded.message
		check(fmt.Sprintf("message %q", message.Subject), message.Validate)
		_, dup := providerIDs[message.ProviderMessageID]
		assert.Falsef(t, dup, "provider id %s is seeded twice", message.ProviderMessageID)
		providerIDs[message.ProviderMessageID] = struct{}{}
		for _, attachment := range seeded.attachments {
			check("attachment "+attachment.FileName, attachment.Validate)
		}
		for _, reply := range seeded.replies {
			check("a reply", reply.Validate)
			assert.Equal(t, message.ID, reply.MessageID)
			assert.Equal(t, message.ThreadKey, reply.ThreadKey)
		}
		for _, activity := range seeded.activities {
			assert.Equal(t, message.ID, activity.MessageID, "an activity line names its message")
			assert.GreaterOrEqual(t, activity.CreatedAt, message.ReceivedAt)
		}
	}
}

func TestInboundMessageSeed_FillsEveryLane(t *testing.T) {
	t.Parallel()

	refs, _, _, messages := seededTriage(t)

	lanes := map[string]int{}
	events := map[inboundmessage.SnoozeEvent]bool{}
	for _, seeded := range messages {
		message := seeded.message
		switch {
		case message.IsWaiting():
			lanes["waiting"]++
			if message.OwnerID == refs.admin.ID {
				lanes["mine"]++
			}
			if message.Status == inboundmessage.StatusQuarantined {
				lanes["held"]++
			}
			if message.DueAt < refs.now {
				lanes["overdue"]++
			}
		case message.IsSnoozed():
			lanes["snoozed"]++
			events[message.SnoozeEvent] = true
		case message.IsAwaitingReply():
			lanes["them"]++
		case message.Status == inboundmessage.StatusActioned:
			lanes["handled"]++
		case message.Status == inboundmessage.StatusIgnored:
			lanes["ignored"]++
		}
	}

	for _, lane := range []string{
		"waiting", "mine", "held", "overdue", "snoozed", "them", "handled", "ignored",
	} {
		assert.Positivef(t, lanes[lane], "nothing is seeded in %s", lane)
	}
	assert.True(t, events[""], "no message is snoozed until a time")
	assert.True(t, events[inboundmessage.SnoozeEventShipmentDelivered])
	assert.True(t, events[inboundmessage.SnoozeEventDocumentAttached])
}

func TestInboundMessageSeed_OpenMessagesAreDueAndSettledOnesAreNot(t *testing.T) {
	t.Parallel()

	_, _, _, messages := seededTriage(t)

	for _, seeded := range messages {
		message := seeded.message
		if message.NeedsReview() {
			assert.Equalf(t,
				message.ReceivedAt+int64(triage.DefaultTargetMinutes(message.Classification))*60,
				message.DueAt, "%q is due by its kind's target", message.Subject)

			continue
		}
		assert.Zerof(t, message.DueAt, "%q is settled and due for nothing", message.Subject)
		assert.Emptyf(t, message.Parked, "%q is settled and parked", message.Subject)
	}
}

func TestInboundMessageSeed_EveryMatchCarriesItsReason(t *testing.T) {
	t.Parallel()

	_, _, _, messages := seededTriage(t)

	for _, seeded := range messages {
		message := seeded.message
		if !message.HasRecord() {
			continue
		}
		assert.NotEmptyf(t, message.MatchReason,
			"%q names a record with no reason given", message.Subject)
	}
}

func TestInboundMessageSeed_HasAnAttachmentThatCouldNotBeRead(t *testing.T) {
	t.Parallel()

	_, _, _, messages := seededTriage(t)

	var refused, read, split int
	for _, seeded := range messages {
		for _, attachment := range seeded.attachments {
			if attachment.FailureText != "" {
				refused++

				continue
			}
			read++
			if attachment.PageCount > 1 {
				split++
				assert.Len(t, attachment.Pages, attachment.PageCount)
			}
		}
	}

	assert.Positive(t, read, "no attachment was read, so the good path is not shown")
	assert.Positive(t, refused, "no attachment failed, so the failure path is not shown")
	assert.Positive(t, split, "no attachment has pages to split")
}

func TestInboundMessageSeed_NovaSettledTheDaysRoutineMailUnderRules(t *testing.T) {
	t.Parallel()

	refs, _, rules, messages := seededTriage(t)
	byID := make(map[pulid.ID]*triage.Rule, len(rules))
	for _, rule := range rules {
		byID[rule.ID] = rule
	}

	today := map[inboundmessage.NovaAction]int{}
	for _, seeded := range messages {
		message := seeded.message
		if message.HandledByRuleID.IsNil() {
			continue
		}
		rule := byID[message.HandledByRuleID]
		require.NotNil(t, rule)
		assert.Truef(t, rule.Matches(message), "%q was settled by a rule that does not match it",
			message.Subject)
		assert.True(t, message.Status.Terminal())
		if message.ReceivedAt >= refs.dayStart {
			today[message.HandledAction]++
		}
	}

	assert.GreaterOrEqual(t, today[inboundmessage.NovaActionFileOnShipment], seedTodayPODs)
	assert.GreaterOrEqual(t, today[inboundmessage.NovaActionFileAndQueuePay], seedTodayInvoices)
	assert.GreaterOrEqual(t, today[inboundmessage.NovaActionIgnore], seedTodayNewsletters)

	var paused int
	for _, rule := range rules {
		if !rule.Enabled {
			paused++
		}
	}
	assert.Equal(t, 4, len(rules))
	assert.Equal(t, 1, paused, "one of the four rules is paused")
}

func TestInboundMessageSeed_ThreadsAndRepliesHangTogether(t *testing.T) {
	t.Parallel()

	_, _, _, messages := seededTriage(t)

	threads := map[string]int{}
	statuses := map[triage.ReplyStatus]int{}
	for _, seeded := range messages {
		threads[seeded.message.ThreadKey]++
		for _, reply := range seeded.replies {
			statuses[reply.Status]++
			if reply.Sent() {
				assert.True(t, seeded.message.IsAwaitingReply(),
					"the message a reply went out on waits on them")
			}
		}
	}

	var multi int
	for _, count := range threads {
		if count > 1 {
			multi++
		}
	}
	assert.Equal(t, 2, multi, "two conversations run to more than one message")
	assert.Equal(t, 1, statuses[triage.ReplySent])
	assert.Equal(t, 1, statuses[triage.ReplyDraft])
}

func TestInboundMessageSeed_ThreeCorrectionsThisWeekAndClaimsIsQuiet(t *testing.T) {
	t.Parallel()

	refs, boxes, _, messages := seededTriage(t)

	var corrections int
	for _, seeded := range messages {
		assert.LessOrEqualf(t, seeded.message.ReceivedAt, refs.now,
			"%q arrives in the future", seeded.message.Subject)
		if seeded.message.MailboxID == boxes.claims.ID {
			assert.LessOrEqual(t, seeded.message.ReceivedAt, boxes.claims.LastReceivedAt)
		}
		for _, activity := range seeded.activities {
			if activity.Kind == triage.ActivityCorrected && activity.CreatedAt >= refs.weekStart {
				assert.Equal(t, triage.ActorPerson, activity.ActorKind)
				corrections++
			}
		}
	}

	assert.Equal(t, len(seedCorrectionsThisWeek), corrections)
	assert.Len(t, seedSenderHints(refs), len(seedCorrectionsThisWeek)+1)
	health := inboundmessage.HealthOf(inboundmessage.SourceHealthInput{
		Status:         boxes.claims.Status,
		CreatedAt:      refs.now - 30*inboundSeedDay,
		LastReceivedAt: boxes.claims.LastReceivedAt,
		Now:            refs.now,
	})
	assert.Equal(t, inboundmessage.SourceQuiet, health)
}

func TestInboundMessageSeed_StoresTheTokensHashed(t *testing.T) {
	t.Parallel()

	_, boxes := inboundRefs(t)

	tokens := map[*inboundmessage.Mailbox]string{
		boxes.intake:  SeedInboundMailboxToken,
		boxes.billing: seedBillingMailboxToken,
		boxes.claims:  seedClaimsMailboxToken,
	}
	for mailbox, token := range tokens {
		assert.NotEqual(t, token, mailbox.TokenHash)
		assert.Equal(t, hashutils.SHA256Hex(token), mailbox.TokenHash)
	}
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
