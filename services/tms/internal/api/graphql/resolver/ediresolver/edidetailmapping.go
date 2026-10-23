package ediresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/shopspring/decimal"
)

const (
	maxRawBytes          = 16 * 1024
	maxDetailMessages    = 50
	maxDetailFindings    = 50
	defaultSinceHours    = 24
	maxSinceHours        = 30 * 24
	minOverdueSeconds    = 60
	maxOverdueSeconds    = 30 * 24 * 60 * 60
	ackHealthOverdueRows = 25
)

// callerOrganization is the organization the read is made for, which a
// transfer's direction and counterparty are judged from: the tender's
// receiver sees it inbound.
func callerOrganization(ctx context.Context) pulid.ID {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return pulid.Nil
	}

	return authCtx.OrganizationID
}

func optionalText(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func optionalInt[T ~int16 | ~int32 | ~int64](value *T) *int {
	if value == nil {
		return nil
	}
	out := int(*value)

	return &out
}

func tenderStopCount(payload *edi.LoadTenderPayload) int {
	count := 0
	for idx := range payload.Moves {
		count += len(payload.Moves[idx].Stops)
	}

	return count
}

func tenderStopsToModel(payload *edi.LoadTenderPayload) []*gqlmodel.EDITenderStop {
	out := make([]*gqlmodel.EDITenderStop, 0, tenderStopCount(payload))
	for moveIdx := range payload.Moves {
		stops := payload.Moves[moveIdx].Stops
		for stopIdx := range stops {
			stop := &stops[stopIdx]
			opens := int(stop.ScheduledWindowStart)
			model := &gqlmodel.EDITenderStop{
				Sequence: int(stop.Sequence),
				Type:     stop.Type,
				Location: optionalText(stringutils.FirstNonEmpty(
					stop.LocationName, stop.LocationLabel, stop.LocationCode,
				)),
				Address: optionalText(stringutils.FirstNonEmpty(
					stop.LocationAddressLine1, stop.AddressLine,
				)),
				City:         optionalText(stop.LocationCity),
				State:        optionalText(stop.LocationStateCode),
				PostalCode:   optionalText(stop.LocationPostalCode),
				WindowCloses: optionalInt(stop.ScheduledWindowEnd),
				Pieces:       optionalInt(stop.Pieces),
				Weight:       optionalInt(stop.Weight),
			}
			if opens > 0 {
				model.WindowOpens = &opens
			}
			out = append(out, model)
		}
	}

	return out
}

func tenderCommoditiesToModel(payload *edi.LoadTenderPayload) []*gqlmodel.EDITenderCommodity {
	out := make([]*gqlmodel.EDITenderCommodity, 0, len(payload.Commodities))
	for idx := range payload.Commodities {
		commodity := &payload.Commodities[idx]
		out = append(out, &gqlmodel.EDITenderCommodity{
			Commodity: stringutils.FirstNonEmpty(
				commodity.CommodityLabel,
				commodity.CommodityName,
				commodity.CommodityDescription,
			),
			Weight: int(commodity.Weight),
			Pieces: int(commodity.Pieces),
		})
	}

	return out
}

func tenderChargesToModel(payload *edi.LoadTenderPayload) *gqlmodel.EDITenderCharges {
	charges := &gqlmodel.EDITenderCharges{
		Freight:    fixedAmount(payload.FreightChargeAmount),
		Other:      fixedAmount(payload.OtherChargeAmount),
		Total:      fixedAmount(payload.TotalChargeAmount),
		Additional: make([]*gqlmodel.EDITenderCharge, 0, len(payload.AdditionalCharges)),
	}
	for idx := range payload.AdditionalCharges {
		charge := &payload.AdditionalCharges[idx]
		charges.Additional = append(charges.Additional, &gqlmodel.EDITenderCharge{
			Charge: stringutils.FirstNonEmpty(
				charge.AccessorialLabel,
				charge.AccessorialCode,
				charge.AccessorialDescription,
			),
			Method: optionalText(charge.Method),
			Amount: charge.Amount.StringFixed(2),
		})
	}

	return charges
}

func fixedAmount(amount decimal.NullDecimal) *string {
	if !amount.Valid {
		return nil
	}
	text := amount.Decimal.StringFixed(2)

	return &text
}

func mappingNeedsToModel(needs []edi.MappingResolution) []*gqlmodel.EDITransferMappingNeed {
	out := make([]*gqlmodel.EDITransferMappingNeed, 0, len(needs))
	for idx := range needs {
		need := &needs[idx]
		out = append(out, &gqlmodel.EDITransferMappingNeed{
			EntityType: need.EntityType,
			Source:     stringutils.FirstNonEmpty(need.SourceLabel, need.SourceID.String()),
		})
	}

	return out
}

// rawText is the first 16 KB of a message's or file's raw X12, or nothing
// once it was purged under the retention policy.
func rawText(raw string, purgedAt *int64) *gqlmodel.EDIRawText {
	if purgedAt != nil || raw == "" {
		return nil
	}
	text := stringutils.TruncateBytes(raw, maxRawBytes)

	return &gqlmodel.EDIRawText{
		Text:      text,
		Truncated: len(text) < len(raw),
		SizeBytes: len(raw),
	}
}

func ediMessageDetailToModel(message *edi.EDIMessage) *gqlmodel.EDIMessageDetail {
	detail := &gqlmodel.EDIMessageDetail{
		Message:         message,
		DiagnosticCount: len(message.ValidationErrors),
		Diagnostics: make([]*gqlmodel.EDIMessageDiagnostic, 0,
			min(len(message.ValidationErrors), maxDetailFindings)),
		Raw: rawText(message.RawX12, message.RawPurgedAt),
	}
	if message.PayloadSnapshot.TransactionSet != "" {
		detail.Decoded = message.PayloadSnapshot
	}
	for _, finding := range message.ValidationErrors {
		if finding == nil || len(detail.Diagnostics) == maxDetailFindings {
			continue
		}
		model := &gqlmodel.EDIMessageDiagnostic{
			Severity:     string(finding.Severity),
			Code:         finding.Code,
			SegmentID:    optionalText(finding.SegmentID),
			Path:         optionalText(finding.Path),
			Message:      finding.Message,
			SuggestedFix: optionalText(finding.SuggestedFix),
		}
		if finding.ElementPosition > 0 {
			position := finding.ElementPosition
			model.ElementPosition = &position
		}
		detail.Diagnostics = append(detail.Diagnostics, model)
	}

	return detail
}

func ediInboundFileDetailToModel(file *edi.EDIInboundFile) *gqlmodel.EDIInboundFileDetail {
	detail := &gqlmodel.EDIInboundFileDetail{
		File:         file,
		MessageCount: len(file.Messages),
		Messages:     make([]*edi.EDIMessage, 0, min(len(file.Messages), maxDetailMessages)),
		Raw:          rawText(file.RawContent, file.RawPurgedAt),
	}
	if file.CommunicationProfile != nil {
		detail.CommunicationProfileName = optionalText(file.CommunicationProfile.Name)
	}
	for _, message := range file.Messages {
		if message == nil || len(detail.Messages) == maxDetailMessages {
			continue
		}
		detail.Messages = append(detail.Messages, message)
	}

	return detail
}

type ackHealthArguments struct {
	tenant              pagination.TenantInfo
	partnerID           *string
	transactionSet      *string
	overdueAfterSeconds *int
	sinceHours          *int
}

// ackHealthRequest reads the acknowledgment health read's arguments,
// refusing a transaction set, overdue age or window outside what it reads.
func ackHealthRequest(args *ackHealthArguments) (*ediservice.GetEDIAckHealthRequest, error) {
	partnerID, err := base.ParseOptionalIDField("partnerId", args.partnerID)
	if err != nil {
		return nil, err
	}
	req := &ediservice.GetEDIAckHealthRequest{
		TenantInfo:   args.tenant,
		PartnerID:    partnerID,
		OverdueLimit: ackHealthOverdueRows,
	}
	if set := typeutils.ValueOrZero(args.transactionSet); set != "" {
		req.TransactionSet = edi.TransactionSet(set)
		if !req.TransactionSet.IsValid() {
			return nil, errortypes.NewValidationError("transactionSet", errortypes.ErrInvalid,
				"Transaction set must be 204, 210, 214, 990, 997 or 999")
		}
	}
	if args.overdueAfterSeconds != nil {
		seconds := *args.overdueAfterSeconds
		if seconds < minOverdueSeconds || seconds > maxOverdueSeconds {
			return nil, errortypes.NewValidationError("overdueAfterSeconds", errortypes.ErrInvalid,
				"Overdue age must be between 60 and 2592000 seconds")
		}
		req.OverdueAckAge = int64(seconds)
	}
	hours := defaultSinceHours
	if args.sinceHours != nil {
		hours = *args.sinceHours
		if hours < 1 || hours > maxSinceHours {
			return nil, errortypes.NewValidationError("sinceHours", errortypes.ErrInvalid,
				"The window must be between 1 and 720 hours")
		}
	}
	req.Since = timeutils.NowUnix() - int64(hours)*60*60

	return req, nil
}

func ediAckHealthToModel(health *ediservice.EDIAckHealth) *gqlmodel.EDIAckHealth {
	model := &gqlmodel.EDIAckHealth{
		AsOf:                int(health.AsOf),
		Since:               int(health.Since),
		OverdueAfterSeconds: int(health.OverdueAckAge),
		AnswersEverySet:     health.AnswersEverySet,
		AckStatusCounts:     ackStatusCountsToModel(health.AckStatusCounts),
		Partners: make(
			[]*gqlmodel.EDIPartnerAckCounts,
			0,
			len(health.AckCountsByPartner),
		),
		OverdueCount: health.OverdueAckCount,
		OverdueByPartner: make(
			[]*gqlmodel.EDIOverduePartnerAcks,
			0,
			len(health.OverdueByPartner),
		),
		OverdueNotListed:   health.OverdueOmitted,
		RecentDeadLettered: nonNilSlice(health.RecentDeadLettered),
	}
	for _, group := range health.AckCountsByPartner {
		if group == nil {
			continue
		}
		model.Partners = append(model.Partners, &gqlmodel.EDIPartnerAckCounts{
			PartnerID:       group.PartnerID.String(),
			PartnerName:     group.PartnerName,
			PartnerCode:     group.PartnerCode,
			AckStatusCounts: ackStatusCountsToModel(group.Counts),
		})
	}
	for _, group := range health.OverdueByPartner {
		if group == nil {
			continue
		}
		model.OverdueByPartner = append(model.OverdueByPartner, &gqlmodel.EDIOverduePartnerAcks{
			PartnerID:         group.PartnerID.String(),
			PartnerName:       group.PartnerName,
			PartnerCode:       group.PartnerCode,
			OverdueCount:      group.OverdueCount,
			OldestGeneratedAt: int(group.OldestGeneratedAt),
			Messages:          nonNilSlice(group.Messages),
		})
	}

	return model
}

// ackStatusOrder is the order acknowledgment counts are read in: what is
// still awaited, then what came back, then what will not.
var ackStatusOrder = [...]edi.MessageAcknowledgmentStatus{
	edi.MessageAcknowledgmentStatusPending,
	edi.MessageAcknowledgmentStatusAccepted,
	edi.MessageAcknowledgmentStatusAcceptedWithErrors,
	edi.MessageAcknowledgmentStatusRejected,
	edi.MessageAcknowledgmentStatusFailed,
	edi.MessageAcknowledgmentStatusNotExpected,
}

func ackStatusCountsToModel(
	counts map[edi.MessageAcknowledgmentStatus]int,
) []*gqlmodel.EDIAckStatusCount {
	out := make([]*gqlmodel.EDIAckStatusCount, 0, len(ackStatusOrder))
	for _, status := range ackStatusOrder {
		out = append(out, &gqlmodel.EDIAckStatusCount{Status: status, Count: counts[status]})
	}

	return out
}
