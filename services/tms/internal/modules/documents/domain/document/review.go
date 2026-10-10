package document

import (
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxRejectionReasonLength = 1000

type ReviewDecision string

const (
	ReviewDecisionApprove ReviewDecision = "approve"
	ReviewDecisionReject  ReviewDecision = "reject"
)

func (d ReviewDecision) IsValid() bool {
	return d == ReviewDecisionApprove || d == ReviewDecisionReject
}

func (d *Document) CanBeReviewed() error {
	if !d.IsCurrentVersion {
		return errortypes.NewBusinessError(
			"Only the current version of a document can be reviewed. Review the latest version instead.",
		)
	}

	if d.Status == StatusArchived {
		return errortypes.NewBusinessError("An archived document cannot be reviewed.")
	}

	return nil
}

func (d *Document) Approve(reviewerID pulid.ID, at int64) error {
	if err := d.CanBeReviewed(); err != nil {
		return err
	}

	if d.Status == StatusActive && d.ApprovedAt != nil {
		return errortypes.NewBusinessError("This document is already approved.")
	}

	if d.ExpirationDate != nil && *d.ExpirationDate <= at {
		return errortypes.NewBusinessError(
			"This document is past its expiration date. Upload a current version instead.",
		)
	}

	d.Status = StatusActive
	d.ApprovedByID = reviewerID
	d.ApprovedAt = &at
	d.clearRejection()

	return nil
}

func (d *Document) Reject(reviewerID pulid.ID, at int64, reason string) error {
	if err := d.CanBeReviewed(); err != nil {
		return err
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("reason", errortypes.ErrRequired, "Say why the document is rejected")
		return multiErr
	}

	if len([]rune(reason)) > MaxRejectionReasonLength {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"reason",
			errortypes.ErrInvalidLength,
			"Rejection reason cannot be longer than 1000 characters",
		)
		return multiErr
	}

	if d.Status == StatusRejected {
		return errortypes.NewBusinessError("This document is already rejected.")
	}

	d.Status = StatusRejected
	d.RejectedByID = reviewerID
	d.RejectedAt = &at
	d.RejectionReason = reason
	d.ApprovedByID = pulid.Nil
	d.ApprovedAt = nil

	return nil
}

func (d *Document) clearRejection() {
	d.RejectedByID = pulid.Nil
	d.RejectedAt = nil
	d.RejectionReason = ""
}
