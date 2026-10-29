package shipmentrating

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func ApprovalCovers(a *shipment.RatingReviewApproval, reasons []string) bool {
	if a == nil || len(a.Reviews) != len(reasons) {
		return false
	}
	approved := slices.Clone(a.Reviews)
	pending := slices.Clone(reasons)
	slices.Sort(approved)
	slices.Sort(pending)
	return slices.Equal(approved, pending)
}

func ReviewReasons(d *shipment.RatingDetail) []string {
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

func DetailAwaitsReview(d *shipment.RatingDetail) bool {
	reasons := ReviewReasons(d)
	return len(reasons) > 0 && !ApprovalCovers(d.ReviewApproval, reasons)
}

func CarryReviewApproval(d, previous *shipment.RatingDetail) {
	if d == nil {
		return
	}
	d.ReviewApproval = nil
	if previous == nil || previous.ReviewApproval == nil {
		return
	}
	reasons := ReviewReasons(d)
	if len(reasons) == 0 || !ApprovalCovers(previous.ReviewApproval, reasons) {
		return
	}
	approval := *previous.ReviewApproval
	approval.Reviews = slices.Clone(previous.ReviewApproval.Reviews)
	d.ReviewApproval = &approval
}

func ApproveReview(d *shipment.RatingDetail, userID pulid.ID, at int64) (bool, error) {
	reasons := ReviewReasons(d)
	if len(reasons) == 0 {
		return false, errortypes.NewBusinessError(
			"This shipment's rating has nothing waiting for review",
		)
	}
	if ApprovalCovers(d.ReviewApproval, reasons) {
		return false, nil
	}
	d.ReviewApproval = &shipment.RatingReviewApproval{
		Reviews:      reasons,
		ApprovedByID: userID,
		ApprovedAt:   at,
	}
	return true, nil
}

func AwaitsReview(s *shipment.Shipment) bool {
	return s != nil && DetailAwaitsReview(s.RatingDetail)
}

func ReviewHoldError(shipments []*shipment.Shipment) error {
	held := make([]string, 0, len(shipments))
	numbers := make([]string, 0, len(shipments))
	for _, entity := range shipments {
		if !AwaitsReview(entity) {
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
