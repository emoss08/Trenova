package edidecision

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	TransferVerbApproved = "approved"
	TransferVerbRejected = "rejected"
	TransferVerbCanceled = "canceled"
	TransferVerbExpired  = "expired"
)

type TransferApprovalMark struct {
	BusinessUnitID pulid.ID
	Mapping        []editransfer.MappingResolution
	ApproverID     pulid.ID
	At             int64
}

func MarkTransferApprovalStarted(transfer *editransfer.EDITransfer, mark *TransferApprovalMark) {
	at := mark.At
	transfer.Status = editransfer.TransferStatusProcessing
	transfer.TargetBusinessUnitID = mark.BusinessUnitID
	transfer.MappingSnapshot = mark.Mapping
	transfer.ApprovedByID = mark.ApproverID
	transfer.ApprovedAt = &at
	transfer.ProcessingStartedAt = &at
	transfer.ApprovalWorkflowID = buildLoadTenderApprovalWorkflowID(transfer.ID)
	transfer.ApprovalWorkflowRunID = ""
	transfer.FailureReason = ""
}

func MarkTransferRejected(
	transfer *editransfer.EDITransfer,
	reason string,
	actorID pulid.ID,
	at int64,
) {
	transfer.Status = editransfer.TransferStatusRejected
	transfer.RejectionReason = reason
	transfer.RejectedByID = actorID
	transfer.RejectedAt = &at
}

func MarkTransferCanceled(transfer *editransfer.EDITransfer, actorID pulid.ID, at int64) {
	transfer.Status = editransfer.TransferStatusCanceled
	transfer.CanceledByID = actorID
	transfer.CanceledAt = &at
}

func MarkTransferExpired(transfer *editransfer.EDITransfer, at int64) {
	transfer.Status = editransfer.TransferStatusExpired
	transfer.ProcessedAt = &at
}

type ChangeReviewMark struct {
	Applied    bool
	ReviewerID pulid.ID
	Reason     string
	At         int64
}

func MarkTenderChangeReviewed(change *editenderchange.TenderChange, mark *ChangeReviewMark) {
	at := mark.At
	change.Status = editenderchange.TenderChangeStatusRejected
	if mark.Applied {
		change.Status = editenderchange.TenderChangeStatusApplied
		change.AppliedByID = mark.ReviewerID
		change.AppliedAt = &at
	}
	change.ReviewedByID = mark.ReviewerID
	change.ReviewedAt = &at
	if reason := strings.TrimSpace(mark.Reason); reason != "" {
		change.FailureReason = reason
	}
}

func MarkTransferChangeReviewed(change *edisync.TransferChange, mark *ChangeReviewMark) {
	at := mark.At
	change.Status = edisync.TransferChangeStatusRejected
	if mark.Applied {
		change.Status = edisync.TransferChangeStatusApplied
		change.AppliedByID = mark.ReviewerID
		change.AppliedAt = &at
	}
	change.ReviewedByID = mark.ReviewerID
	change.ReviewedAt = &at
	if reason := strings.TrimSpace(mark.Reason); reason != "" {
		change.FailureReason = reason
	}
}

func RequireActionableTransfer(transfer *editransfer.EDITransfer, verb string) error {
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

func CheckTenderChangeReview(
	change *editenderchange.TenderChange,
	tenantInfo pagination.TenantInfo,
) error {
	if change == nil || change.Status != editenderchange.TenderChangeStatusPendingReview {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI tender change is not pending review",
		)
	}

	return validateTenderChangeReviewer(change, tenantInfo)
}
