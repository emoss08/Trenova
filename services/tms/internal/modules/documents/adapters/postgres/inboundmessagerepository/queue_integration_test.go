//go:build integration

package inboundmessagerepository

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const queueNow int64 = 1_790_000_000

type queueTenant struct {
	ctx     context.Context
	info    pagination.TenantInfo
	mailbox *inboundmessage.Mailbox
}

type queueFixture struct {
	admin     *keel.DB
	queue     inboundmessageports.TriageQueueRepository
	messages  inboundmessageports.InboundMessageRepository
	mailboxes inboundmessageports.TriageMailboxStateRepository
	a         queueTenant
	b         queueTenant
}

func newQueueFixture(t *testing.T) *queueFixture {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	a := seedtest.SeedFullTestData(t, ctx, admin)
	b := seedtest.SeedAdditionalTenant(t, ctx, admin, "IMB")
	conn := dbtest.NewRLSConnection(t, admin, dsn)
	params := Params{DB: conn, Logger: zap.NewNop()}

	f := &queueFixture{
		admin:     admin,
		queue:     NewQueueRepository(params),
		messages:  New(params),
		mailboxes: NewMailboxStateRepository(MailboxParams{DB: conn, Logger: zap.NewNop()}),
		a:         queueTenant{ctx: a.Ctx(t.Context()), info: a.Info()},
		b:         queueTenant{ctx: b.Ctx(t.Context()), info: b.Info()},
	}
	f.a.mailbox = f.newMailbox(t, f.a, "a")
	f.b.mailbox = f.newMailbox(t, f.b, "b")

	return f
}

func (f *queueFixture) newMailbox(
	t *testing.T,
	tenant queueTenant,
	label string,
) *inboundmessage.Mailbox {
	t.Helper()

	mailbox := &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		OrganizationID: tenant.info.OrgID,
		BusinessUnitID: tenant.info.BuID,
		Name:           "Dispatch " + label,
		Address:        "dispatch-" + label + "@inbox.test",
		Provider:       inboundmessage.ProviderPostmark,
		TokenHash:      pulid.MustNew("tok_").String(),
		ReviewPolicy:   inboundmessage.ReviewAlways,
		Status:         inboundmessage.MailboxActive,
	}
	require.NoError(t, inboundmessagedb.Mailboxes.Create(tenant.ctx, f.admin, mailbox))

	return mailbox
}

func (f *queueFixture) newMessage(
	t *testing.T,
	tenant queueTenant,
	edit func(m *inboundmessage.InboundMessage),
) *inboundmessage.InboundMessage {
	t.Helper()

	id := pulid.MustNew("imsg_")
	message := &inboundmessage.InboundMessage{
		ID:                id,
		OrganizationID:    tenant.info.OrgID,
		BusinessUnitID:    tenant.info.BuID,
		MailboxID:         tenant.mailbox.ID,
		ProviderMessageID: "pm-" + id.String(),
		FromAddress:       "broker@example.com",
		FromName:          "Broker",
		ToAddresses:       []string{tenant.mailbox.Address},
		Subject:           "Load " + id.String(),
		TextBody:          "Please confirm the load.",
		ReceivedAt:        queueNow - 3600,
		Status:            inboundmessage.StatusInReview,
		ThreadKey:         "thread-" + strings.ToLower(id.String()),
		SenderDomain:      "example.com",
	}
	if edit != nil {
		edit(message)
	}
	require.NoError(t, inboundmessagedb.InboundMessages.Create(tenant.ctx, f.admin, message))

	return message
}

func (f *queueFixture) reload(
	t *testing.T,
	tenant queueTenant,
	id pulid.ID,
) *inboundmessage.InboundMessage {
	t.Helper()

	message, err := f.messages.GetByID(tenant.ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         id,
		TenantInfo: tenant.info,
	})
	require.NoError(t, err)

	return message
}

func TestEveryQueuePatchIsOneValidStatement(t *testing.T) {
	f := newQueueFixture(t)

	owner := f.a.info.UserID
	cases := map[string]inboundmessageports.QueuePatch{
		"snooze until a time": {Park: &inboundmessageports.ParkPatch{
			Parking:      inboundmessage.ParkingSnoozed,
			SnoozedUntil: 1_800_000_000,
			By:           owner,
		}},
		"snooze until an event": {Park: &inboundmessageports.ParkPatch{
			Parking:       inboundmessage.ParkingSnoozed,
			SnoozeEvent:   inboundmessage.SnoozeEventShipmentDelivered,
			WatchShipment: true,
			By:            owner,
		}},
		"wait for a reply": {Park: &inboundmessageports.ParkPatch{
			Parking:       inboundmessage.ParkingAwaitingReply,
			AwaitingUntil: 1_800_000_000,
		}},
		"park while also asked to unpark": {
			Unpark: true,
			Park: &inboundmessageports.ParkPatch{
				Parking:      inboundmessage.ParkingSnoozed,
				SnoozedUntil: 1_800_000_000,
			},
		},
		"unpark and settle": {
			Unpark: true,
			Status: inboundmessage.StatusActioned,
			Review: &inboundmessageports.ReviewPatch{ReviewerID: owner},
			Owner:  &owner,
			Due:    inboundmessageports.DueClear,
		},
		"reopen against the targets": {
			Due:     inboundmessageports.DueFromTargets,
			Targets: triage.ResolveTargets(nil),
		},
		"hand off": {
			HandOff: &inboundmessageports.HandOffPatch{To: owner},
		},
	}

	for name, patch := range cases {
		message := f.newMessage(t, f.a, nil)
		states, err := f.queue.UpdateQueueState(f.a.ctx, inboundmessageports.UpdateQueueStateRequest{
			TenantInfo: f.a.info,
			IDs:        []pulid.ID{message.ID},
			Patch:      patch,
			Now:        queueNow,
		})
		require.NoError(t, err, name)
		require.Len(t, states, 1, name)
		assert.Equal(t, message.ID, states[0].ID, name)
	}
}

func TestAQueueChangeStaysInsideOneTenant(t *testing.T) {
	f := newQueueFixture(t)

	ours := f.newMessage(t, f.a, nil)
	theirs := f.newMessage(t, f.b, nil)
	reviewer := f.a.info.UserID

	states, err := f.queue.UpdateQueueState(f.a.ctx, inboundmessageports.UpdateQueueStateRequest{
		TenantInfo: f.a.info,
		IDs:        []pulid.ID{ours.ID, theirs.ID},
		Patch: inboundmessageports.QueuePatch{
			Status: inboundmessage.StatusActioned,
			Review: &inboundmessageports.ReviewPatch{ReviewerID: reviewer, Note: "done"},
			Unpark: true,
			Due:    inboundmessageports.DueClear,
		},
		Now: queueNow,
	})
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, ours.ID, states[0].ID)
	assert.Equal(t, inboundmessage.StatusActioned, states[0].Status)
	assert.Equal(t, "done", states[0].ReviewNote)

	settled := f.reload(t, f.a, ours.ID)
	assert.Equal(t, reviewer, settled.ReviewedBy)
	assert.Equal(t, queueNow, settled.ReviewedAt)
	assert.Equal(t, ours.Version+1, settled.Version)

	untouched := f.reload(t, f.b, theirs.ID)
	assert.Equal(t, inboundmessage.StatusInReview, untouched.Status)
	assert.Equal(t, theirs.Version, untouched.Version)
}

func TestAnEventWakesOnlyTheMessagesWatchingIt(t *testing.T) {
	f := newQueueFixture(t)

	shipmentID := pulid.MustNew("shp_")
	snooze := func(m *inboundmessage.InboundMessage) {
		m.Parked = inboundmessage.ParkingSnoozed
		m.SnoozeEvent = inboundmessage.SnoozeEventShipmentDelivered
		m.SnoozeWatchID = shipmentID
	}
	watching := f.newMessage(t, f.a, snooze)
	other := f.newMessage(t, f.a, func(m *inboundmessage.InboundMessage) {
		snooze(m)
		m.SnoozeWatchID = pulid.MustNew("shp_")
	})
	elsewhere := f.newMessage(t, f.b, snooze)

	woken, err := f.queue.WakeByEvent(f.a.ctx, inboundmessageports.WakeByEventRequest{
		TenantInfo: f.a.info,
		Event:      inboundmessage.SnoozeEventShipmentDelivered,
		WatchID:    shipmentID,
		Now:        queueNow,
	})
	require.NoError(t, err)
	require.Len(t, woken, 1)
	assert.Equal(t, watching.ID, woken[0].ID)
	assert.Empty(t, woken[0].Parked)
	assert.Empty(t, woken[0].SnoozeWatchID)

	assert.Equal(t, inboundmessage.ParkingSnoozed, f.reload(t, f.a, other.ID).Parked)
	assert.Equal(t, inboundmessage.ParkingSnoozed, f.reload(t, f.b, elsewhere.ID).Parked)
}

func TestTheReconcileReadsBothTimersAcrossTenants(t *testing.T) {
	f := newQueueFixture(t)

	snoozed := f.newMessage(t, f.a, func(m *inboundmessage.InboundMessage) {
		m.Parked = inboundmessage.ParkingSnoozed
		m.SnoozedUntil = queueNow - 60
	})
	awaiting := f.newMessage(t, f.b, func(m *inboundmessage.InboundMessage) {
		m.Parked = inboundmessage.ParkingAwaitingReply
		m.AwaitingUntil = queueNow - 60
	})
	f.newMessage(t, f.a, func(m *inboundmessage.InboundMessage) {
		m.Parked = inboundmessage.ParkingSnoozed
		m.SnoozedUntil = queueNow + 3600
	})
	f.newMessage(t, f.a, func(m *inboundmessage.InboundMessage) {
		m.Parked = inboundmessage.ParkingSnoozed
		m.SnoozeEvent = inboundmessage.SnoozeEventInvoicePaid
	})

	refs, err := f.queue.ListOverdueParked(t.Context(), inboundmessageports.ListOverdueParkedRequest{
		Now:   queueNow,
		Limit: 500,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []inboundmessageports.ParkedRef{
		{ID: snoozed.ID, OrganizationID: f.a.info.OrgID, BusinessUnitID: f.a.info.BuID},
		{ID: awaiting.ID, OrganizationID: f.b.info.OrgID, BusinessUnitID: f.b.info.BuID},
	}, refs)

	capped, err := f.queue.ListOverdueParked(t.Context(), inboundmessageports.ListOverdueParkedRequest{
		Now:   queueNow,
		Limit: 1,
	})
	require.NoError(t, err)
	assert.Len(t, capped, 1)
}

func TestATargetChangeRecomputesOneKindInOneStatement(t *testing.T) {
	f := newQueueFixture(t)

	unread := f.newMessage(t, f.a, nil)
	other := f.newMessage(t, f.a, func(m *inboundmessage.InboundMessage) {
		m.Classification = inboundmessage.ClassificationOther
	})
	tender := f.newMessage(t, f.a, func(m *inboundmessage.InboundMessage) {
		m.Classification = inboundmessage.ClassificationTender
		m.DueAt = queueNow
	})
	settled := f.newMessage(t, f.a, func(m *inboundmessage.InboundMessage) {
		m.Classification = inboundmessage.ClassificationOther
		m.Status = inboundmessage.StatusActioned
	})
	elsewhere := f.newMessage(t, f.b, func(m *inboundmessage.InboundMessage) {
		m.Classification = inboundmessage.ClassificationOther
	})

	n, err := f.queue.RecomputeDue(f.a.ctx, inboundmessageports.RecomputeDueRequest{
		TenantInfo:     f.a.info,
		Classification: inboundmessage.ClassificationOther,
		Minutes:        90,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), n, "an unread message waits on the Other target")

	assert.Equal(t, unread.ReceivedAt+5400, f.reload(t, f.a, unread.ID).DueAt)
	assert.Equal(t, other.ReceivedAt+5400, f.reload(t, f.a, other.ID).DueAt)
	assert.Equal(t, queueNow, f.reload(t, f.a, tender.ID).DueAt)
	assert.Zero(t, f.reload(t, f.a, settled.ID).DueAt)
	assert.Zero(t, f.reload(t, f.b, elsewhere.ID).DueAt)
}

func TestMarkingAMailboxReceivedNeverMovesItBackwards(t *testing.T) {
	f := newQueueFixture(t)

	mark := func(at int64) {
		t.Helper()
		require.NoError(t, f.mailboxes.MarkReceived(f.a.ctx, inboundmessageports.MailboxReceivedRequest{
			TenantInfo: f.a.info,
			MailboxID:  f.a.mailbox.ID,
			ReceivedAt: at,
		}))
	}
	lastReceived := func() int64 {
		t.Helper()
		mb := inboundmessagedb.Mailboxes
		mailbox, err := mb.Get(f.a.ctx, f.admin, f.a.mailbox.ID)
		require.NoError(t, err)
		return mailbox.LastReceivedAt
	}

	mark(queueNow)
	assert.Equal(t, queueNow, lastReceived())

	mark(queueNow - 600)
	assert.Equal(t, queueNow, lastReceived())

	require.NoError(t, f.mailboxes.MarkReceived(f.a.ctx, inboundmessageports.MailboxReceivedRequest{
		TenantInfo: f.a.info,
		MailboxID:  f.b.mailbox.ID,
		ReceivedAt: queueNow,
	}))
	mb := inboundmessagedb.Mailboxes
	theirs, err := mb.Get(f.b.ctx, f.admin, f.b.mailbox.ID)
	require.NoError(t, err)
	assert.Zero(t, theirs.LastReceivedAt)
}

func TestRetentionDeletesOnlyInsideOneTenant(t *testing.T) {
	f := newQueueFixture(t)

	ours := f.newMessage(t, f.a, nil)
	theirs := f.newMessage(t, f.b, nil)

	deleted, err := f.messages.DeleteByIDs(f.a.ctx, inboundmessageports.DeleteInboundMessagesRequest{
		TenantInfo: f.a.info,
		IDs:        []pulid.ID{ours.ID, theirs.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	m := inboundmessagedb.InboundMessages
	gone, err := m.Where(m.ID.Eq(ours.ID)).Exists(f.a.ctx, f.admin)
	require.NoError(t, err)
	assert.False(t, gone)

	kept, err := m.Where(m.ID.Eq(theirs.ID)).Exists(f.b.ctx, f.admin)
	require.NoError(t, err)
	assert.True(t, kept)
}
