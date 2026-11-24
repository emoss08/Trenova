package ediboardreader

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/ediboard"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	figureTendersIn        = "tendersIn"
	figureShipmentsMade    = "shipmentsMade"
	figureAnswersDelivered = "answersDelivered"
	figureNeedHand         = "needHand"
	figureMedianAnswer     = "medianAnswer"
)

var shipmentKind = recordkind.Kind(permission.ResourceShipment)

type partnerDayInput struct {
	partner string
	period  boardcatalog.Period
	rows    []tenderRow
	figures *dayFigures
}

func partnerDay(ctx context.Context, in *partnerDayInput) *boardcatalog.MetricRead {
	lines := distinctLines(in.rows)
	lines = lines[:min(len(lines), ediboard.PartnerDayRows)]
	slices.Reverse(lines)
	entries := make([]boardcatalog.LogEntry, 0, len(lines))
	for idx := range lines {
		entries = append(entries, logEntry(ctx, in.partner, &lines[idx]))
	}
	figures := in.figures
	whole := figures.Shipments == figures.Visible
	count := func(n int64, readable bool) *boardcatalog.Value {
		if !readable {
			return nil
		}
		value := boardcatalog.Count(n)
		return &value
	}
	median := medianSeconds(figures.Median)
	if !whole {
		median = nil
	}

	return &boardcatalog.MetricRead{
		Title:  dayTitle(ctx, in.period, in.partner),
		Source: daySource(ctx, figures, len(entries)),
		Figures: map[string]*boardcatalog.Value{
			figureTendersIn:        count(figures.Tenders, true),
			figureShipmentsMade:    count(figures.Shipments, whole),
			figureAnswersDelivered: count(figures.Delivered, whole),
			figureNeedHand:         count(figures.NeedHand, whole),
			figureMedianAnswer:     median,
		},
		Log: entries,
	}
}

func distinctLines(rows []tenderRow) []tenderRow {
	type key struct{ file, tender pulid.ID }
	seen := make(map[key]struct{}, len(rows))
	out := make([]tenderRow, 0, len(rows))
	for idx := range rows {
		line := key{file: rows[idx].FileID, tender: rows[idx].TenderID}
		if _, dup := seen[line]; dup {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, rows[idx])
	}

	return out
}

func dayTitle(ctx context.Context, period boardcatalog.Period, partner string) string {
	if period == boardcatalog.PeriodYesterday {
		return i18n.T(ctx, "Yesterday's tenders from {0}", partner)
	}

	return i18n.T(ctx, "Today's tenders from {0}", partner)
}

func daySource(ctx context.Context, figures *dayFigures, shown int) string {
	if figures.Lines > int64(shown) {
		return i18n.T(ctx,
			"{0, plural, one {# inbound tender} other {# inbound tenders}} and "+
				"{1, plural, one {# answer} other {# answers}}, the latest {2} shown",
			figures.Tenders, figures.Answers, shown)
	}

	return i18n.T(ctx,
		"{0, plural, one {# inbound tender} other {# inbound tenders}} and "+
			"{1, plural, one {# answer} other {# answers}}",
		figures.Tenders, figures.Answers)
}

func logEntry(ctx context.Context, partner string, row *tenderRow) boardcatalog.LogEntry {
	entry := boardcatalog.LogEntry{At: row.ReceivedAt}
	switch {
	case row.TenderID.IsNil():
		fileEntry(ctx, partner, row, &entry)
	case row.TransferID.IsNil():
		unopenedEntry(ctx, row, &entry)
	default:
		transferEntry(ctx, partner, row, &entry)
	}
	if lineNeedsHand(row) {
		entry.Tone = boardcatalog.ToneWarn
	}

	return entry
}

func tenderName(ctx context.Context, row *tenderRow) string {
	if bol := strings.TrimSpace(row.BOL); bol != "" {
		return i18n.T(ctx, "Tender {0}", bol)
	}

	return i18n.T(ctx, "A tender")
}

func fileRef(row *tenderRow) boardcatalog.Ref {
	return boardcatalog.Ref{Kind: recordkind.EDIInboundFile, ID: row.FileID}
}

func fileEntry(ctx context.Context, partner string, row *tenderRow, entry *boardcatalog.LogEntry) {
	reason := firstLine(row.FileReason)
	entry.Refs = []boardcatalog.Ref{fileRef(row)}
	switch ediinboundfile.InboundFileStatus(row.FileStatus) {
	case ediinboundfile.InboundFileStatusQuarantined:
		entry.Text = withReason(
			ctx,
			i18n.T(ctx, "A file from {0} was quarantined", partner),
			reason,
		)
		entry.Detail = i18n.T(ctx, "Inbound file quarantined, nothing read from it")
	case ediinboundfile.InboundFileStatusPartiallyProcessed:
		entry.Text = withReason(ctx, i18n.T(ctx, "A file from {0} was partly processed", partner),
			reason)
		entry.Detail = i18n.T(ctx, "Inbound file partly processed, no tender read from it")
	case ediinboundfile.InboundFileStatusReceived, ediinboundfile.InboundFileStatusParsed:
		entry.Text = i18n.T(ctx, "A file from {0} is still being read", partner)
		entry.Detail = i18n.T(ctx, "Inbound file received")
	case ediinboundfile.InboundFileStatusProcessed, ediinboundfile.InboundFileStatusDuplicate:
		entry.Text = i18n.T(ctx, "A file from {0} held no tender", partner)
		entry.Detail = i18n.T(ctx, "Inbound file processed")
	}
}

func unopenedEntry(ctx context.Context, row *tenderRow, entry *boardcatalog.LogEntry) {
	name := tenderName(ctx, row)
	reason := firstLine(row.FileReason)
	entry.Refs = []boardcatalog.Ref{fileRef(row)}
	switch {
	case ediinboundfile.InboundFileStatus(row.FileStatus) ==
		ediinboundfile.InboundFileStatusQuarantined:
		entry.Text = withReason(ctx, i18n.T(ctx, "{0} was quarantined", name), reason)
		entry.Detail = i18n.T(ctx, "204 in, file quarantined, no shipment made")
	case edimessage.MessageStatus(row.TenderStatus) == edimessage.MessageStatusFailed:
		entry.Text = withReason(ctx, i18n.T(ctx, "{0} could not be read", name), reason)
		entry.Detail = i18n.T(ctx, "204 in, not read, no shipment made")
	case fileUnderWay(row.FileStatus):
		entry.Text = i18n.T(ctx, "{0} is being read", name)
		entry.Detail = i18n.T(ctx, "204 in")
	default:
		entry.Text = withReason(ctx, i18n.T(ctx, "{0} was kept without being processed", name),
			reason)
		entry.Detail = i18n.T(ctx, "204 in, no shipment made")
	}
}

func transferEntry(
	ctx context.Context,
	partner string,
	row *tenderRow,
	entry *boardcatalog.LogEntry,
) {
	name := tenderName(ctx, row)
	reason := firstLine(row.TransferReason)
	switch editransfer.TransferStatus(row.TransferStatus) {
	case editransfer.TransferStatusApproved:
		if shipmentHidden(row) {
			entry.Text = i18n.T(ctx, "{0} was accepted; its shipment is not yours to read", name)
			entry.Detail = i18n.T(ctx, "204 in")
			return
		}
		accepted := i18n.T(ctx, "{0} was accepted", name)
		if pro := strings.TrimSpace(row.ProNumber); pro != "" {
			accepted = i18n.T(ctx, "{0} became {1}", name, pro)
			entry.Refs = append(
				entry.Refs,
				boardcatalog.Ref{Kind: shipmentKind, ID: row.ShipmentID},
			)
		}
		answerEntry(ctx, partner, row, accepted, entry)
	case editransfer.TransferStatusRejected:
		answerEntry(ctx, partner, row, i18n.T(ctx, "{0} was declined", name), entry)
	case editransfer.TransferStatusMappingRequired:
		entry.Text = i18n.T(ctx, "{0} waits on mapping before it can be accepted", name)
		entry.Detail = i18n.T(ctx, "204 in, mapping required, no shipment yet")
		entry.Refs = []boardcatalog.Ref{fileRef(row)}
	case editransfer.TransferStatusPendingApproval:
		entry.Text = i18n.T(ctx, "{0} waits for approval", name)
		entry.Detail = i18n.T(ctx, "204 in, waiting for approval, no shipment yet")
	case editransfer.TransferStatusSubmitted, editransfer.TransferStatusProcessing:
		entry.Text = i18n.T(ctx, "{0} is being accepted", name)
		entry.Detail = i18n.T(ctx, "204 in, no shipment yet")
	case editransfer.TransferStatusFailed:
		entry.Text = withReason(ctx, i18n.T(ctx, "{0} could not be accepted", name), reason)
		entry.Detail = i18n.T(ctx, "204 in, no shipment made")
		entry.Refs = []boardcatalog.Ref{fileRef(row)}
	case editransfer.TransferStatusExpired:
		entry.Text = i18n.T(ctx, "{0} expired before it was answered", name)
		entry.Detail = i18n.T(ctx, "204 in, no shipment made")
	case editransfer.TransferStatusCanceled:
		entry.Text = i18n.T(ctx, "{0} was canceled", name)
		entry.Detail = i18n.T(ctx, "204 in, no shipment made")
	}
}

func answerEntry(
	ctx context.Context,
	partner string,
	row *tenderRow,
	opening string,
	entry *boardcatalog.LogEntry,
) {
	answerRef := boardcatalog.Ref{Kind: recordkind.EDIMessage, ID: row.AnswerID}
	if row.AnswerID.IsNil() {
		entry.Text = i18n.T(ctx, "{0}; our answer has not been sent", opening)
		entry.Detail = i18n.T(ctx, "204 in, no 990 yet")
		return
	}
	if edimessage.MessageStatus(row.AnswerStatus) == edimessage.MessageStatusFailed {
		entry.Text = i18n.T(ctx, "{0}, but our answer could not be written", opening)
		entry.Detail = i18n.T(ctx, "990 not generated")
		entry.Refs = append(entry.Refs, answerRef)
		return
	}
	switch edimessage.MessageDeliveryStatus(row.Delivery) {
	case edimessage.MessageDeliveryStatusFailed, edimessage.MessageDeliveryStatusDeadLettered:
		entry.Text = i18n.T(ctx, "{0}, but our answer did not reach {1}", opening, partner)
		entry.Detail = withReason(
			ctx,
			deliveryFailure(ctx, row.Delivery),
			firstLine(row.DeliveryError),
		)
		entry.Refs = append(entry.Refs, answerRef)
		return
	case edimessage.MessageDeliveryStatusQueued, edimessage.MessageDeliveryStatusSending:
		entry.Text = i18n.T(ctx, "{0}; our answer is on its way", opening)
		entry.Detail = i18n.T(ctx, "204 in, 990 queued")
		return
	case edimessage.MessageDeliveryStatusSent:
	}
	acknowledged(ctx, partner, row, opening, entry)
	if lineNeedsHand(row) {
		entry.Refs = append(entry.Refs, answerRef)
	}
}

func acknowledged(
	ctx context.Context,
	partner string,
	row *tenderRow,
	opening string,
	entry *boardcatalog.LogEntry,
) {
	ackError := firstLine(row.AckError)
	switch edimessage.MessageAcknowledgmentStatus(row.Ack) {
	case edimessage.MessageAcknowledgmentStatusAccepted:
		entry.Text = i18n.T(ctx, "{0}; our answer was sent and acknowledged", opening)
		entry.Detail = i18n.T(ctx, "204 in, 990 out, 997 accepted")
	case edimessage.MessageAcknowledgmentStatusPending:
		entry.Text = i18n.T(ctx, "{0}; our answer was sent", opening)
		entry.Detail = i18n.T(ctx, "204 in, 990 out, acknowledgment pending")
	case edimessage.MessageAcknowledgmentStatusAcceptedWithErrors:
		entry.Text = i18n.T(ctx, "{0}; {1} acknowledged our answer with errors", opening, partner)
		entry.Detail = withReason(ctx, i18n.T(ctx, "997 accepted with errors"), ackError)
	case edimessage.MessageAcknowledgmentStatusRejected:
		entry.Text = i18n.T(ctx, "{0}, but {1} rejected our answer", opening, partner)
		entry.Detail = withReason(ctx, i18n.T(ctx, "997 rejected"), ackError)
	case edimessage.MessageAcknowledgmentStatusFailed:
		entry.Text = i18n.T(ctx, "{0}; the acknowledgment of our answer could not be read",
			opening)
		entry.Detail = withReason(ctx, i18n.T(ctx, "997 unreadable"), ackError)
	case edimessage.MessageAcknowledgmentStatusNotExpected:
		entry.Text = i18n.T(ctx, "{0}; our answer was sent", opening)
		entry.Detail = i18n.T(ctx, "204 in, 990 out")
	default:
		entry.Text = i18n.T(ctx, "{0}; our answer was sent", opening)
		entry.Detail = i18n.T(ctx, "204 in, 990 out")
	}
}

func deliveryFailure(ctx context.Context, delivery string) string {
	if edimessage.MessageDeliveryStatus(delivery) == edimessage.MessageDeliveryStatusDeadLettered {
		return i18n.T(ctx, "990 dead-lettered")
	}

	return i18n.T(ctx, "990 failed")
}

func withReason(ctx context.Context, words, reason string) string {
	if reason == "" {
		return words
	}

	return i18n.T(ctx, "{0}: {1}", words, reason)
}

func lineNeedsHand(row *tenderRow) bool {
	if fileNeedsHand(row.FileStatus) ||
		edimessage.MessageStatus(row.TenderStatus) == edimessage.MessageStatusFailed {
		return true
	}
	switch editransfer.TransferStatus(row.TransferStatus) {
	case editransfer.TransferStatusMappingRequired,
		editransfer.TransferStatusPendingApproval,
		editransfer.TransferStatusFailed:
		return true
	case editransfer.TransferStatusSubmitted,
		editransfer.TransferStatusProcessing,
		editransfer.TransferStatusApproved,
		editransfer.TransferStatusRejected,
		editransfer.TransferStatusExpired,
		editransfer.TransferStatusCanceled:
	}
	if row.AnswerID.IsNil() || shipmentHidden(row) {
		return false
	}

	return messageNeedsHand(&messageState{
		Direction: string(edidocument.DocumentDirectionOutbound),
		Status:    row.AnswerStatus,
		Delivery:  row.Delivery,
		Ack:       row.Ack,
	})
}

func shipmentHidden(row *tenderRow) bool {
	return row.MadeShipment.IsNotNil() && row.ShipmentID.IsNil()
}
