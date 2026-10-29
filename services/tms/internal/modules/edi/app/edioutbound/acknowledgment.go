package edioutbound

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

type GenerateAcknowledgmentRequest struct {
	TenantInfo           pagination.TenantInfo
	EDIPartnerID         pulid.ID
	Type                 edidocument.AcknowledgmentType
	Group                *x12.Group
	SourceMessageID      pulid.ID
	GeneratedByID        pulid.ID
	IdempotencyKey       string
	DisableDeliveryQueue bool
}

type acknowledgmentControlNumbers struct {
	interchange string
	group       string
	transaction string
}

var provisionalAcknowledgmentNumbers = acknowledgmentControlNumbers{
	interchange: "000000000",
	group:       "0",
	transaction: "0000",
}

func acknowledgmentSet(
	ackType edidocument.AcknowledgmentType,
) (edidocument.TransactionSet, x12.AcknowledgmentKind, error) {
	switch ackType {
	case edidocument.AcknowledgmentType997:
		return edidocument.TransactionSet997, x12.AcknowledgmentKind997, nil
	case edidocument.AcknowledgmentType999:
		return edidocument.TransactionSet999, x12.AcknowledgmentKind999, nil
	case edidocument.AcknowledgmentTypeNone:
	}
	return "", "", errortypes.NewValidationError(
		"acknowledgment.type",
		errortypes.ErrInvalid,
		"Acknowledgment type must be 997 or 999",
	)
}

func (s *Service) GenerateAcknowledgment(
	ctx context.Context,
	req *GenerateAcknowledgmentRequest,
) (*edimessage.EDIMessage, error) {
	ackSet, kind, err := acknowledgmentSet(req.Type)
	if err != nil {
		return nil, err
	}
	if req.Group == nil {
		return nil, errortypes.NewValidationError(
			"group",
			errortypes.ErrRequired,
			"The functional group to acknowledge is required",
		)
	}
	if existing, found, lookupErr := s.generatedDocument(ctx, &GenerateEDIDocumentRequest{
		TenantInfo:           req.TenantInfo,
		TransactionSet:       ackSet,
		IdempotencyKey:       req.IdempotencyKey,
		DisableDeliveryQueue: req.DisableDeliveryQueue,
	}); lookupErr != nil || found {
		return existing, lookupErr
	}

	envelope, program, err := s.partnerEnvelope(ctx, req.TenantInfo, req.EDIPartnerID)
	if err != nil {
		return nil, err
	}
	documentType, err := s.documentTypeFor(ctx, ackSet, edidocument.DocumentDirectionOutbound)
	if err != nil {
		return nil, err
	}
	numbers, err := s.allocateAcknowledgmentNumbers(
		ctx,
		req.TenantInfo,
		req.EDIPartnerID,
		documentType.ID,
	)
	if err != nil {
		return nil, err
	}
	written, err := writeAcknowledgment(ctx, &envelope, kind, req.Group, numbers)
	if err != nil {
		return nil, err
	}

	message := &edimessage.EDIMessage{
		BusinessUnitID:           req.TenantInfo.BuID,
		OrganizationID:           req.TenantInfo.OrgID,
		EDIPartnerID:             req.EDIPartnerID,
		DocumentTypeID:           documentType.ID,
		LadingVersionID:          program.Version.ID,
		Direction:                edidocument.DocumentDirectionOutbound,
		Standard:                 edidocument.EDIStandardX12,
		TransactionSet:           ackSet,
		X12Version:               acknowledgmentVersion(kind),
		Status:                   edimessage.MessageStatusGenerated,
		InterchangeControlNumber: numbers.interchange,
		GroupControlNumber:       numbers.group,
		TransactionControlNumber: numbers.transaction,
		SegmentCount:             int64(written.SegmentCount),
		RawX12:                   written.Text,
		PayloadSnapshot:          acknowledgmentPayload(ackSet, req.SourceMessageID, req.Group),
		AckStatus:                edimessage.MessageAcknowledgmentStatusNotExpected,
		GeneratedByID:            req.GeneratedByID,
		IdempotencyKey:           req.IdempotencyKey,
	}
	created, replayed, err := s.createAcknowledgmentMessage(ctx, req, message)
	if err != nil || replayed {
		return created, err
	}
	if !req.DisableDeliveryQueue {
		if err = s.delivery.QueueMessage(ctx, created); err != nil {
			s.l.Warn(
				"failed to queue EDI acknowledgment for delivery",
				zap.String("messageId", created.ID.String()),
				zap.Error(err),
			)
		}
	}
	return created, nil
}

func (s *Service) allocateAcknowledgmentNumbers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
	documentTypeID pulid.ID,
) (acknowledgmentControlNumbers, error) {
	allocated, err := s.controlNumberRepo.AllocateControlNumbers(
		ctx,
		edipartnerports.AllocateEDIControlNumbersRequest{
			TenantInfo:     tenantInfo,
			PartnerID:      partnerID,
			DocumentTypeID: documentTypeID,
			Kinds: []edidocument.ControlNumberKind{
				edidocument.ControlNumberKindInterchange,
				edidocument.ControlNumberKindGroup,
				edidocument.ControlNumberKindTransaction,
			},
		},
	)
	if err != nil {
		return acknowledgmentControlNumbers{}, err
	}
	return acknowledgmentControlNumbers{
		interchange: fmt.Sprintf("%09d", allocated[edidocument.ControlNumberKindInterchange]),
		group:       strconv.FormatInt(allocated[edidocument.ControlNumberKindGroup], 10),
		transaction: fmt.Sprintf("%04d", allocated[edidocument.ControlNumberKindTransaction]),
	}, nil
}

func (s *Service) createAcknowledgmentMessage(
	ctx context.Context,
	req *GenerateAcknowledgmentRequest,
	message *edimessage.EDIMessage,
) (created *edimessage.EDIMessage, replayed bool, err error) {
	created, err = s.messageRepo.CreateMessageWithDiagnostics(
		ctx,
		edimessageports.CreateEDIMessageWithDiagnosticsRequest{Message: message},
	)
	if err == nil {
		return created, false, nil
	}
	if req.IdempotencyKey == "" || !dberror.IsUniqueConstraintViolation(err) {
		return nil, false, err
	}
	existing, found, lookupErr := s.generatedDocument(ctx, &GenerateEDIDocumentRequest{
		TenantInfo:           req.TenantInfo,
		TransactionSet:       message.TransactionSet,
		IdempotencyKey:       req.IdempotencyKey,
		DisableDeliveryQueue: true,
	})
	if lookupErr != nil {
		return nil, false, lookupErr
	}
	if !found {
		return nil, false, err
	}
	return existing, true, nil
}

func (s *Service) previewMessageAcknowledgment(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*EDIDocumentPreview, error) {
	source, err := s.messageAcknowledgmentSource(ctx, req)
	if err != nil {
		return nil, err
	}
	written, err := writeAcknowledgment(
		ctx,
		&source.envelope,
		source.kind,
		source.group,
		provisionalAcknowledgmentNumbers,
	)
	if err != nil {
		return nil, err
	}
	return &EDIDocumentPreview{
		RawX12:                   written.Text,
		SegmentCount:             int64(written.SegmentCount),
		X12Version:               acknowledgmentVersion(source.kind),
		InterchangeControlNumber: provisionalAcknowledgmentNumbers.interchange,
		GroupControlNumber:       provisionalAcknowledgmentNumbers.group,
		TransactionControlNumber: provisionalAcknowledgmentNumbers.transaction,
		LadingVersionID:          source.program.Version.ID,
	}, nil
}

func (s *Service) generateMessageAcknowledgment(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*edimessage.EDIMessage, error) {
	source, err := s.messageAcknowledgmentSource(ctx, &PreviewEDIDocumentRequest{
		TenantInfo:      req.TenantInfo,
		EDIPartnerID:    req.EDIPartnerID,
		SourceMessageID: req.SourceMessageID,
		TransactionSet:  req.TransactionSet,
		Direction:       req.Direction,
	})
	if err != nil {
		return nil, err
	}
	ackType := edidocument.AcknowledgmentType997
	if source.kind == x12.AcknowledgmentKind999 {
		ackType = edidocument.AcknowledgmentType999
	}
	return s.GenerateAcknowledgment(ctx, &GenerateAcknowledgmentRequest{
		TenantInfo:           req.TenantInfo,
		EDIPartnerID:         source.partnerID,
		Type:                 ackType,
		Group:                source.group,
		SourceMessageID:      req.SourceMessageID,
		GeneratedByID:        req.GeneratedByID,
		IdempotencyKey:       req.IdempotencyKey,
		DisableDeliveryQueue: req.DisableDeliveryQueue,
	})
}

type acknowledgmentSource struct {
	partnerID pulid.ID
	program   *edilading.Program
	envelope  edidocument.X12EnvelopeSettings
	kind      x12.AcknowledgmentKind
	group     *x12.Group
}

// messageAcknowledgmentSource reads a stored inbound message again against its guide
// and makes the group an acknowledgment of it reports on: the message's own
// transaction set inside the functional group it arrived in.
func (s *Service) messageAcknowledgmentSource(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*acknowledgmentSource, error) {
	if !req.TransactionSet.IsAcknowledgment() {
		return nil, sourceTransactionSetError(
			"sourceMessageId",
			"source message",
			req.TransactionSet,
			edidocument.TransactionSet997,
			edidocument.TransactionSet999,
		)
	}
	_, kind, err := acknowledgmentSet(edidocument.AcknowledgmentType(req.TransactionSet))
	if err != nil {
		return nil, err
	}
	message, err := s.messageRepo.GetMessageByID(ctx, edimessageports.GetEDIMessageByIDRequest{
		ID:         req.SourceMessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if message.Direction != edidocument.DocumentDirectionInbound {
		return nil, errortypes.NewValidationError(
			"sourceMessageId",
			errortypes.ErrInvalid,
			"Only a message received from a partner can be acknowledged",
		)
	}
	partnerID := req.EDIPartnerID
	if partnerID.IsNil() {
		partnerID = message.EDIPartnerID
	}
	envelope, program, err := s.partnerEnvelope(ctx, req.TenantInfo, partnerID)
	if err != nil {
		return nil, err
	}

	reading, err := x12.Default().Read(ctx, &x12.ReadRequest{
		Text:     message.RawX12,
		Fragment: edix12inspect.IsFragment(message.RawX12),
	})
	if err != nil {
		return nil, err
	}
	transactions := edix12inspect.AllTransactions(reading)
	if len(transactions) == 0 {
		return nil, errortypes.NewValidationError(
			"sourceMessageId",
			errortypes.ErrInvalid,
			"The source message holds no transaction set to acknowledge",
		)
	}
	transaction := *transactions[0]
	return &acknowledgmentSource{
		partnerID: partnerID,
		program:   program,
		envelope:  envelope,
		kind:      kind,
		group: &x12.Group{
			Header: &x12.GSHeader{
				FunctionalID:  edidocument.FunctionalGroupDefault(message.TransactionSet),
				ControlNumber: message.GroupControlNumber,
				Version:       message.X12Version,
			},
			Transactions:   []x12.Transaction{transaction},
			Acknowledgment: singleTransactionAcknowledgment(transaction.Status),
		},
	}, nil
}

func singleTransactionAcknowledgment(status x12.TransactionStatus) x12.GroupAcknowledgment {
	acknowledgment := x12.GroupAcknowledgment{Code: "A", Included: 1, Received: 1, Accepted: 1}
	switch status {
	case x12.TransactionRejected:
		acknowledgment.Code = "R"
		acknowledgment.Accepted = 0
	case x12.TransactionAcceptedWithErrors:
		acknowledgment.Code = "E"
	case x12.TransactionAccepted:
	}
	return acknowledgment
}

func writeAcknowledgment(
	ctx context.Context,
	envelope *edidocument.X12EnvelopeSettings,
	kind x12.AcknowledgmentKind,
	group *x12.Group,
	numbers acknowledgmentControlNumbers,
) (*x12.Acknowledgment, error) {
	defaults := edidocument.DefaultX12EnvelopeSettings()
	now := time.Now().UTC()
	acknowledgments, err := x12.Default().Acknowledge(ctx, &x12.AcknowledgeRequest{
		Kind: kind,
		Separators: &x12.Separators{
			Element:    envelope.ElementSeparator,
			Component:  envelope.ComponentSeparator,
			Repetition: envelope.RepetitionSeparator,
			Segment:    envelope.SegmentTerminator,
		},
		Acknowledgments: []x12.AcknowledgeInput{{
			Group: group,
			Envelope: x12.AcknowledgmentEnvelope{
				SenderQualifier: stringutils.FirstNonEmpty(
					envelope.InterchangeSenderQualifier,
					defaults.InterchangeSenderQualifier,
				),
				SenderID: strings.TrimSpace(envelope.InterchangeSenderID),
				ReceiverQualifier: stringutils.FirstNonEmpty(
					envelope.InterchangeReceiverQualifier,
					defaults.InterchangeReceiverQualifier,
				),
				ReceiverID: strings.TrimSpace(envelope.InterchangeReceiverID),
				ApplicationSender: stringutils.FirstNonEmpty(
					envelope.ApplicationSenderCode,
					envelope.InterchangeSenderID,
				),
				ApplicationReceiver: stringutils.FirstNonEmpty(
					envelope.ApplicationReceiverCode,
					envelope.InterchangeReceiverID,
				),
				Date:                     now.Format("20060102"),
				Time:                     now.Format("1504"),
				InterchangeControlNumber: numbers.interchange,
				GroupControlNumber:       numbers.group,
				TransactionControlNumber: numbers.transaction,
				Usage: stringutils.FirstNonEmpty(
					envelope.InterchangeUsageIndicator,
					defaults.InterchangeUsageIndicator,
				),
			},
		}},
	})
	if err != nil {
		return nil, err
	}
	return &acknowledgments[0], nil
}

func acknowledgmentVersion(kind x12.AcknowledgmentKind) string {
	if kind == x12.AcknowledgmentKind999 {
		return "005010X231A1"
	}
	return edidocument.DefaultX12204Version
}

func acknowledgmentPayload(
	ackSet edidocument.TransactionSet,
	sourceMessageID pulid.ID,
	group *x12.Group,
) edipayload.DocumentPayload {
	ack := edipayload.FunctionalAcknowledgmentPayload{
		SourceMessageID:             sourceMessageID,
		GroupAcknowledgmentCode:     group.Acknowledgment.Code,
		AcceptedTransactionSetCount: int64(group.Acknowledgment.Accepted),
		ReceivedTransactionSetCount: int64(group.Acknowledgment.Received),
		IncludedTransactionSetCount: int64(group.Acknowledgment.Included),
	}
	if group.Header != nil {
		ack.OriginalFunctionalGroupID = group.Header.FunctionalID
		ack.OriginalGroupControlNumber = group.Header.ControlNumber
	}
	if len(group.Transactions) == 1 {
		transaction := &group.Transactions[0]
		ack.OriginalTransactionSet = edidocument.TransactionSet(transaction.TransactionSet)
		ack.OriginalTransactionControlNumber = transaction.ControlNumber
		ack.TransactionAcknowledgmentCode = transactionAcknowledgmentCode(transaction.Status)
	}
	for index := range group.Transactions {
		ack.Diagnostics = append(
			ack.Diagnostics,
			acknowledgmentDiagnostics(&group.Transactions[index])...,
		)
	}
	if ackSet == edidocument.TransactionSet999 {
		implementation := edipayload.ImplementationAckPayload(ack)
		return edipayload.DocumentPayload{
			TransactionSet:               edidocument.TransactionSet999,
			ImplementationAcknowledgment: &implementation,
		}
	}
	return edipayload.DocumentPayload{
		TransactionSet:           edidocument.TransactionSet997,
		FunctionalAcknowledgment: &ack,
	}
}

func acknowledgmentDiagnostics(transaction *x12.Transaction) []edipayload.AcknowledgmentDiagnostic {
	diagnostics := make([]edipayload.AcknowledgmentDiagnostic, 0, len(transaction.Findings))
	for index := range transaction.Findings {
		finding := &transaction.Findings[index]
		if finding.Severity == x12.SeverityInfo || finding.Code == "" {
			continue
		}
		diagnostic := edipayload.AcknowledgmentDiagnostic{
			SegmentID: finding.SegmentID,
			ErrorCode: finding.Code,
			Message:   finding.Message.String(),
		}
		if finding.Position != nil {
			diagnostic.SegmentPosition = int64(*finding.Position)
		}
		if finding.Element != nil {
			diagnostic.ElementPosition = int64(*finding.Element)
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics
}

func transactionAcknowledgmentCode(status x12.TransactionStatus) string {
	switch status {
	case x12.TransactionRejected:
		return "R"
	case x12.TransactionAcceptedWithErrors:
		return "E"
	case x12.TransactionAccepted:
	}
	return "A"
}
