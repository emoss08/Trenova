package detentionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StayStage string

const (
	StayStageAll         = StayStage("")
	StayStageToBill      = StayStage("ToBill")
	StayStageRunning     = StayStage("Running")
	StayStageBilled      = StayStage("Billed")
	StayStageDisputed    = StayStage("Disputed")
	StayStageWaived      = StayStage("Waived")
	StayStageNotBillable = StayStage("NotBillable")
)

const MaxStays = 100

type ListDetentionStaysRequest struct {
	TenantInfo pagination.TenantInfo
	From       int64
	To         int64
	Stage      StayStage
	LocationID pulid.ID
	Limit      int
	Offset     int
}

type DetentionStay struct {
	Occurrence      *detention.DetentionOccurrence
	ArrivalByHand   bool
	DepartureByHand bool
}

func (s *DetentionStay) HoldBack(now int64) detention.HoldBack {
	facts := detention.HoldBackFacts{
		Status:             s.Occurrence.Status,
		NotificationStatus: s.Occurrence.NotificationStatus,
		NoticeDeadlineAt:   s.Occurrence.NoticeDeadlineAt,
		RequiresApproval:   s.Occurrence.RequiresApproval,
		ArrivalByHand:      s.ArrivalByHand,
		DepartureByHand:    s.DepartureByHand,
	}

	return facts.HoldBack(now)
}

type DetentionStayPage struct {
	Items []*DetentionStay
	Total int
}

type DetentionStayRepository interface {
	ListStays(ctx context.Context, req *ListDetentionStaysRequest) (*DetentionStayPage, error)
}
