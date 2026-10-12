package triagebriefingservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chicago(t *testing.T) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	return loc
}

func baseInputs(t *testing.T, hour int) *Inputs {
	t.Helper()

	loc := chicago(t)
	dayStart := time.Date(2026, time.October, 12, 0, 0, 0, 0, loc).Unix()
	now := time.Date(2026, time.October, 12, hour, 30, 0, 0, loc).Unix()

	return &Inputs{
		Now:      now,
		DayStart: dayStart,
		Location: loc,
		Settings: triage.DefaultSettings("org_test", "bu_test"),
		Counts:   &triageports.BriefingCounts{AnyMessage: true, Waiting: 3, Overdue: 1},
		Mailboxes: []*inboundmessage.Mailbox{
			{ID: "imbx_intake", Status: inboundmessage.MailboxActive, CreatedAt: now - 86_400, LastReceivedAt: now - 60},
			{ID: "imbx_claims", Status: inboundmessage.MailboxActive, CreatedAt: now - 30*86_400, LastReceivedAt: now - 7*86_400},
			{ID: "imbx_old", Status: inboundmessage.MailboxInactive, CreatedAt: now - 90*86_400},
		},
		Corrections: &triageports.CorrectionTotals{},
		Replies:     map[pulid.ID]*triage.Reply{},
	}
}

func TestAssemble_DayAndEndOfDay(t *testing.T) {
	t.Parallel()

	day := Assemble(baseInputs(t, 14))
	assert.Equal(t, inboundmessageports.BriefingDay, day.Phase)
	assert.Empty(t, day.CarryOver)

	evening := baseInputs(t, 17)
	evening.Settings.OnCallUserID = "usr_marcus"
	evening.Waiting = []*inboundmessage.InboundMessage{waitingTender("imsg_1", 100)}
	eod := Assemble(evening)
	assert.Equal(t, inboundmessageports.BriefingEndOfDay, eod.Phase)
	require.Len(t, eod.CarryOver, 1)
	assert.Equal(t, pulid.ID("usr_marcus"), eod.OnCallUserID)
}

func TestAssemble_SourcesAndFirstDay(t *testing.T) {
	t.Parallel()

	in := baseInputs(t, 10)
	briefing := Assemble(in)
	assert.Equal(t, 2, briefing.ActiveSources)
	assert.Equal(t, []pulid.ID{"imbx_claims"}, briefing.UnhealthySourceIDs)
	assert.False(t, briefing.FirstDay)

	in.Counts.AnyMessage = false
	assert.True(t, Assemble(in).FirstDay)

	in = baseInputs(t, 10)
	in.Mailboxes = nil
	assert.True(t, Assemble(in).FirstDay)
}

func TestArrivals_FillEveryHourSinceMidnight(t *testing.T) {
	t.Parallel()

	in := baseInputs(t, 3)
	buckets := Arrivals([]*triageports.ArrivalRow{
		{Hour: 1, HandledByNova: 2, LeftForPerson: 1},
		{Hour: 9, HandledByNova: 5},
	}, in.DayStart, in.Now)

	require.Len(t, buckets, 4)
	assert.Equal(t, in.DayStart+3600, buckets[1].HourStart)
	assert.Equal(t, 2, buckets[1].HandledByNova)
	assert.Equal(t, 1, buckets[1].LeftForPerson)
	assert.Zero(t, buckets[3].HandledByNova, "an hour past now is not drawn")
}

func TestHandledGroups_KeepCountsAndThreeSamples(t *testing.T) {
	t.Parallel()

	row := func(id string, action inboundmessage.NovaAction, count int) *triageports.HandledRow {
		return &triageports.HandledRow{
			MessageRow: triageports.MessageRow{
				ID:             pulid.ID(id),
				Classification: inboundmessage.ClassificationProofOfDelivery,
				HandledAction:  action,
				Subject:        "POD S-1041",
			},
			GroupCount: count,
		}
	}

	groups := HandledGroups([]*triageports.HandledRow{
		row("a", inboundmessage.NovaActionFileOnShipment, 12),
		row("b", inboundmessage.NovaActionFileOnShipment, 12),
		row("c", inboundmessage.NovaActionFileOnShipment, 12),
		row("d", inboundmessage.NovaActionFileOnShipment, 12),
		row("e", inboundmessage.NovaActionIgnore, 2),
	})

	require.Len(t, groups, 2)
	assert.Equal(t, 12, groups[0].Count)
	assert.Len(t, groups[0].Samples, 3)
	require.NotNil(t, groups[0].Classification)
	assert.Equal(t, inboundmessage.NovaActionIgnore, groups[1].Action)
}

func TestRiver_SplitsNovaFromPeople(t *testing.T) {
	t.Parallel()

	nova, yours := River([]*triageports.RiverRow{
		{MessageRow: triageports.MessageRow{ID: "a", Status: inboundmessage.StatusActioned}, Nova: true},
		{MessageRow: triageports.MessageRow{ID: "b", Status: inboundmessage.StatusInReview, DueAt: 10}},
		{MessageRow: triageports.MessageRow{ID: "c", Status: inboundmessage.StatusInReview, Parked: inboundmessage.ParkingSnoozed}},
	})

	require.Len(t, nova, 1)
	require.Len(t, yours, 2)
	assert.True(t, nova[0].Done)
	assert.False(t, yours[0].Done)
	require.NotNil(t, yours[0].DueAt)
	assert.True(t, yours[1].Done, "a parked message is off the person's plate for now")
}

func waitingTender(id string, due int64) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:             pulid.ID(id),
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationTender,
		NovaRead:       true,
		DueAt:          due,
		ReceivedAt:     due - 7200,
	}
}

func TestDeck_GroupsWaitingMailByItsNextStep(t *testing.T) {
	t.Parallel()

	pod := func(id string, due int64) *inboundmessage.InboundMessage {
		return &inboundmessage.InboundMessage{
			ID:                pulid.ID(id),
			Status:            inboundmessage.StatusInReview,
			Classification:    inboundmessage.ClassificationProofOfDelivery,
			NovaRead:          true,
			DueAt:             due,
			MatchedShipmentID: "shp_1",
			Attachments: []*inboundmessage.InboundAttachment{
				{ID: pulid.ID("att_" + id), DocumentID: "doc_1", Kind: inboundmessage.AttachmentProofOfDelivery},
			},
		}
	}
	status := &inboundmessage.InboundMessage{
		ID:                "imsg_status",
		Status:            inboundmessage.StatusInReview,
		Classification:    inboundmessage.ClassificationStatusRequest,
		NovaRead:          true,
		DueAt:             50,
		MatchedShipmentID: "shp_2",
	}
	held := &inboundmessage.InboundMessage{
		ID:             "imsg_held",
		Status:         inboundmessage.StatusQuarantined,
		Classification: inboundmessage.ClassificationOther,
		NovaRead:       true,
		DueAt:          70,
	}
	unread := &inboundmessage.InboundMessage{ID: "imsg_unread", Status: inboundmessage.StatusInReview}

	deck := Deck([]*inboundmessage.InboundMessage{
		status, pod("imsg_pod1", 60), held, pod("imsg_pod2", 80), waitingTender("imsg_t", 90), unread,
	}, map[pulid.ID]*triage.Reply{})

	groups := make(map[triage.DeckGroup]*inboundmessageports.DeckCard, len(deck.cards))
	order := make([]triage.DeckGroup, 0, len(deck.cards))
	for _, card := range deck.cards {
		groups[card.Group] = card
		order = append(order, card.Group)
	}
	assert.Equal(t, []triage.DeckGroup{triage.DeckFile, triage.DeckDraft, triage.DeckDecide, triage.DeckHeld}, order)
	assert.Equal(t, 2, groups[triage.DeckFile].Count)
	assert.Equal(t, 2, groups[triage.DeckFile].EstimateMinutes)
	assert.Equal(t, []pulid.ID{"imsg_pod1", "imsg_pod2"}, groups[triage.DeckFile].MessageIDs)
	assert.Equal(t, 2, groups[triage.DeckDraft].Count,
		"a status request and a tender are both drafts Nova can write")
	assert.Equal(t, 1, groups[triage.DeckDecide].Count, "an unread message needs a person's call")
	assert.Equal(t, 1, groups[triage.DeckHeld].Count)
	assert.Equal(t, 2+4+4+1, deck.estimate)
	require.NotNil(t, deck.longest)
	assert.Equal(t, pulid.ID("imsg_status"), deck.longest.MessageID)
	require.NotNil(t, groups[triage.DeckFile].Items[0].Action)
	assert.Equal(t, triage.ActionFile, groups[triage.DeckFile].Items[0].Action.Kind)
}

func TestCorrections_AccuracyInPercentagePoints(t *testing.T) {
	t.Parallel()

	summary := Corrections(&triageports.CorrectionTotals{
		CorrectionsThisWeek: 4,
		ReadThisWeek:        200,
		CorrectedThisWeek:   4,
		ReadLastWeek:        100,
		CorrectedLastWeek:   5,
	})

	assert.Equal(t, 4, summary.ThisWeek)
	assert.InDelta(t, 98.0, summary.Accuracy, 0.001)
	assert.InDelta(t, 3.0, summary.AccuracyChange, 0.001)

	quiet := Corrections(&triageports.CorrectionTotals{ReadThisWeek: 10})
	assert.InDelta(t, 100.0, quiet.Accuracy, 0.001)
	assert.Zero(t, quiet.AccuracyChange, "no reading last week means no change to report")
}
