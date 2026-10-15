package ediservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	TransferDirectionInbound  = "inbound"
	TransferDirectionOutbound = "outbound"
	TransferDirectionAny      = ""
	TransferVerbApproved      = "approved"
	TransferVerbRejected      = "rejected"
	TransferVerbCanceled      = "canceled"
	TransferVerbExpired       = "expired"
)

type TransferApprovalPlan struct {
	Transfer      *edi.EDITransfer
	After         *edi.EDITransfer
	Mapping       *MappingPreview
	Shipment      *shipment.Shipment
	SendsResponse bool
	Refusal       error
}

type DeliveryPlan struct {
	Message *edi.EDIMessage
	Profile *edi.EDICommunicationProfile
}

type LoadTenderPlan struct {
	SourceShipment *shipment.Shipment
	SourcePartner  *edi.EDIPartner
	TargetPartner  *edi.EDIPartner
	Payload        edi.LoadTenderPayload
	Mapping        *MappingPreview
	Status         edi.TransferStatus
}

type TransferApprovalMark struct {
	BusinessUnitID pulid.ID
	Mapping        []edi.MappingResolution
	ApproverID     pulid.ID
	At             int64
}

func MarkTransferApprovalStarted(transfer *edi.EDITransfer, mark *TransferApprovalMark) {
	at := mark.At
	transfer.Status = edi.TransferStatusProcessing
	transfer.TargetBusinessUnitID = mark.BusinessUnitID
	transfer.MappingSnapshot = mark.Mapping
	transfer.ApprovedByID = mark.ApproverID
	transfer.ApprovedAt = &at
	transfer.ProcessingStartedAt = &at
	transfer.ApprovalWorkflowID = buildLoadTenderApprovalWorkflowID(transfer.ID)
	transfer.ApprovalWorkflowRunID = ""
	transfer.FailureReason = ""
}

func MarkTransferRejected(transfer *edi.EDITransfer, reason string, actorID pulid.ID, at int64) {
	transfer.Status = edi.TransferStatusRejected
	transfer.RejectionReason = reason
	transfer.RejectedByID = actorID
	transfer.RejectedAt = &at
}

func MarkTransferCanceled(transfer *edi.EDITransfer, actorID pulid.ID, at int64) {
	transfer.Status = edi.TransferStatusCanceled
	transfer.CanceledByID = actorID
	transfer.CanceledAt = &at
}

func MarkTransferExpired(transfer *edi.EDITransfer, at int64) {
	transfer.Status = edi.TransferStatusExpired
	transfer.ProcessedAt = &at
}

type ChangeReviewMark struct {
	Applied    bool
	ReviewerID pulid.ID
	Reason     string
	At         int64
}

func MarkTenderChangeReviewed(change *edi.TenderChange, mark *ChangeReviewMark) {
	at := mark.At
	change.Status = edi.TenderChangeStatusRejected
	if mark.Applied {
		change.Status = edi.TenderChangeStatusApplied
		change.AppliedByID = mark.ReviewerID
		change.AppliedAt = &at
	}
	change.ReviewedByID = mark.ReviewerID
	change.ReviewedAt = &at
	if reason := strings.TrimSpace(mark.Reason); reason != "" {
		change.FailureReason = reason
	}
}

func MarkTransferChangeReviewed(change *edi.TransferChange, mark *ChangeReviewMark) {
	at := mark.At
	change.Status = edi.TransferChangeStatusRejected
	if mark.Applied {
		change.Status = edi.TransferChangeStatusApplied
		change.AppliedByID = mark.ReviewerID
		change.AppliedAt = &at
	}
	change.ReviewedByID = mark.ReviewerID
	change.ReviewedAt = &at
	if reason := strings.TrimSpace(mark.Reason); reason != "" {
		change.FailureReason = reason
	}
}

func TenderResponseFor(transfer *edi.EDITransfer) *edi.TenderResponsePayload {
	return buildTenderResponsePayload(transfer).TenderResponse
}

func RequireActionableTransfer(transfer *edi.EDITransfer, verb string) error {
	if transfer != nil && transfer.Status.IsActionable() {
		return nil
	}

	switch verb {
	case TransferVerbApproved:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer cannot be approved while finalized or processing",
		)
	case TransferVerbRejected:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer cannot be rejected while finalized or processing",
		)
	case TransferVerbCanceled:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer cannot be canceled while finalized or processing",
		)
	default:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer cannot be expired while finalized or processing",
		)
	}
}

func TransferRejectionReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if reason == "" {
		return "", errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Rejection reason is required",
		)
	}

	return reason, nil
}

func CheckTenderChangeReview(change *edi.TenderChange, tenantInfo pagination.TenantInfo) error {
	if change == nil || change.Status != edi.TenderChangeStatusPendingReview {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI tender change is not pending review",
		)
	}

	return validateTenderChangeReviewer(change, tenantInfo)
}

func (s *Service) CheckTransferChangeReview(
	ctx context.Context,
	change *edi.TransferChange,
	tenantInfo pagination.TenantInfo,
) error {
	if change == nil || change.Status != edi.TransferChangeStatusPendingReview {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer change has already been reviewed",
		)
	}

	return s.validateTransferChangeReviewer(ctx, change, tenantInfo)
}

func (s *Service) PlanApproveTransfer(
	ctx context.Context,
	req *ApproveTransferRequest,
	actor *permissionports.RequestActor,
) (*TransferApprovalPlan, error) {
	transfer, err := s.transferRepo.GetTransferByID(ctx, ediports.GetEDITransferByIDRequest{
		ID:         req.TransferID,
		TenantInfo: req.TenantInfo,
		Direction:  TransferDirectionInbound,
	})
	if err != nil {
		return nil, err
	}

	plan := &TransferApprovalPlan{
		Transfer:      transfer,
		SendsResponse: transfer.InboundMessageID.IsNotNil(),
		Refusal:       RequireActionableTransfer(transfer, TransferVerbApproved),
	}
	if plan.Refusal == nil {
		if err = s.planApproval(ctx, req, actor, plan); err != nil {
			return nil, err
		}
	}

	return plan, nil
}

func (s *Service) planApproval(
	ctx context.Context,
	req *ApproveTransferRequest,
	actor *permissionports.RequestActor,
	plan *TransferApprovalPlan,
) error {
	transfer := plan.Transfer
	targetPartner, err := s.partnerRepo.GetByID(ctx, ediports.GetEDIPartnerByIDRequest{
		ID:         transfer.TargetPartnerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	plan.Mapping, err = s.buildMappingPreview(ctx, targetPartner, transfer.TenderPayload)
	if err != nil {
		return err
	}
	if len(plan.Mapping.Unresolved) > 0 {
		plan.Refusal = unresolvedMappingsError(plan.Mapping.Unresolved)
		return nil
	}

	approverID := pulid.Nil
	if actor != nil {
		approverID = actor.UserID
	}
	after := *transfer
	MarkTransferApprovalStarted(&after, &TransferApprovalMark{
		BusinessUnitID: req.TenantInfo.BuID,
		Mapping:        plan.Mapping.All,
		ApproverID:     approverID,
		At:             timeutils.NowUnix(),
	})
	plan.After = &after
	plan.Shipment, plan.Refusal = s.buildTargetShipment(
		&after,
		req.TenantInfo.BuID,
		plan.Mapping.All,
		approverID,
	)

	return nil
}

func (s *Service) PlanRetryMessageDelivery(
	ctx context.Context,
	req *RetryMessageDeliveryRequest,
) (*DeliveryPlan, error) {
	message, err := s.deliverableMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	if message.Direction != edi.DocumentDirectionOutbound {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrInvalidOperation,
			"Only outbound EDI messages can be delivered",
		)
	}
	if message.DeliveryStatus != edi.MessageDeliveryStatusQueued &&
		!message.DeliveryStatus.IsRetryable() {
		return nil, errortypes.NewValidationError(
			"deliveryStatus",
			errortypes.ErrInvalidOperation,
			"Only queued, failed, or dead-lettered EDI messages can be retried",
		)
	}
	profile, err := s.deliveryProfileForMessage(ctx, message)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"messageId",
				errortypes.ErrInvalidOperation,
				"EDI partner has no active SFTP or VAN communication profile for delivery",
			)
		}
		return nil, err
	}

	return &DeliveryPlan{Message: message, Profile: profile}, nil
}

func (s *Service) PlanReplayMessageDelivery(
	ctx context.Context,
	req *RetryMessageDeliveryRequest,
) (*DeliveryPlan, error) {
	message, err := s.deliverableMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	if message.Direction != edi.DocumentDirectionOutbound {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrInvalidOperation,
			"Only outbound EDI messages can be replayed",
		)
	}
	if message.DeliveryStatus != edi.MessageDeliveryStatusSent {
		return nil, errortypes.NewValidationError(
			"deliveryStatus",
			errortypes.ErrInvalidOperation,
			"Only delivered EDI messages can be replayed; use retry for failed messages",
		)
	}
	if message.RawPurgedAt != nil {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrInvalidOperation,
			"The raw X12 payload was purged by the retention policy and can no longer be replayed",
		)
	}
	profile, err := s.deliveryProfileForMessage(ctx, message)
	if err != nil {
		return nil, err
	}

	return &DeliveryPlan{Message: message, Profile: profile}, nil
}

func (s *Service) deliverableMessage(
	ctx context.Context,
	req *RetryMessageDeliveryRequest,
) (*edi.EDIMessage, error) {
	if req == nil || req.MessageID.IsNil() {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrRequired,
			"EDI message ID is required",
		)
	}

	return s.messageRepo.GetMessageByID(ctx, ediports.GetEDIMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
}

type OutboundDocumentPreview struct {
	Document *EDIDocumentPreview
	Blocking bool
}

func (s *Service) PreviewOutboundDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*OutboundDocumentPreview, error) {
	resolved, err := s.resolveGenerateContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return resolved.outboundPreview()
}

type TenderResponseChoice string

const (
	TenderResponseAccept  = TenderResponseChoice("Accept")
	TenderResponseDecline = TenderResponseChoice("Decline")
)

type PreviewTenderResponseRequest struct {
	TenantInfo pagination.TenantInfo
	PartnerID  pulid.ID
	TransferID pulid.ID
	Choice     TenderResponseChoice
	Reason     string
}

type TenderResponsePreview struct {
	OutboundDocumentPreview

	Transfer *edi.EDITransfer
	Choice   TenderResponseChoice
}

func (s *Service) PreviewTenderResponse(
	ctx context.Context,
	req *PreviewTenderResponseRequest,
) (*TenderResponsePreview, error) {
	transfer, err := s.transferRepo.GetTransferByID(ctx, ediports.GetEDITransferByIDRequest{
		ID:         req.TransferID,
		TenantInfo: req.TenantInfo,
		Direction:  TransferDirectionInbound,
	})
	if err != nil {
		return nil, err
	}
	answered, err := tenderResponseToPreview(transfer, req.PartnerID, req.Choice, req.Reason)
	if err != nil {
		return nil, err
	}

	profile, err := s.documentProfileRepo.GetActivePartnerDocumentProfile(
		ctx,
		ediports.GetActiveEDIPartnerDocumentProfileRequest{
			PartnerID:      transfer.TargetPartnerID,
			TenantInfo:     req.TenantInfo,
			TransactionSet: edi.TransactionSet990,
			Direction:      edi.DocumentDirectionOutbound,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"partnerId",
				errortypes.ErrInvalidOperation,
				"This partner has no active outbound 990 profile yet. Trenova creates one from the partner's envelope and the base 990 template when the tender is decided, so its 990 cannot be previewed before then",
			)
		}
		return nil, err
	}

	preview, err := s.PreviewOutboundDocument(
		ctx,
		tenderResponseRequest(req.TenantInfo, answered, profile.ID, req.TenantInfo.UserID),
	)
	if err != nil {
		return nil, err
	}

	return &TenderResponsePreview{
		OutboundDocumentPreview: *preview,
		Transfer:                transfer,
		Choice:                  tenderChoiceOf(answered),
	}, nil
}

func tenderResponseRequest(
	tenantInfo pagination.TenantInfo,
	transfer *edi.EDITransfer,
	profileID pulid.ID,
	actorID pulid.ID,
) *GenerateEDIDocumentRequest {
	payload := buildTenderResponsePayload(transfer)

	return &GenerateEDIDocumentRequest{
		TenantInfo:               tenantInfo,
		PartnerDocumentProfileID: profileID,
		EDIPartnerID:             transfer.TargetPartnerID,
		TransferID:               transfer.ID,
		TransactionSet:           edi.TransactionSet990,
		Direction:                edi.DocumentDirectionOutbound,
		Payload:                  &payload,
		GeneratedByID:            actorID,
	}
}

func tenderResponseToPreview(
	transfer *edi.EDITransfer,
	partnerID pulid.ID,
	choice TenderResponseChoice,
	reason string,
) (*edi.EDITransfer, error) {
	if transfer.InboundMessageID.IsNil() {
		return nil, errortypes.NewValidationError(
			"transferId",
			errortypes.ErrInvalidOperation,
			"This tender came from another Trenova organization and is answered inside Trenova, so no 990 is sent for it",
		)
	}
	if partnerID.IsNotNil() && partnerID != transfer.TargetPartnerID {
		return nil, errortypes.NewValidationError(
			"partnerId",
			errortypes.ErrInvalidReference,
			"This tender came from another trading partner; its 990 goes to the partner that sent it",
		)
	}

	answered := *transfer
	switch {
	case transfer.Status == edi.TransferStatusApproved || transfer.Status == edi.TransferStatusRejected:
		decided := tenderChoiceOf(transfer)
		if choice != "" && choice != decided {
			return nil, errortypes.NewValidationError(
				"response",
				errortypes.ErrInvalidOperation,
				"This tender was already answered {0}; its 990 says so",
				string(decided),
			)
		}
	case transfer.Status.IsActionable():
		switch choice {
		case TenderResponseAccept:
			answered.Status = edi.TransferStatusApproved
		case TenderResponseDecline:
			answered.Status = edi.TransferStatusRejected
			answered.RejectionReason = strings.TrimSpace(reason)
		default:
			return nil, errortypes.NewValidationError(
				"response",
				errortypes.ErrRequired,
				"This tender is not answered yet; say whether to preview the 990 that accepts it or the one that declines it",
			)
		}
	default:
		return nil, errortypes.NewValidationError(
			"transferId",
			errortypes.ErrInvalidOperation,
			"This tender is {0} and will not be answered with a 990",
			string(transfer.Status),
		)
	}

	return &answered, nil
}

func tenderChoiceOf(transfer *edi.EDITransfer) TenderResponseChoice {
	if transfer.Status == edi.TransferStatusRejected {
		return TenderResponseDecline
	}

	return TenderResponseAccept
}

func (s *Service) PlanGenerateDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*EDIDocumentPreview, error) {
	resolved, err := s.resolveGenerateContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return resolved.provisionalRender()
}

type TransferChangeEffect struct {
	Shipment   *shipment.Shipment
	NextStatus shipment.Status
}

func (s *Service) PlanTransferChangeEffect(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	change *edi.TransferChange,
) (*TransferChangeEffect, error) {
	switch change.ChangeType {
	case edi.TransferChangeTypeShipmentStatus214, edi.TransferChangeTypeShipmentCancel214:
	default:
		return &TransferChangeEffect{}, nil
	}

	applyCtx, err := s.buildApprovedTransferChangeContext(ctx, tenantInfo, change)
	if err != nil {
		return nil, err
	}

	return &TransferChangeEffect{Shipment: applyCtx.opposite, NextStatus: applyCtx.nextStatus}, nil
}
