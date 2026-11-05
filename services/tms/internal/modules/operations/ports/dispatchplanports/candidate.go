package dispatchplanports

import (
	"cmp"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/shared/pulid"
)

type OriginSource string

const (
	OriginUnknown    = OriginSource("")
	OriginCommitment = OriginSource("commitment")
	OriginPosition   = OriginSource("position")
	OriginLastDrop   = OriginSource("lastDrop")
)

type CandidateScore struct {
	WorkerID   pulid.ID `json:"workerId"`
	WorkerName string   `json:"workerName"`
	TractorID  pulid.ID `json:"tractorId"`
	TrailerID  pulid.ID `json:"trailerId"`
	MoveID     pulid.ID `json:"moveId"`

	Cost      dispatcheconomics.Cost `json:"cost"`
	HardCents int64                  `json:"hardCents"`
	SoftCents int64                  `json:"softCents"`
	Verdict   string                 `json:"verdict"`

	DeadheadMiles       *float64     `json:"deadheadMiles"`
	OriginSource        OriginSource `json:"originSource"`
	LoadedMiles         float64      `json:"loadedMiles"`
	EstimatedDriveMs    int64        `json:"estimatedDriveMs"`
	RestMs              int64        `json:"restMs"`
	ProjectedArrival    int64        `json:"projectedArrival"`
	ProjectedDelivery   int64        `json:"projectedDelivery"`
	ProjectedFreeAt     int64        `json:"projectedFreeAt"`
	MinutesOfSlack      int64        `json:"minutesOfSlack"`
	SlackKnown          bool         `json:"slackKnown"`
	DeliveryLateMinutes int64        `json:"deliveryLateMinutes"`
	ProjectedAvailable  int64        `json:"projectedTimeAvailable"`

	HOSKnown             bool   `json:"hosKnown"`
	DriveRemainingMs     int64  `json:"driveRemainingMs"`
	ShiftRemainingMs     int64  `json:"shiftRemainingMs"`
	CycleRemainingMs     int64  `json:"cycleRemainingMs"`
	HOSStrategy          string `json:"hosStrategy"`
	HOSRestStartDeadline int64  `json:"hosRestStartDeadline"`
	HOSProjectedDriveMs  int64  `json:"hosProjectedDriveMs"`
	HOSProjectedShiftMs  int64  `json:"hosProjectedShiftMs"`
	HOSProjectedCycleMs  int64  `json:"hosProjectedCycleMs"`

	Findings []dispatchfinding.Finding `json:"findings"`
}

func (c *CandidateScore) Blocked() bool {
	for i := range c.Findings {
		if c.Findings[i].Severity == dispatchfinding.SeverityBlock {
			return true
		}
	}
	return false
}

func (c *CandidateScore) FirstBlock() (string, bool) {
	if c == nil {
		return "", false
	}
	for i := range c.Findings {
		if c.Findings[i].Severity == dispatchfinding.SeverityBlock {
			return c.Findings[i].Message, true
		}
	}
	return "", false
}

func (c *CandidateScore) Warnings() int {
	count := 0
	for i := range c.Findings {
		if c.Findings[i].Severity == dispatchfinding.SeverityWarn {
			count++
		}
	}
	return count
}

func (c *CandidateScore) LocationKnown() bool {
	return c.DeadheadMiles != nil
}

func CompareRank(left, right *CandidateScore) int {
	if c := compareBool(left.Blocked(), right.Blocked()); c != 0 {
		return c
	}
	if c := compareBool(!left.LocationKnown(), !right.LocationKnown()); c != 0 {
		return c
	}

	return cmp.Compare(left.Cost.TotalCents, right.Cost.TotalCents)
}

func compareBool(left, right bool) int {
	switch {
	case left == right:
		return 0
	case left:
		return 1
	default:
		return -1
	}
}
