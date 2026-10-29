package editransferservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edidecision"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
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
)

const fieldTransferID = "transferId"

type TransferApprovalPlan struct {
	Transfer      *editransfer.EDITransfer
	After         *editransfer.EDITransfer
	Mapping       *editendermapping.MappingPreview
	Shipment      *shipment.Shipment
	SendsResponse bool
	Refusal       error
}

func (s *Service) PlanApproveTransfer(
	ctx context.Context,
	req *ApproveTransferRequest,
	actor *permissionports.RequestActor,
) (*TransferApprovalPlan, error) {
	transfer, err := s.transferRepo.GetTransferByID(ctx, editenderports.GetEDITransferByIDRequest{
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
		Refusal: edidecision.RequireActionableTransfer(
			transfer,
			edidecision.TransferVerbApproved,
		),
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
	targetPartner, err := s.partnerRepo.GetByID(ctx, edipartnerports.GetEDIPartnerByIDRequest{
		ID:         transfer.TargetPartnerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	plan.Mapping, err = s.tenderMapping.BuildMappingPreview(
		ctx,
		targetPartner,
		&transfer.TenderPayload,
	)
	if err != nil {
		return err
	}
	if len(plan.Mapping.Unresolved) > 0 {
		plan.Refusal = editendermapping.UnresolvedMappingsError(plan.Mapping.Unresolved)
		return nil
	}

	approverID := pulid.Nil
	if actor != nil {
		approverID = actor.UserID
	}
	after := *transfer
	edidecision.MarkTransferApprovalStarted(&after, &edidecision.TransferApprovalMark{
		BusinessUnitID: req.TenantInfo.BuID,
		Mapping:        plan.Mapping.All,
		ApproverID:     approverID,
		At:             timeutils.NowUnix(),
	})
	plan.After = &after
	plan.Shipment, plan.Refusal = s.tenderMapping.BuildTargetShipment(
		&after,
		req.TenantInfo.BuID,
		plan.Mapping.All,
		approverID,
	)

	return nil
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
	edioutbound.OutboundDocumentPreview

	Transfer *editransfer.EDITransfer
	Choice   TenderResponseChoice
}

func (s *Service) PreviewTenderResponse(
	ctx context.Context,
	req *PreviewTenderResponseRequest,
) (*TenderResponsePreview, error) {
	transfer, err := s.transferRepo.GetTransferByID(ctx, editenderports.GetEDITransferByIDRequest{
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

	preview, err := s.outbound.PreviewOutboundDocument(
		ctx,
		tenderResponseRequest(req.TenantInfo, answered, req.TenantInfo.UserID),
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
	transfer *editransfer.EDITransfer,
	actorID pulid.ID,
) *edioutbound.GenerateEDIDocumentRequest {
	payload := edipayloadbuild.BuildTenderResponsePayload(transfer)

	return &edioutbound.GenerateEDIDocumentRequest{
		TenantInfo:     tenantInfo,
		EDIPartnerID:   transfer.TargetPartnerID,
		TransferID:     transfer.ID,
		TransactionSet: edidocument.TransactionSet990,
		Direction:      edidocument.DocumentDirectionOutbound,
		Payload:        &payload,
		GeneratedByID:  actorID,
	}
}

func tenderResponseToPreview(
	transfer *editransfer.EDITransfer,
	partnerID pulid.ID,
	choice TenderResponseChoice,
	reason string,
) (*editransfer.EDITransfer, error) {
	if transfer.InboundMessageID.IsNil() {
		return nil, errortypes.NewValidationError(
			fieldTransferID,
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
	case transfer.Status == editransfer.TransferStatusApproved || transfer.Status == editransfer.TransferStatusRejected:
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
			answered.Status = editransfer.TransferStatusApproved
		case TenderResponseDecline:
			answered.Status = editransfer.TransferStatusRejected
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
			fieldTransferID,
			errortypes.ErrInvalidOperation,
			"This tender is {0} and will not be answered with a 990",
			string(transfer.Status),
		)
	}

	return &answered, nil
}

func tenderChoiceOf(transfer *editransfer.EDITransfer) TenderResponseChoice {
	if transfer.Status == editransfer.TransferStatusRejected {
		return TenderResponseDecline
	}

	return TenderResponseAccept
}
