package ediboardreader

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/ediboard"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	morning = int64(1_795_960_000)
	partner = "Acme Manufacturing EDI"
)

func answered(bol, pro, delivery, ack string, at int64) tenderRow {
	return tenderRow{
		FileID:         pulid.ID("ediinf_" + bol),
		FileStatus:     "Processed",
		ReceivedAt:     at,
		TenderID:       pulid.ID("edimsg_t" + bol),
		TenderStatus:   "Generated",
		TransferID:     pulid.ID("edilt_" + bol),
		TransferStatus: "Approved",
		BOL:            bol,
		MadeShipment:   pulid.ID("shp_" + bol),
		ShipmentID:     pulid.ID("shp_" + bol),
		ProNumber:      pro,
		AnswerID:       pulid.ID("edimsg_a" + bol),
		AnswerStatus:   "Generated",
		Delivery:       delivery,
		Ack:            ack,
	}
}

func quarantined(bol string, at int64) tenderRow {
	return tenderRow{
		FileID:       pulid.ID("ediinf_" + bol),
		FileStatus:   "Quarantined",
		FileReason:   "ship-to 0412 is not a known location\nsecond line",
		ReceivedAt:   at,
		TenderID:     pulid.ID("edimsg_t" + bol),
		TenderStatus: "Failed",
	}
}

func TestFileAndMessageStatusesAreTheEDICardsWordsAndTones(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	files := map[string]struct {
		word string
		tone boardcatalog.Tone
	}{
		"Received":           {"Received", boardcatalog.ToneInfo},
		"Parsed":             {"Handling its sets", boardcatalog.ToneInfo},
		"Processed":          {"Processed", boardcatalog.ToneOK},
		"PartiallyProcessed": {"Partly processed", boardcatalog.ToneWarn},
		"Quarantined":        {"Quarantined", boardcatalog.ToneWarn},
		"Duplicate":          {"Duplicate interchange", ""},
	}
	for status, want := range files {
		word, tone := fileStatus(ctx, status)
		assert.Equal(t, want.word, word, status)
		assert.Equal(t, want.tone, tone, status)
	}

	messages := []struct {
		state messageState
		word  string
		tone  boardcatalog.Tone
	}{
		{messageState{"Outbound", "Generated", "Failed", "Pending"}, "Delivery failed",
			boardcatalog.ToneWarn},
		{messageState{"Outbound", "Generated", "DeadLettered", ""}, "Dead-lettered",
			boardcatalog.ToneWarn},
		{messageState{"Outbound", "Generated", "Sent", "Pending"}, "Awaiting acknowledgment",
			boardcatalog.ToneInfo},
		{messageState{"Outbound", "Generated", "Sent", "Accepted"}, "Accepted",
			boardcatalog.ToneOK},
		{messageState{"Outbound", "Generated", "Sent", "Rejected"}, "Rejected by partner",
			boardcatalog.ToneWarn},
		{messageState{"Outbound", "Generated", "Sent", "NotExpected"}, "Delivered",
			boardcatalog.ToneOK},
		{messageState{"Outbound", "Generated", "Queued", ""}, "Queued", ""},
		{messageState{"Outbound", "Failed", "", ""}, "Not generated", boardcatalog.ToneWarn},
		{messageState{"Inbound", "Failed", "", "NotExpected"}, "Could not be read",
			boardcatalog.ToneWarn},
		{messageState{"Inbound", "Generated", "", "NotExpected"}, "Received",
			boardcatalog.ToneOK},
	}
	for _, tc := range messages {
		word, tone := messageStatus(ctx, &tc.state)
		assert.Equal(t, tc.word, word, tc.state)
		assert.Equal(t, tc.tone, tone, tc.state)
	}
}

func TestAQuarantinedFileWaitsWithItsReasonAndAProcessedOneDoesNot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	waiting := fileWaiting(ctx, &fileRow{
		Status:        "Quarantined",
		FailureReason: "ship-to 0412 is not a known location\nsegment N1",
		ReceivedAt:    morning,
	}, morning+2*3600+60)
	assert.Equal(t, boardcatalog.Duration(2*3600+60), waiting.Value)
	assert.Equal(t, "{0} waiting", waiting.Standing.Text)
	assert.Equal(t, boardcatalog.ToneWarn, waiting.Tone)
	require.NotNil(t, waiting.Fact)
	assert.Equal(t, "ship-to 0412 is not a known location", waiting.Fact.Value.Text,
		"the second fact is the reason's first line, carried as a value, never as the template")

	processed := morning + 20
	done := fileWaiting(ctx, &fileRow{
		Status: "Processed", ReceivedAt: morning, ProcessedAt: &processed,
	}, morning+3600)
	assert.True(t, done.Value.Number.IsZero())
	assert.Equal(t, "Processed {0}", done.Standing.Text)
	assert.Equal(t, processed, done.Standing.At)
	assert.Empty(t, done.Tone)
	assert.Nil(t, done.Fact)
}

func TestAnUndeliveredAnswerWaitsSinceItWasWrittenWithItsError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	failed := messageWaiting(ctx, &messageRow{
		Direction: "Outbound", Status: "Generated", Delivery: "Failed", Ack: "Pending",
		DeliveryError: "SFTP login refused", GeneratedAt: morning,
	}, morning+36*60)
	assert.Equal(t, boardcatalog.Duration(36*60), failed.Value)
	assert.Equal(t, boardcatalog.ToneWarn, failed.Tone)
	require.NotNil(t, failed.Fact)
	assert.Equal(t, "SFTP login refused", failed.Fact.Value.Text)

	acked := morning + 600
	sent := morning + 180
	done := messageWaiting(ctx, &messageRow{
		Direction: "Outbound", Status: "Generated", Delivery: "Sent", Ack: "Accepted",
		GeneratedAt: morning, SentAt: &sent, AckReceivedAt: &acked,
	}, morning+3600)
	assert.True(t, done.Value.Number.IsZero())
	assert.Equal(t, "Acknowledged {0}", done.Standing.Text)
	assert.Equal(t, acked, done.Standing.At)
}

func TestAMessageIsNamedByTheTenderItAnswersNeverByAnID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	assert.Equal(t, "990 for tender 88244", messageLabel(ctx, &messageRow{
		TransactionSet: "990", TenderBOL: "88244", ProNumber: "SEED-SHP-005", Partner: partner,
	}))
	assert.Equal(t, "214 for SEED-SHP-005", messageLabel(ctx, &messageRow{
		TransactionSet: "214", ProNumber: "SEED-SHP-005", Partner: partner,
	}))
	assert.Equal(t, "997 from "+partner, messageLabel(ctx, &messageRow{
		Direction: "Inbound", TransactionSet: "997", Partner: partner,
	}))
}

func TestPartnerDayLogsEachTenderInOrderWithWhatBecameOfIt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	rows := []tenderRow{
		answered("88244", "SEED-SHP-006", "Failed", "Pending", morning+500),
		answered("88240", "SEED-SHP-004", "Sent", "Pending", morning+300),
		quarantined("88231", morning+200),
		answered("88214", "SEED-SHP-001", "Sent", "Accepted", morning),
		answered("88214", "SEED-SHP-001", "Failed", "Pending", morning),
	}
	rows[0].DeliveryError = "SFTP login refused"
	median := 180.0
	read := partnerDay(ctx, &partnerDayInput{
		partner: partner,
		period:  boardcatalog.PeriodToday,
		rows:    rows,
		figures: &dayFigures{
			Lines: 4, Tenders: 4, Shipments: 3, Visible: 3, Answers: 3, Delivered: 2, NeedHand: 2,
			Median: &median,
		},
	})

	assert.Equal(t, "Today's tenders from "+partner, read.Title)
	assert.Equal(t, "4 inbound tenders and 3 answers", read.Source)
	require.Len(t, read.Log, 4, "a second row of the same tender is the same line")

	first := read.Log[0]
	assert.Equal(t, morning, first.At)
	assert.Equal(t, "Tender 88214 became SEED-SHP-001; our answer was sent and acknowledged",
		first.Text, "the first row of a tender, its latest answer, is the one kept")
	assert.Equal(t, "204 in, 990 out, 997 accepted", first.Detail)
	assert.Empty(t, first.Tone)
	assert.Equal(t, []boardcatalog.Ref{{Kind: shipmentKind, ID: "shp_88214"}}, first.Refs)

	held := read.Log[1]
	assert.Equal(t, "A tender was quarantined: ship-to 0412 is not a known location", held.Text)
	assert.Equal(t, boardcatalog.ToneWarn, held.Tone)
	assert.Equal(t, []boardcatalog.Ref{{Kind: recordkind.EDIInboundFile, ID: "ediinf_88231"}},
		held.Refs)

	assert.Equal(t, "Tender 88240 became SEED-SHP-004; our answer was sent", read.Log[2].Text)
	assert.Equal(t, "204 in, 990 out, acknowledgment pending", read.Log[2].Detail)

	lost := read.Log[3]
	assert.Equal(t, "Tender 88244 became SEED-SHP-006, but our answer did not reach "+partner,
		lost.Text)
	assert.Equal(t, "990 failed: SFTP login refused", lost.Detail)
	assert.Equal(t, boardcatalog.ToneWarn, lost.Tone)
	assert.Equal(t, []boardcatalog.Ref{
		{Kind: shipmentKind, ID: "shp_88244"},
		{Kind: recordkind.EDIMessage, ID: "edimsg_a88244"},
	}, lost.Refs)

	for _, entry := range read.Log {
		assert.NotContains(t, entry.Text, "_", "no id is written into a log line")
		assert.NotContains(t, entry.Detail, "_")
	}
	assert.Equal(t, boardcatalog.Count(4), *read.Figures[figureTendersIn])
	assert.Equal(t, boardcatalog.Count(3), *read.Figures[figureShipmentsMade])
	assert.Equal(t, boardcatalog.Count(2), *read.Figures[figureAnswersDelivered])
	assert.Equal(t, boardcatalog.Count(2), *read.Figures[figureNeedHand])
	assert.Equal(t, boardcatalog.Duration(180), *read.Figures[figureMedianAnswer])
}

func TestPartnerDayKeepsTheLatestLinesAndSaysSo(t *testing.T) {
	t.Parallel()

	rows := make([]tenderRow, 0, ediboard.PartnerDayRows+5)
	for idx := range ediboard.PartnerDayRows + 5 {
		bol := fmt.Sprintf("9%02d", idx)
		rows = append(rows, answered(bol, "PRO-"+bol, "Sent", "Accepted",
			morning+int64(1000-idx)))
	}
	read := partnerDay(context.Background(), &partnerDayInput{
		partner: partner,
		period:  boardcatalog.PeriodYesterday,
		rows:    rows,
		figures: &dayFigures{Lines: 40, Tenders: 40, Answers: 40},
	})

	require.Len(t, read.Log, ediboard.PartnerDayRows)
	assert.Equal(t, morning+int64(1000-ediboard.PartnerDayRows+1), read.Log[0].At)
	assert.Equal(t, morning+1000, read.Log[len(read.Log)-1].At, "the newest is last")
	assert.Equal(t, "Yesterday's tenders from "+partner, read.Title)
	assert.Equal(t, "40 inbound tenders and 40 answers, the latest 20 shown", read.Source)
	assert.Nil(t, read.Figures[figureMedianAnswer], "no answer timed is no median, not zero")
}

func TestATenderWaitingOnAPersonNeedsAHand(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"MappingRequired", "PendingApproval", "Failed"} {
		row := answered("1", "", "", "", morning)
		row.TransferStatus = status
		row.AnswerID = ""
		assert.True(t, lineNeedsHand(&row), status)
	}
	declined := answered("2", "", "Sent", "Accepted", morning)
	declined.TransferStatus = "Rejected"
	assert.False(t, lineNeedsHand(&declined))
	entry := logEntry(context.Background(), partner, &declined)
	assert.Equal(t, "Tender 2 was declined; our answer was sent and acknowledged", entry.Text)
}

func TestATenderWhoseShipmentTheReaderMayNotSeeSaysNothingOfIt(t *testing.T) {
	t.Parallel()

	hidden := answered("88244", "", "Failed", "Pending", morning)
	hidden.ShipmentID = ""
	hidden.DeliveryError = "SFTP login refused"
	read := partnerDay(context.Background(), &partnerDayInput{
		partner: partner,
		period:  boardcatalog.PeriodToday,
		rows:    []tenderRow{hidden},
		figures: &dayFigures{
			Lines: 1, Tenders: 1, Shipments: 1, Visible: 0, Answers: 1, NeedHand: 1,
		},
	})

	require.Len(t, read.Log, 1)
	entry := read.Log[0]
	assert.Equal(t, "Tender 88244 was accepted; its shipment is not yours to read", entry.Text)
	assert.Empty(t, entry.Refs, "neither the shipment nor its 990 is named")
	assert.Empty(t, entry.Tone, "what became of an answer it may not read is not judged")
	assert.NotContains(t, entry.Detail, "SFTP")
	assert.Equal(t, boardcatalog.Count(1), *read.Figures[figureTendersIn],
		"the files are the partner's, read whole")
	for _, name := range []string{
		figureShipmentsMade, figureAnswersDelivered, figureNeedHand, figureMedianAnswer,
	} {
		assert.Nil(t, read.Figures[name], "%s covers a shipment the reader may not see", name)
	}
}
