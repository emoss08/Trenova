package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAckHealth struct {
	health  *ediservice.EDIAckHealth
	request *ediservice.GetEDIAckHealthRequest
}

func (f *fakeAckHealth) GetEDIAckHealth(
	_ context.Context,
	req *ediservice.GetEDIAckHealthRequest,
) (*ediservice.EDIAckHealth, error) {
	f.request = req

	return f.health, nil
}

func overdueMessage(partner *edi.EDIPartner, generatedAt int64) *edi.EDIMessage {
	sentAt := generatedAt + 30

	return &edi.EDIMessage{
		ID:             pulid.MustNew("edimsg_"),
		EDIPartnerID:   partner.ID,
		Partner:        partner,
		TransactionSet: edi.TransactionSet214,
		Direction:      edi.DocumentDirectionOutbound,
		DeliveryStatus: edi.MessageDeliveryStatusSent,
		DeliverySentAt: &sentAt,
		AckStatus:      edi.MessageAcknowledgmentStatusPending,
		GeneratedAt:    generatedAt,
	}
}

func TestGetEDIAckHealth_GivesCountsOverdueByPartnerAndDeadLetters(t *testing.T) {
	t.Parallel()

	acme := &edi.EDIPartner{ID: pulid.MustNew("edip_"), Name: "Acme Retail", Code: "ACME"}
	bolt := &edi.EDIPartner{ID: pulid.MustNew("edip_"), Name: "Bolt", Code: "BOLT"}
	oldest := overdueMessage(acme, 1_790_000_000)
	next := overdueMessage(acme, 1_790_003_600)
	boltOld := overdueMessage(bolt, 1_790_001_000)
	dead := &edi.EDIMessage{
		ID:                pulid.MustNew("edimsg_"),
		EDIPartnerID:      bolt.ID,
		Partner:           bolt,
		TransactionSet:    edi.TransactionSet210,
		Direction:         edi.DocumentDirectionOutbound,
		DeliveryStatus:    edi.MessageDeliveryStatusDeadLettered,
		DeliveryLastError: "AS2 MDN refused",
		RawX12:            "ISA*00*SECRET~",
	}
	fake := &fakeAckHealth{health: &ediservice.EDIAckHealth{
		Since:         1_789_950_000,
		OverdueAckAge: ediservice.DefaultOverdueAckAgeSeconds,
		PendingSince:  1_790_022_000,
		AckStatusCounts: map[edi.MessageAcknowledgmentStatus]int{
			edi.MessageAcknowledgmentStatusAccepted:           12,
			edi.MessageAcknowledgmentStatusPending:            4,
			edi.MessageAcknowledgmentStatusRejected:           1,
			edi.MessageAcknowledgmentStatusAcceptedWithErrors: 2,
			edi.MessageAcknowledgmentStatusNotExpected:        30,
		},
		AckCountsByPartner: []*ediservice.EDIPartnerAckCounts{
			{
				PartnerID: acme.ID, PartnerName: acme.Name, PartnerCode: acme.Code,
				Counts: map[edi.MessageAcknowledgmentStatus]int{
					edi.MessageAcknowledgmentStatusRejected: 1,
					edi.MessageAcknowledgmentStatusPending:  3,
					edi.MessageAcknowledgmentStatusAccepted: 9,
				},
			},
			{
				PartnerID: bolt.ID, PartnerName: bolt.Name, PartnerCode: bolt.Code,
				Counts: map[edi.MessageAcknowledgmentStatus]int{
					edi.MessageAcknowledgmentStatusPending: 1,
				},
			},
		},
		OverdueAckCount: 4,
		OverdueByPartner: []*ediservice.EDIOverduePartner{
			{
				PartnerID: acme.ID, PartnerName: acme.Name, PartnerCode: acme.Code,
				OverdueCount: 3, OldestGeneratedAt: oldest.GeneratedAt,
				Messages: []*edi.EDIMessage{oldest, next},
			},
			{
				PartnerID: bolt.ID, PartnerName: bolt.Name, PartnerCode: bolt.Code,
				OverdueCount: 1, OldestGeneratedAt: boltOld.GeneratedAt,
				Messages: []*edi.EDIMessage{boltOld},
			},
		},
		OverdueOmitted:     1,
		RecentDeadLettered: []*edi.EDIMessage{dead},
	}}
	tool := newGetEDIAckHealthTool(fake)

	result, err := tool.Query(t.Context(), agentParams(nil, ""))
	require.NoError(t, err)

	assert.Zero(t, fake.request.OverdueAckAge, "the service's own default stands")
	assert.True(t, fake.request.PartnerID.IsNil())

	view := result.(*ediAckHealthView)
	assert.Equal(t, 4, view.OverdueCount)
	assert.EqualValues(t, ediservice.DefaultOverdueAckAgeSeconds, view.OverdueAfterSeconds)
	assert.Equal(t, "4h 00m", view.OverdueAfter)
	assert.Equal(t, []ackCountRow{
		{Status: "Pending", Count: 4},
		{Status: "Accepted", Count: 12},
		{Status: "AcceptedWithErrors", Count: 2},
		{Status: "Rejected", Count: 1},
		{Status: "Failed", Count: 0},
		{Status: "NotExpected", Count: 30},
	}, view.AckCounts)

	assert.Equal(t, 15, view.Acknowledged, "accepted, accepted with errors and rejected")
	assert.Equal(t, 4, view.Awaiting)

	assert.Equal(t, []partnerAckCountsRow{
		{
			PartnerID: acme.ID.String(), Partner: "Acme Retail", PartnerCode: "ACME",
			Acknowledged: 10, Awaiting: 3,
			Answer: "10 of 13 came back with a 997 or 999 (9 accepted, 1 rejected); 3 still awaiting one",
			AckCounts: []ackCountRow{
				{Status: "Pending", Count: 3},
				{Status: "Accepted", Count: 9},
				{Status: "Rejected", Count: 1},
			},
		},
		{
			PartnerID: bolt.ID.String(), Partner: "Bolt", PartnerCode: "BOLT",
			Awaiting:  1,
			Answer:    "0 of 1 came back with a 997 or 999; 1 still awaiting one",
			AckCounts: []ackCountRow{{Status: "Pending", Count: 1}},
		},
	}, view.AckCountsByPartner, "each partner's own counts, so one call answers for one partner")

	require.Len(t, view.OverdueByPartner, 2)
	group := view.OverdueByPartner[0]
	assert.Equal(t, acme.ID.String(), group.PartnerID)
	assert.Equal(t, "Acme Retail", group.Partner)
	assert.Equal(t, 3, group.OverdueCount)
	assert.Equal(t, 1, group.NotListed)
	require.Len(t, group.Messages, 2)
	assert.Equal(t, oldest.ID.String(), group.Messages[0].ID)
	assert.Equal(t, next.ID.String(), group.Messages[1].ID)
	assert.Equal(t, bolt.ID.String(), view.OverdueByPartner[1].PartnerID)

	require.Len(t, view.RecentDeadLettered, 1)
	assert.Equal(t, "AS2 MDN refused", view.RecentDeadLettered[0].DeliveryError)
	assert.Contains(t, view.FollowUp, "edi_get_message")
	assert.Contains(t, view.FollowUp, "edi_retry_message")

	assert.ElementsMatch(t, []agent.RecordRef{
		{EntityType: ediMessageEntity, ID: oldest.ID.String()},
		{EntityType: ediMessageEntity, ID: next.ID.String()},
		{EntityType: ediMessageEntity, ID: boltOld.ID.String()},
		{EntityType: ediMessageEntity, ID: dead.ID.String()},
	}, view.TaintedRecords())

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceEDI, policy.Resource)
	assert.Equal(t, permission.OpRead, policy.Operation)
	assert.Equal(t, agent.ExternalReadAlways, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceEDI, policy.Source)
}

func TestGetEDIAckHealth_PassesThePartnerAgeAndWindowAsked(t *testing.T) {
	t.Parallel()

	partnerID := pulid.MustNew("edip_")
	fake := &fakeAckHealth{health: &ediservice.EDIAckHealth{
		OverdueAckAge:    2 * 60 * 60,
		AckStatusCounts:  map[edi.MessageAcknowledgmentStatus]int{},
		OverdueByPartner: []*ediservice.EDIOverduePartner{},
	}}
	params := agentParams(map[string]any{
		paramPartnerID:           partnerID.String(),
		paramTransactionSet:      "214",
		paramOverdueAfterSeconds: float64(7200),
		paramWindowHours:         float64(168),
	}, "")

	before := timeutils.NowUnix()
	result, err := newGetEDIAckHealthTool(fake).Query(t.Context(), params)
	require.NoError(t, err)
	after := timeutils.NowUnix()

	assert.Equal(t, partnerID, fake.request.PartnerID)
	assert.Equal(t, edi.TransactionSet214, fake.request.TransactionSet)
	assert.EqualValues(t, 7200, fake.request.OverdueAckAge)
	assert.GreaterOrEqual(t, fake.request.Since, before-168*60*60)
	assert.LessOrEqual(t, fake.request.Since, after-168*60*60)
	view := result.(*ediAckHealthView)
	assert.Equal(t, "2h 00m", view.OverdueAfter)
	assert.Empty(t, view.OverdueByPartner)
	assert.Contains(t, view.Note, "Nothing is overdue")
}

func TestGetEDIAckHealth_RefusesAnAgeOrWindowOutOfBounds(t *testing.T) {
	t.Parallel()

	for name, params := range map[string]map[string]any{
		"age under a minute": {paramOverdueAfterSeconds: float64(30)},
		"age over a month":   {paramOverdueAfterSeconds: float64(40 * 24 * 60 * 60)},
		"no window":          {paramWindowHours: float64(0)},
		"window too wide":    {paramWindowHours: float64(721)},
		"age not a number":   {paramOverdueAfterSeconds: "a while"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeAckHealth{}
			_, err := newGetEDIAckHealthTool(fake).Query(t.Context(), agentParams(params, ""))
			require.Error(t, err)
			assert.Nil(t, fake.request)
		})
	}
}

func TestAckAnswer_SaysWhatCameBackOutOfWhatWasSent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "1 of 4 came back with a 997 or 999 (1 rejected); 3 still awaiting one", ackAnswer(
		map[edi.MessageAcknowledgmentStatus]int{
			edi.MessageAcknowledgmentStatusRejected:    1,
			edi.MessageAcknowledgmentStatusPending:     3,
			edi.MessageAcknowledgmentStatusNotExpected: 1,
		}))
	assert.Equal(t, "2 of 2 came back with a 997 or 999 (1 accepted, 1 accepted with errors)", ackAnswer(
		map[edi.MessageAcknowledgmentStatus]int{
			edi.MessageAcknowledgmentStatusAccepted:           1,
			edi.MessageAcknowledgmentStatusAcceptedWithErrors: 1,
		}))
	assert.Equal(t, "nothing sent in this window expects an acknowledgment", ackAnswer(
		map[edi.MessageAcknowledgmentStatus]int{edi.MessageAcknowledgmentStatusNotExpected: 2}))
}

func TestGetEDIAckHealth_ReadsA997AsEverySetItAnswers(t *testing.T) {
	t.Parallel()

	fake := &fakeAckHealth{health: &ediservice.EDIAckHealth{
		AckStatusCounts:  map[edi.MessageAcknowledgmentStatus]int{},
		OverdueByPartner: []*ediservice.EDIOverduePartner{},
	}}

	result, err := newGetEDIAckHealthTool(fake).Query(t.Context(),
		agentParams(map[string]any{paramTransactionSet: "997"}, ""))
	require.NoError(t, err)

	assert.Empty(t, fake.request.TransactionSet, "who is behind on 997s asks about every set")
	assert.Contains(t, result.(*ediAckHealthView).Note,
		"997s and 999s are the acknowledgments themselves")
}
