package detentionintelligence

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// Request asks for detention intelligence over the last Days whole days of the organization's
// time zone, ending with Through (YYYY-MM-DD) or today when it is empty. The window before it,
// of the same length, is read for comparison. Visible is the protected field groups the reader
// holds: the figures and the sentences built from them leave out what the reader may not see.
type Request struct {
	TenantInfo pagination.TenantInfo
	Days       int
	Through    string
	LocationID pulid.ID
	Visible    protectedfield.GroupSet
}

type Facility struct {
	detentionports.FacilityIntelligence
	// PreviousPastFreeMinutes is the facility's minutes past free time in the window before, or
	// nil when it did not rank there.
	PreviousPastFreeMinutes *int64
}

// RecoveryStep is one bucket of where the accrued money ended up, in a fixed order.
type RecoveryStep struct {
	Bucket          string
	OccurrenceCount int
	Amount          decimal.Decimal
}

// Recovery follows the money from what the clock accrued to what was billed, invoiced on a
// posted invoice and collected on a paid one.
type Recovery struct {
	Accrued   decimal.Decimal
	Steps     []RecoveryStep
	Billed    decimal.Decimal
	Invoiced  decimal.Decimal
	Collected decimal.Decimal
}

// PolicyChange is a facility moving to another detention policy partway through the window.
type PolicyChange struct {
	LocationID   pulid.ID
	LocationName string
	PolicyID     pulid.ID
	PolicyName   string
	ChangedAt    int64
}

type SegmentTarget string

const (
	TargetFacility SegmentTarget = "Facility"
	TargetCustomer SegmentTarget = "Customer"
	TargetSection  SegmentTarget = "Section"
)

// The sections of the page a sentence can point at.
const (
	SectionFocus     = "focus"
	SectionPaid      = "paid"
	SectionCustomers = "customers"
)

const (
	toneDanger = "danger"
	toneGood   = "good"
)

// Segment is one run of a sentence: plain, strong, or a reference that opens what it names.
type Segment struct {
	Text     string
	Strong   bool
	Tone     string
	Target   SegmentTarget
	TargetID string
}

type StepKind string

const (
	StepShipperReview StepKind = "ShipperReview"
	StepFreeTime      StepKind = "FreeTime"
	StepNotice        StepKind = "Notice"
	StepCustomerTerms StepKind = "CustomerTerms"
	StepLateArrivals  StepKind = "LateArrivals"
	StepEarlyArrivals StepKind = "EarlyArrivals"
)

// NextStep is one move the figures point to, sized in minutes or money so the steps can be
// ranked. The client labels the action from the kind.
type NextStep struct {
	Kind          StepKind
	Title         []Segment
	Detail        string
	ImpactMinutes *int64
	ImpactAmount  *decimal.Decimal
	ImpactCaption string
	LocationID    pulid.ID
	CustomerID    pulid.ID
}

type Intelligence struct {
	Days int
	// Through is the window's last day, YYYY-MM-DD in the organization's time zone: a read of
	// one facility passes it to cover the same days.
	Through       string
	From          int64
	To            int64
	PreviousFrom  int64
	PreviousTo    int64
	Timezone      string
	Totals        detentionports.IntelligenceTotals
	Previous      detentionports.IntelligenceTotals
	Facilities    []Facility
	Customers     []detentionports.CustomerIntelligence
	Waivers       []detentionports.WaiverLeakageStat
	Recovery      Recovery
	Arrivals      []detentionports.ArrivalCell
	Splits        detentionports.Splits
	Weeks         []detentionports.WeekBucket
	PolicyChanges []PolicyChange
	// Summary is the briefing: paragraphs, each a run of segments.
	Summary   [][]Segment
	NextSteps []NextStep
	// Truncated is true when more facilities had detention than the read returns, so the
	// facility list, and anything ranked from it, leaves some out.
	Truncated bool
}
