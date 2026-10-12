package triageworkservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogExceptionOnMatchedShipment(t *testing.T) {
	t.Parallel()

	f := newFixture()

	updated, err := f.service.LogException(t.Context(), &LogExceptionRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Kind:       inboundmessage.ExceptionShortage,
		Quantity:   3,
		Unit:       " pallets ",
		Note:       "Driver noted on BOL",
	})
	require.NoError(t, err)

	require.Len(t, f.exceptions.requests, 1)
	req := f.exceptions.requests[0]
	assert.Equal(t, f.shipmentID, req.ShipmentID)
	assert.Equal(t, "pallets", req.Unit)
	assert.Equal(t, testUserID, req.Actor.UserID)
	assert.Equal(t, inboundmessage.ExceptionShortage, updated.ActionState.ExceptionKind)
	assert.Equal(t, int64(1_791_000_000), updated.ActionState.ExceptionLoggedAt)
	assert.Equal(t, "3 pallets short", f.recorder.find(triage.ActivityExceptionLogged).Subject)
}

func TestLogExceptionValidation(t *testing.T) {
	t.Parallel()

	f := newFixture()

	_, err := f.service.LogException(t.Context(), &LogExceptionRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
		Kind:       inboundmessage.ExceptionKind("Lost"),
		Quantity:   0,
	})
	require.Error(t, err)
	for _, field := range []string{"kind", "quantity", "unit"} {
		assert.Contains(t, err.Error(), field)
	}
	assert.Empty(t, f.exceptions.requests)
}

func TestLogExceptionNeedsShipment(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.messages[f.message.ID].MatchedShipmentID = pulid.Nil

	_, err := f.service.LogException(t.Context(), &LogExceptionRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
		Kind:       inboundmessage.ExceptionDamage,
		Quantity:   1,
		Unit:       "case",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestBlockAndUnblockSender(t *testing.T) {
	t.Parallel()

	f := newFixture()

	updated, err := f.service.BlockSender(t.Context(), &BlockSenderRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
	})
	require.NoError(t, err)

	mailbox := f.mailboxes.mailboxes[f.message.MailboxID]
	assert.Equal(t, []string{"billing@carrier.example"}, mailbox.BlockedSenders)
	assert.Equal(t, inboundmessage.StatusIgnored, updated.Status)
	assert.Equal(t, testUserID, updated.ReviewedBy)
	assert.NotZero(t, updated.ActionState.SenderBlockedAt)
	assert.NotNil(t, f.recorder.find(triage.ActivitySenderBlocked))
	assert.NotNil(t, f.recorder.find(triage.ActivityIgnored))

	_, err = f.service.BlockSender(t.Context(), &BlockSenderRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
	})
	require.NoError(t, err)
	assert.Len(t, f.mailboxes.mailboxes[f.message.MailboxID].BlockedSenders, 1)

	unblocked, err := f.service.UnblockSender(t.Context(), &UnblockSenderRequest{
		TenantInfo: testTenant,
		MailboxID:  f.message.MailboxID,
		Sender:     " Billing@Carrier.example ",
	})
	require.NoError(t, err)
	assert.Empty(t, unblocked.BlockedSenders)

	_, err = f.service.UnblockSender(t.Context(), &UnblockSenderRequest{
		TenantInfo: testTenant,
		MailboxID:  f.message.MailboxID,
	})
	require.Error(t, err)
}

func TestDecideStopsSettlesAndDrafts(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.messages[f.message.ID].Classification = inboundmessage.ClassificationDetentionDispute

	updated, err := f.service.DecideStops(t.Context(), &DecideStopsRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Decision:   inboundmessage.StopsDecisionWaive,
	})
	require.NoError(t, err)

	require.Len(t, f.detention.requests, 1)
	assert.Equal(t, inboundmessage.StopsDecisionWaive, f.detention.requests[0].Decision)
	assert.Equal(t, inboundmessage.StopsDecisionWaive, updated.ActionState.StopsDecision)
	assert.Equal(t, "150.00 USD", f.recorder.find(triage.ActivityStopsDecided).Detail)
	assert.Equal(t, []pulid.ID{f.message.ID}, f.drafter.drafted)
}

func TestDecideStopsOnlyForDisputes(t *testing.T) {
	t.Parallel()

	f := newFixture()

	_, err := f.service.DecideStops(t.Context(), &DecideStopsRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
		Decision:   inboundmessage.StopsDecisionStandBy,
	})
	require.Error(t, err)
	assert.Empty(t, f.detention.requests)

	_, err = f.service.DecideStops(t.Context(), &DecideStopsRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
		Decision:   inboundmessage.StopsDecision("Maybe"),
	})
	require.Error(t, err)
	assert.Equal(t, []string{"decision"}, errorFields(err))
}
