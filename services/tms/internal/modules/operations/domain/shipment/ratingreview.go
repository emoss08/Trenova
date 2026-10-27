package shipment

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type RatingReviewApproval struct {
	Reviews      []string `json:"reviews"`
	ApprovedByID pulid.ID `json:"approvedById"`
	ApprovedAt   int64    `json:"approvedAt"`
}

func (a *RatingReviewApproval) Covers(reasons []string) bool {
	if a == nil || len(a.Reviews) != len(reasons) {
		return false
	}
	approved := slices.Clone(a.Reviews)
	pending := slices.Clone(reasons)
	slices.Sort(approved)
	slices.Sort(pending)
	return slices.Equal(approved, pending)
}

func (d *RatingDetail) ReviewReasons() []string {
	if d == nil || d.Bill == nil || len(d.Bill.Reviews) == 0 {
		return nil
	}
	out := make([]string, 0, len(d.Bill.Reviews))
	for i := range d.Bill.Reviews {
		if text := strings.TrimSpace(d.Bill.Reviews[i].Text); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func (d *RatingDetail) AwaitsReview() bool {
	reasons := d.ReviewReasons()
	return len(reasons) > 0 && !d.ReviewApproval.Covers(reasons)
}

func (d *RatingDetail) CarryReviewApproval(previous *RatingDetail) {
	if d == nil {
		return
	}
	d.ReviewApproval = nil
	if previous == nil || previous.ReviewApproval == nil {
		return
	}
	reasons := d.ReviewReasons()
	if len(reasons) == 0 || !previous.ReviewApproval.Covers(reasons) {
		return
	}
	approval := *previous.ReviewApproval
	approval.Reviews = slices.Clone(previous.ReviewApproval.Reviews)
	d.ReviewApproval = &approval
}

func (d *RatingDetail) ApproveReview(userID pulid.ID, at int64) (bool, error) {
	reasons := d.ReviewReasons()
	if len(reasons) == 0 {
		return false, errortypes.NewBusinessError(
			"This shipment's rating has nothing waiting for review",
		)
	}
	if d.ReviewApproval.Covers(reasons) {
		return false, nil
	}
	d.ReviewApproval = &RatingReviewApproval{
		Reviews:      reasons,
		ApprovedByID: userID,
		ApprovedAt:   at,
	}
	return true, nil
}

func (s *Shipment) AwaitsRatingReview() bool {
	return s != nil && s.RatingDetail.AwaitsReview()
}

func RatingReviewHoldError(shipments []*Shipment) error {
	held := make([]string, 0, len(shipments))
	numbers := make([]string, 0, len(shipments))
	for _, entity := range shipments {
		if !entity.AwaitsRatingReview() {
			continue
		}
		held = append(held, entity.ID.String())
		numbers = append(numbers, entity.ProNumber)
	}
	if len(held) == 0 {
		return nil
	}

	var err *errortypes.BusinessError
	if len(held) == 1 {
		err = errortypes.NewBusinessError(
			"The tariff held the rating of shipment {0} for review. Approve the rating before the shipment can be billed",
			numbers[0],
		)
	} else {
		err = errortypes.NewBusinessError(
			"The tariff held the rating of {0, plural, one {# shipment} other {# shipments}} for review. Approve each rating before those shipments can be billed",
			len(held),
		)
	}

	return err.WithParam("shipmentIds", strings.Join(held, ","))
}
