package dispatchplanports

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Request struct {
	TenantInfo   pagination.TenantInfo
	WindowStart  int64
	WindowEnd    int64
	FleetCodeIDs []pulid.ID
	MoveIDs      []pulid.ID
}

type Pairing struct {
	MoveID               pulid.ID                        `json:"moveId"`
	MoveVersion          int64                           `json:"moveVersion"`
	ProNumber            string                          `json:"proNumber"`
	OriginLabel          string                          `json:"originLabel"`
	DestinationLabel     string                          `json:"destinationLabel"`
	PickupOpens          int64                           `json:"pickupOpens"`
	Candidate            *CandidateScore                 `json:"candidate"`
	PlanCostCents        int64                           `json:"planCostCents"`
	PreferenceFloorCents int64                           `json:"preferenceFloorCents"`
	CoveragePriceCents   int64                           `json:"coveragePriceCents"`
	DriverValueCents     int64                           `json:"driverValueCents"`
	Alternatives         []dispatchplanprice.Alternative `json:"alternatives"`

	TourID              pulid.ID `json:"tourId"`
	SequenceIndex       int      `json:"sequenceIndex"`
	ProjectedStartAt    int64    `json:"projectedStartAt"`
	ProjectedCompleteAt int64    `json:"projectedCompleteAt"`
}

type Uncovered struct {
	MoveID               pulid.ID                          `json:"moveId"`
	ShipmentID           pulid.ID                          `json:"shipmentId"`
	ProNumber            string                            `json:"proNumber"`
	StartsAt             int64                             `json:"startsAt"`
	Reason               dispatchplanprice.UncoveredReason `json:"reason"`
	PlanCostCents        int64                             `json:"planCostCents"`
	PreferenceFloorCents int64                             `json:"preferenceFloorCents"`
	CoveragePriceCents   int64                             `json:"coveragePriceCents"`
	BestCandidate        *CandidateScore                   `json:"bestCandidate"`
}

type Tour struct {
	TourID     pulid.ID   `json:"tourId"`
	WorkerID   pulid.ID   `json:"workerId"`
	WorkerName string     `json:"workerName"`
	MoveIDs    []pulid.ID `json:"moveIds"`
	StartsAt   int64      `json:"startsAt"`
	EndsAt     int64      `json:"endsAt"`
	CostCents  int64      `json:"costCents"`
	EmptyMiles float64    `json:"emptyMiles"`
}

type Plan struct {
	Mode           dispatchcontrol.PlanningMode `json:"mode"`
	GeneratedAt    int64                        `json:"generatedAt"`
	Pairings       []*Pairing                   `json:"pairings"`
	Uncovered      []*Uncovered                 `json:"uncovered"`
	Tours          []*Tour                      `json:"tours"`
	CostCents      int64                        `json:"costCents"`
	UncoveredCents int64                        `json:"uncoveredCents"`
}

type Planner interface {
	Plan(ctx context.Context, req *Request) (*Plan, error)
}

const (
	noCandidateText     = "No driver could take this move."
	locationUnknownText = "No driver near the pickup has a known location."
)

func (u *Uncovered) Explanation() string {
	best := u.BestCandidate
	if best == nil {
		switch u.Reason {
		case dispatchplanprice.UncoveredLocationUnknown:
			return locationUnknownText
		case dispatchplanprice.UncoveredNotWorthIt:
			return fmt.Sprintf(
				"Every driver costs more than the %s it is worth to cover.",
				dispatcheconomics.Dollars(u.CoveragePriceCents),
			)
		case dispatchplanprice.UncoveredNoCandidate,
			dispatchplanprice.UncoveredDisqualified,
			dispatchplanprice.UncoveredBeyondDeadhead,
			dispatchplanprice.UncoveredOutbid:
			return noCandidateText
		default:
			return noCandidateText
		}
	}

	switch u.Reason {
	case dispatchplanprice.UncoveredDisqualified:
		if finding, ok := best.FirstBlock(); ok {
			return fmt.Sprintf(
				"The nearest driver, %s, can't take it: %s.",
				best.WorkerName,
				finding,
			)
		}
		return fmt.Sprintf("The nearest driver, %s, can't take it.", best.WorkerName)
	case dispatchplanprice.UncoveredBeyondDeadhead:
		return fmt.Sprintf("The nearest driver, %s, is past the empty-mile limit.", best.WorkerName)
	case dispatchplanprice.UncoveredLocationUnknown:
		return locationUnknownText
	case dispatchplanprice.UncoveredNotWorthIt:
		return fmt.Sprintf(
			"Every driver costs more than leaving it: the cheapest, %s, costs %s against %s.",
			best.WorkerName,
			dispatcheconomics.Dollars(u.PlanCostCents),
			dispatcheconomics.Dollars(u.CoveragePriceCents),
		)
	case dispatchplanprice.UncoveredOutbid:
		return fmt.Sprintf(
			"Its drivers are worth more on other moves; the cheapest, %s, would cost %s.",
			best.WorkerName,
			dispatcheconomics.Dollars(best.Cost.TotalCents),
		)
	case dispatchplanprice.UncoveredNoCandidate:
		return noCandidateText
	default:
		return noCandidateText
	}
}
