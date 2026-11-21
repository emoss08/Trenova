package ediinboundservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	interchangeControlWidth = 9
	interchangeRejectedCode = "R"
)

type acknowledgmentRecord struct {
	tenantInfo   pagination.TenantInfo
	original     *edimessage.EDIMessage
	status       edimessage.MessageAcknowledgmentStatus
	detail       string
	ackMessageID pulid.ID
	receivedAt   int64
}

func (s *Service) routeAcknowledgment(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	message *edimessage.EDIMessage,
	transaction *parsedTransaction,
) []string {
	entries := parseAcknowledgments(transaction)
	if len(entries) == 0 {
		return []string{fmt.Sprintf(
			"acknowledgment %s/%s does not reference any transactions",
			transaction.set,
			transaction.controlNumber,
		)}
	}
	tenantInfo := inboundFileTenant(file)
	warnings := make([]string, 0)
	now := timeutils.NowUnix()
	for index := range entries {
		entry := &entries[index]
		originals, warning := s.acknowledgedOriginals(ctx, tenantInfo, partner, transaction, entry)
		if warning != "" {
			warnings = append(warnings, warning)
			continue
		}
		ackStatus, ackError := acknowledgmentResolution(entry)
		for _, original := range originals {
			if err := s.recordAcknowledgment(ctx, &acknowledgmentRecord{
				tenantInfo:   tenantInfo,
				original:     original,
				status:       ackStatus,
				detail:       ackError,
				ackMessageID: message.ID,
				receivedAt:   now,
			}); err != nil {
				warnings = append(warnings, fmt.Sprintf(
					"failed to record acknowledgment for message %s: %s",
					original.ID,
					err.Error(),
				))
			}
		}
	}
	return warnings
}

func (s *Service) acknowledgedOriginals(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partner *edi.EDIPartner,
	transaction *parsedTransaction,
	entry *acknowledgmentEntry,
) ([]*edimessage.EDIMessage, string) {
	if entry.originalControlNumber != "" {
		original, err := s.messageRepo.GetOutboundMessageForAck(
			ctx,
			&edimessageports.GetEDIOutboundMessageForAckRequest{
				TenantInfo:               tenantInfo,
				PartnerID:                partner.ID,
				TransactionSet:           transactionSetFromCode(entry.originalTransactionSet),
				GroupControlNumber:       entry.originalGroupControl,
				TransactionControlNumber: entry.originalControlNumber,
			},
		)
		if err != nil {
			return nil, fmt.Sprintf(
				"acknowledgment for control number %s could not be matched to an outbound message",
				entry.originalControlNumber,
			)
		}
		return []*edimessage.EDIMessage{original}, ""
	}
	if entry.originalGroupControl == "" {
		return nil, fmt.Sprintf(
			"acknowledgment %s/%s is missing the original transaction control number",
			transaction.set,
			transaction.controlNumber,
		)
	}
	originals, err := s.messageRepo.ListUnacknowledgedOutboundMessages(
		ctx,
		&edimessageports.ListUnacknowledgedOutboundMessagesRequest{
			TenantInfo:         tenantInfo,
			PartnerID:          partner.ID,
			GroupControlNumber: groupControlNumber(entry.originalGroupControl),
			TransactionSet:     transactionSetFromCode(entry.originalFunctionalGroupID),
		},
	)
	if err != nil {
		return nil, fmt.Sprintf(
			"acknowledgment for group %s could not be matched: %s",
			entry.originalGroupControl,
			err.Error(),
		)
	}
	if len(originals) == 0 {
		return nil, fmt.Sprintf(
			"acknowledgment for group %s matched no outbound message awaiting acknowledgment",
			entry.originalGroupControl,
		)
	}
	return originals, ""
}

func (s *Service) routeInterchangeAcknowledgments(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	interchange *parsedInterchange,
) []string {
	tenantInfo := inboundFileTenant(file)
	warnings := make([]string, 0)
	now := timeutils.NowUnix()
	for index := range interchange.interchangeAcks {
		ack := &interchange.interchangeAcks[index]
		if ack.code != interchangeRejectedCode {
			continue
		}
		control := interchangeControlNumber(ack.controlNumber)
		if control == "" {
			warnings = append(
				warnings,
				"a TA1 rejects an interchange without naming its control number",
			)
			continue
		}
		originals, err := s.messageRepo.ListUnacknowledgedOutboundMessages(
			ctx,
			&edimessageports.ListUnacknowledgedOutboundMessagesRequest{
				TenantInfo:               tenantInfo,
				PartnerID:                partner.ID,
				InterchangeControlNumber: control,
			},
		)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"TA1 rejecting interchange %s could not be matched: %s",
				control,
				err.Error(),
			))
			continue
		}
		if len(originals) == 0 {
			warnings = append(warnings, fmt.Sprintf(
				"TA1 rejecting interchange %s matched no outbound message awaiting acknowledgment",
				control,
			))
			continue
		}
		detail := fmt.Sprintf(
			"interchange %s rejected, TA1 note code %s",
			control,
			stringutils.FirstNonEmpty(ack.note, "missing"),
		)
		for _, original := range originals {
			if err = s.recordAcknowledgment(ctx, &acknowledgmentRecord{
				tenantInfo: tenantInfo,
				original:   original,
				status:     edimessage.MessageAcknowledgmentStatusRejected,
				detail:     detail,
				receivedAt: now,
			}); err != nil {
				warnings = append(warnings, fmt.Sprintf(
					"failed to record the TA1 rejection for message %s: %s",
					original.ID,
					err.Error(),
				))
			}
		}
	}
	return warnings
}

func (s *Service) recordAcknowledgment(ctx context.Context, record *acknowledgmentRecord) error {
	original := record.original
	if original.AckStatus == record.status && original.AckStatus.IsReceived() {
		return nil
	}
	var dispatched *edilading.Dispatched
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if _, err := s.messageRepo.UpdateMessageAcknowledgment(
			txCtx,
			&edimessageports.UpdateEDIMessageAcknowledgmentRequest{
				ID:            original.ID,
				TenantInfo:    record.tenantInfo,
				AckStatus:     record.status,
				AckMessageID:  record.ackMessageID,
				AckReceivedAt: &record.receivedAt,
				AckLastError:  record.detail,
			},
		); err != nil {
			return err
		}
		var err error
		dispatched, err = s.outbound.AcknowledgeFlowMessage(txCtx, &edioutbound.FlowAcknowledgment{
			TenantInfo: record.tenantInfo,
			Message:    original,
			Accepted:   record.status != edimessage.MessageAcknowledgmentStatusRejected,
			Detail:     record.detail,
		})
		return err
	})
	if err != nil {
		return err
	}
	if original.DeliverySentAt != nil && record.receivedAt >= *original.DeliverySentAt {
		s.metrics.RecordAckLatency(
			string(record.status),
			float64(record.receivedAt-*original.DeliverySentAt),
		)
	}
	if len(dispatched.Actions) == 0 {
		return nil
	}
	_, err = s.outbound.ExecuteFlowActions(ctx, record.tenantInfo, dispatched, nil)
	return err
}

func acknowledgmentResolution(
	entry *acknowledgmentEntry,
) (status edimessage.MessageAcknowledgmentStatus, ackError string) {
	code := entry.acknowledgmentCode
	if code == "" {
		code = entry.groupAcknowledgmentCode
	}
	switch code {
	case "A":
		return edimessage.MessageAcknowledgmentStatusAccepted, ""
	case "E":
		return edimessage.MessageAcknowledgmentStatusAcceptedWithErrors, acknowledgmentDetails(
			code,
			entry,
		)
	default:
		return edimessage.MessageAcknowledgmentStatusRejected, acknowledgmentDetails(code, entry)
	}
}

func acknowledgmentDetails(code string, entry *acknowledgmentEntry) string {
	details := make([]string, 0, len(entry.diagnostics)+2)
	details = append(details, "acknowledgment code "+stringutils.FirstNonEmpty(code, "missing"))
	if len(entry.syntaxErrors) > 0 {
		details = append(details, "syntax error codes "+strings.Join(entry.syntaxErrors, ", "))
	}
	for _, diagnostic := range entry.diagnostics {
		detail := diagnostic.Message
		if diagnostic.ErrorCode != "" {
			detail += " (code " + diagnostic.ErrorCode + ")"
		}
		if diagnostic.SegmentID != "" {
			detail += " segment " + diagnostic.SegmentID
		}
		details = append(details, detail)
	}

	return joinDetails(details)
}

func groupControlNumber(value string) string {
	trimmed := strings.TrimLeft(strings.TrimSpace(value), "0")
	if trimmed == "" && strings.TrimSpace(value) != "" {
		return "0"
	}
	return trimmed
}

func interchangeControlNumber(value string) string {
	trimmed := groupControlNumber(value)
	if trimmed == "" || len(trimmed) > interchangeControlWidth {
		return trimmed
	}
	return strings.Repeat("0", interchangeControlWidth-len(trimmed)) + trimmed
}
