package telematics

import "github.com/emoss08/trenova/shared/pulid"

type HOSLink string

const (
	HOSLinkNotConnected = HOSLink("NotConnected")
	HOSLinkNotLinked    = HOSLink("NotLinked")
	HOSLinkNoData       = HOSLink("NoData")
	HOSLinkReporting    = HOSLink("Reporting")
)

func (l HOSLink) String() string { return string(l) }

type HOSLinkState struct {
	Link             HOSLink
	DutyStatus       *string
	DutyStatusSince  *int64
	DriveRemainingMs *int64
	DriveLimitMs     *int64
	ShiftRemainingMs *int64
	ShiftLimitMs     *int64
	CycleRemainingMs *int64
	CycleLimitMs     *int64
	BreakRemainingMs *int64
	BreakLimitMs     *int64
	CurrentTractorID pulid.ID
	RecordedAt       *int64
}
