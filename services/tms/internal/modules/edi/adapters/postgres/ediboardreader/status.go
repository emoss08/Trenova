package ediboardreader

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/i18n"
	"go.uber.org/fx"
)

const maxReasonRunes = 160

type Params struct {
	fx.In

	DB *postgres.Connection
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func firstLine(reason string) string {
	reason = strings.TrimSpace(reason)
	if cut := strings.IndexAny(reason, "\r\n"); cut >= 0 {
		reason = strings.TrimSpace(reason[:cut])
	}
	if utf8.RuneCountInString(reason) <= maxReasonRunes {
		return reason
	}
	runes := []rune(reason)

	return strings.TrimSpace(string(runes[:maxReasonRunes-1])) + "…"
}

func fileStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch ediinboundfile.InboundFileStatus(status) {
	case ediinboundfile.InboundFileStatusReceived:
		return i18n.T(ctx, "Received"), boardcatalog.ToneInfo
	case ediinboundfile.InboundFileStatusParsed:
		return i18n.T(ctx, "Handling its sets"), boardcatalog.ToneInfo
	case ediinboundfile.InboundFileStatusProcessed:
		return i18n.T(ctx, "Processed"), boardcatalog.ToneOK
	case ediinboundfile.InboundFileStatusPartiallyProcessed:
		return i18n.T(ctx, "Partly processed"), boardcatalog.ToneWarn
	case ediinboundfile.InboundFileStatusQuarantined:
		return i18n.T(ctx, "Quarantined"), boardcatalog.ToneWarn
	case ediinboundfile.InboundFileStatusDuplicate:
		return i18n.T(ctx, "Duplicate interchange"), ""
	}

	return status, ""
}

func fileNeedsHand(status string) bool {
	switch ediinboundfile.InboundFileStatus(status) {
	case ediinboundfile.InboundFileStatusQuarantined,
		ediinboundfile.InboundFileStatusPartiallyProcessed:
		return true
	case ediinboundfile.InboundFileStatusReceived,
		ediinboundfile.InboundFileStatusParsed,
		ediinboundfile.InboundFileStatusProcessed,
		ediinboundfile.InboundFileStatusDuplicate:
	}

	return false
}

func fileUnderWay(status string) bool {
	switch ediinboundfile.InboundFileStatus(status) {
	case ediinboundfile.InboundFileStatusReceived, ediinboundfile.InboundFileStatusParsed:
		return true
	case ediinboundfile.InboundFileStatusProcessed,
		ediinboundfile.InboundFileStatusPartiallyProcessed,
		ediinboundfile.InboundFileStatusQuarantined,
		ediinboundfile.InboundFileStatusDuplicate:
	}

	return false
}

type messageState struct {
	Direction string
	Status    string
	Delivery  string
	Ack       string
}

func messageStatus(ctx context.Context, state *messageState) (string, boardcatalog.Tone) {
	inbound := edidocument.DocumentDirection(state.Direction) ==
		edidocument.DocumentDirectionInbound
	if edimessage.MessageStatus(state.Status) == edimessage.MessageStatusFailed {
		if inbound {
			return i18n.T(ctx, "Could not be read"), boardcatalog.ToneWarn
		}
		return i18n.T(ctx, "Not generated"), boardcatalog.ToneWarn
	}
	if inbound {
		return i18n.T(ctx, "Received"), boardcatalog.ToneOK
	}
	if label, tone, ok := deliveryStatus(ctx, state.Delivery); ok {
		return label, tone
	}

	return answerStatus(ctx, state)
}

func deliveryStatus(ctx context.Context, delivery string) (string, boardcatalog.Tone, bool) {
	switch edimessage.MessageDeliveryStatus(delivery) {
	case edimessage.MessageDeliveryStatusDeadLettered:
		return i18n.T(ctx, "Dead-lettered"), boardcatalog.ToneWarn, true
	case edimessage.MessageDeliveryStatusFailed:
		return i18n.T(ctx, "Delivery failed"), boardcatalog.ToneWarn, true
	case edimessage.MessageDeliveryStatusQueued:
		return i18n.T(ctx, "Queued"), "", true
	case edimessage.MessageDeliveryStatusSending:
		return i18n.T(ctx, "Sending"), boardcatalog.ToneInfo, true
	case edimessage.MessageDeliveryStatusSent:
	}

	return "", "", false
}

func answerStatus(ctx context.Context, state *messageState) (string, boardcatalog.Tone) {
	sent := edimessage.MessageDeliveryStatus(state.Delivery) == edimessage.MessageDeliveryStatusSent
	switch edimessage.MessageAcknowledgmentStatus(state.Ack) {
	case edimessage.MessageAcknowledgmentStatusRejected:
		return i18n.T(ctx, "Rejected by partner"), boardcatalog.ToneWarn
	case edimessage.MessageAcknowledgmentStatusFailed:
		return i18n.T(ctx, "Acknowledgment unreadable"), boardcatalog.ToneWarn
	case edimessage.MessageAcknowledgmentStatusAcceptedWithErrors:
		return i18n.T(ctx, "Accepted with errors"), boardcatalog.ToneWarn
	case edimessage.MessageAcknowledgmentStatusAccepted:
		return i18n.T(ctx, "Accepted"), boardcatalog.ToneOK
	case edimessage.MessageAcknowledgmentStatusPending:
		if sent {
			return i18n.T(ctx, "Awaiting acknowledgment"), boardcatalog.ToneInfo
		}
	case edimessage.MessageAcknowledgmentStatusNotExpected:
	}
	if sent {
		return i18n.T(ctx, "Delivered"), boardcatalog.ToneOK
	}

	return i18n.T(ctx, "Generated"), ""
}

func messageNeedsHand(state *messageState) bool {
	if edimessage.MessageStatus(state.Status) == edimessage.MessageStatusFailed {
		return true
	}
	if edidocument.DocumentDirection(state.Direction) == edidocument.DocumentDirectionInbound {
		return false
	}
	if edimessage.MessageDeliveryStatus(state.Delivery).IsRetryable() {
		return true
	}
	switch edimessage.MessageAcknowledgmentStatus(state.Ack) {
	case edimessage.MessageAcknowledgmentStatusRejected,
		edimessage.MessageAcknowledgmentStatusFailed,
		edimessage.MessageAcknowledgmentStatusAcceptedWithErrors:
		return true
	case edimessage.MessageAcknowledgmentStatusNotExpected,
		edimessage.MessageAcknowledgmentStatusPending,
		edimessage.MessageAcknowledgmentStatusAccepted:
	}

	return false
}
