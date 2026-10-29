package triagefixtures

import (
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/seeds/fixtures/fixturetest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const inboundSeedTestNow = int64(1_791_900_000)

func inboundRefs(t *testing.T) (*Refs, *Mailboxes) {
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
	shipments := make(map[string]*shipment.Shipment, len(ShipmentPros))
	for _, pro := range ShipmentPros {
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
		"BOL":      {ID: pulid.MustNew("dt_"), Code: "BOL", Name: "Bill of Lading"},
	}

	carriers := make(map[string]*carrier.Carrier, len(CarrierCodes))
	for _, code := range CarrierCodes {
		carriers[code] = &carrier.Carrier{ID: pulid.MustNew("car_"), Code: code, Name: code}
	}
	pickupAt := inboundSeedTestNow + inboundSeedDay

	org := fixturetest.OrgStub(orgID, buID)
	org.Timezone = "UTC"
	refs := &Refs{
		Org:   org,
		Admin: person("System Administrator"),
		Team: Team{
			Dana:   person("Dana Whitfield"),
			Marcus: person("Marcus Lee"),
			Priya:  person("Priya Shah"),
		},
		Customers: customers,
		Shipments: shipments,
		Broker: &customer.Customer{
			ID:   pulid.MustNew("cus_"),
			Code: BrokerCode,
			Name: "Blue Ridge",
		},
		Carriers: carriers,
		Driver: &worker.Worker{
			ID:        pulid.MustNew("wrk_"),
			FirstName: "Carlos",
			LastName:  "Rivera",
			Email:     DriverEmail,
		},
		Lane: &Lane{
			Shipment:         shipments[OpenLanePro],
			Origin:           "Los Angeles, CA",
			OriginCity:       "Los Angeles",
			OriginState:      "CA",
			Destination:      "Chicago, IL",
			DestinationCity:  "Chicago",
			DestinationState: "IL",
			PickupAt:         pickupAt,
		},
		DocTypes:  docTypes,
		Loc:       time.UTC,
		Now:       inboundSeedTestNow,
		DayStart:  timeutils.DayStart(inboundSeedTestNow, time.UTC),
		WeekStart: timeutils.WeekStart(inboundSeedTestNow, time.UTC),
	}

	return refs, MailboxesFor(refs)
}

func seededTriage(
	t *testing.T,
) (*Refs, *Mailboxes, []*triage.Rule, []Message) {
	t.Helper()

	refs, boxes := inboundRefs(t)
	rules := Rules(refs, boxes)
	templates := ReplyTemplates(refs)
	messages := Messages(refs, boxes, NewRuleSet(rules), templates)

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
	for _, mailbox := range []*inboundmessage.Mailbox{boxes.Intake, boxes.Billing, boxes.Claims} {
		check("mailbox "+mailbox.Name, mailbox.Validate)
	}
	for _, rule := range rules {
		check("a rule", rule.Validate)
	}
	for _, template := range ReplyTemplates(refs) {
		check("template "+template.Name, template.Validate)
	}
	for _, view := range Views(refs) {
		check("view "+view.Name, view.Validate)
	}
	check("the settings", Settings(refs).Validate)

	providerIDs := make(map[string]struct{}, len(messages))
	for _, seeded := range messages {
		message := seeded.Message
		check(fmt.Sprintf("message %q", message.Subject), message.Validate)
		_, dup := providerIDs[message.ProviderMessageID]
		assert.Falsef(t, dup, "provider id %s is seeded twice", message.ProviderMessageID)
		providerIDs[message.ProviderMessageID] = struct{}{}
		for _, attachment := range seeded.Attachments {
			check("attachment "+attachment.FileName, attachment.Validate)
		}
		for _, reply := range seeded.Replies {
			check("a reply", reply.Validate)
			assert.Equal(t, message.ID, reply.MessageID)
			assert.Equal(t, message.ThreadKey, reply.ThreadKey)
		}
		for _, activity := range seeded.Activities {
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
		message := seeded.Message
		switch {
		case message.IsWaiting():
			lanes["waiting"]++
			if message.OwnerID == refs.Admin.ID {
				lanes["mine"]++
			}
			if message.Status == inboundmessage.StatusQuarantined {
				lanes["held"]++
			}
			if message.DueAt < refs.Now {
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
		message := seeded.Message
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
		message := seeded.Message
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
		for _, attachment := range seeded.Attachments {
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
		message := seeded.Message
		if message.HandledByRuleID.IsNil() {
			continue
		}
		rule := byID[message.HandledByRuleID]
		require.NotNil(t, rule)
		assert.Truef(t, rule.Matches(message), "%q was settled by a rule that does not match it",
			message.Subject)
		assert.True(t, message.Status.Terminal())
		if message.ReceivedAt >= refs.DayStart {
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
	assert.Len(t, rules, 5)
	assert.Equal(t, 1, paused, "one rule is paused")

	scopes := map[triage.SenderScope]bool{}
	for _, rule := range rules {
		scopes[rule.SenderScope] = true
	}
	assert.True(t, scopes[triage.SenderDomain])
	assert.True(t, scopes[triage.SenderAddress])
}

func TestInboundMessageSeed_ThreadsAndRepliesHangTogether(t *testing.T) {
	t.Parallel()

	_, _, _, messages := seededTriage(t)

	threads := map[string]int{}
	statuses := map[triage.ReplyStatus]int{}
	for _, seeded := range messages {
		threads[seeded.Message.ThreadKey]++
		for _, reply := range seeded.Replies {
			statuses[reply.Status]++
			if reply.Sent() {
				assert.True(t, seeded.Message.IsAwaitingReply(),
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
	assert.Equal(t, 1, statuses[triage.ReplyFailed])
}

func TestInboundMessageSeed_ThreeCorrectionsThisWeekAndClaimsIsQuiet(t *testing.T) {
	t.Parallel()

	refs, boxes, _, messages := seededTriage(t)

	var corrections int
	for _, seeded := range messages {
		assert.LessOrEqualf(t, seeded.Message.ReceivedAt, refs.Now,
			"%q arrives in the future", seeded.Message.Subject)
		if seeded.Message.MailboxID == boxes.Claims.ID {
			assert.LessOrEqual(t, seeded.Message.ReceivedAt, boxes.Claims.LastReceivedAt)
		}
		for _, activity := range seeded.Activities {
			if activity.Kind == triage.ActivityCorrected && activity.CreatedAt >= refs.WeekStart {
				assert.Equal(t, triage.ActorPerson, activity.ActorKind)
				corrections++
			}
		}
	}

	assert.Equal(t, len(seedCorrectionsThisWeek), corrections)
	assert.Len(t, SenderHints(refs), len(seedCorrectionsThisWeek)+1)
	health := inboundmessage.HealthOf(inboundmessage.SourceHealthInput{
		Status:         boxes.Claims.Status,
		CreatedAt:      refs.Now - 30*inboundSeedDay,
		LastReceivedAt: boxes.Claims.LastReceivedAt,
		Now:            refs.Now,
	})
	assert.Equal(t, inboundmessage.SourceQuiet, health)
}

func TestInboundMessageSeed_StoresTheTokensHashed(t *testing.T) {
	t.Parallel()

	_, boxes := inboundRefs(t)

	tokens := map[*inboundmessage.Mailbox]string{
		boxes.Intake:  SeedInboundMailboxToken,
		boxes.Billing: seedBillingMailboxToken,
		boxes.Claims:  seedClaimsMailboxToken,
	}
	for mailbox, token := range tokens {
		assert.NotEqual(t, token, mailbox.TokenHash)
		assert.Equal(t, hashutils.SHA256Hex(token), mailbox.TokenHash)
	}
}

func TestInboundMessageSeed_EveryDeckGroupHasWork(t *testing.T) {
	t.Parallel()

	_, _, _, messages := seededTriage(t)

	groups := map[triage.DeckGroup]int{}
	for _, seeded := range messages {
		message := seeded.Message
		if !message.IsWaiting() {
			continue
		}
		message.Attachments = seeded.Attachments
		var reply *triage.Reply
		if len(seeded.Replies) > 0 {
			reply = triage.LatestReply(seeded.Replies)
		}
		group, _, ok := triage.GroupFor(message, triage.Actions(triage.ActionInput{
			Message: message,
			Reply:   reply,
		}))
		if ok {
			groups[group]++
		}
	}

	for _, group := range triage.AllDeckGroups() {
		assert.Positivef(t, groups[group], "nothing waiting is in the %s group", group)
	}
}

func TestInboundMessageSeed_TeachesASuggestionAndCarriesCorrections(t *testing.T) {
	t.Parallel()

	refs, _, rules, messages := seededTriage(t)

	byHand := 0
	flaggedThisWeek, flaggedLastWeek, corrections := 0, 0, 0
	for _, seeded := range messages {
		message := seeded.Message
		for _, activity := range seeded.Activities {
			if activity.Kind == triage.ActivityFiled && activity.ActorKind == triage.ActorPerson &&
				message.SenderDomain == "blueridgebrokerage.example" &&
				message.Classification == inboundmessage.ClassificationRateConfirmation {
				byHand++
			}
			if activity.Kind == triage.ActivityFactFlagged {
				if activity.CreatedAt >= refs.WeekStart {
					flaggedThisWeek++
				} else {
					flaggedLastWeek++
				}
			}
		}
		corrections += len(seeded.Corrections)
		for _, correction := range seeded.Corrections {
			assert.Equal(t, message.ID, correction.SourceID)
		}
	}

	assert.GreaterOrEqual(t, byHand, 5,
		"a sender's rate cons are filed by hand often enough to suggest a rule")
	for _, rule := range rules {
		assert.NotEqual(t, inboundmessage.ClassificationRateConfirmation, rule.Classification,
			"no rule already covers the suggestion")
	}
	assert.Positive(t, flaggedThisWeek)
	assert.Positive(t, flaggedLastWeek)
	assert.Equal(t, len(seedCorrectionsThisWeek)+1, corrections)
}

func TestInboundMessageSeed_ReadableAttachmentsHaveADocumentAndAReading(t *testing.T) {
	t.Parallel()

	_, _, _, messages := seededTriage(t)

	var extracted int
	for _, seeded := range messages {
		documents := make(map[pulid.ID]Document, len(seeded.Documents))
		for _, staged := range seeded.Documents {
			documents[staged.Document.ID] = staged
			require.NotNil(t, staged.Content)
			assert.NotNil(t, staged.Content.Findings(), "the reading has findings")
			if staged.Extraction != nil {
				extracted++
			}
		}
		for _, attachment := range seeded.Attachments {
			if attachment.DocumentID.IsNil() {
				continue
			}
			_, ok := documents[attachment.DocumentID]
			assert.Truef(t, ok, "%s points at a document the seed writes", attachment.FileName)
		}
	}
	assert.Positive(t, extracted, "a carrier invoice has an AI extraction to queue for pay")
}
