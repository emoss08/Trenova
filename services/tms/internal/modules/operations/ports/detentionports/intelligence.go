package detentionports

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// InvoiceSettlementColumns are the billing columns detention intelligence reads to say whether a
// billed detention charge reached a posted invoice and whether that invoice was paid. Billing is
// a higher module, so the columns and the two status predicates come in through
// bootstrap/crossref rather than an import of its tables.
type InvoiceSettlementColumns struct {
	LineInvoiceID keelkinds.PulidIDCol
	LineChargeID  keelkinds.PulidIDCol
	InvoiceID     keelkinds.PulidIDCol
	// InvoicePosted holds for an invoice that was posted and not voided.
	InvoicePosted keel.Pred
	// InvoicePaid holds for an invoice settled in full.
	InvoicePaid keel.Pred
}

// IntelligenceWindow is whole days of the organization's time zone, as instants: From is the
// first day's midnight and To the last instant of the last day. [MonthsFrom, MonthsEnd) are the
// whole months inside it, empty when the two are equal, so a read takes them from monthly figures
// and only the days around them from daily ones.
type IntelligenceWindow struct {
	From       int64
	To         int64
	MonthsFrom int64
	MonthsEnd  int64
}

// IntelligenceRequest scopes one detention intelligence read: the window and the window before it
// for comparison, and the organization's time zone, which weekdays and arrival hours are counted
// in. A location narrows every figure to one facility, which is what its scorecard reads.
type IntelligenceRequest struct {
	TenantInfo pagination.TenantInfo
	Current    IntelligenceWindow
	Previous   IntelligenceWindow
	Timezone   string
	LocationID pulid.ID
	Limit      int
}

type IntelligenceTotals struct {
	StopCount       int
	DetainedCount   int
	PastFreeMinutes int64
	// The minutes past free time split by who caused the wait. A stop counts once, in the
	// first of: arrived late, no appointment (an open dock), arrived before its appointment, or
	// on time with the dock slow.
	DockMinutes       int64
	OpenMinutes       int64
	EarlyMinutes      int64
	LateMinutes       int64
	BilledAmount      decimal.Decimal
	DriverPayAmount   decimal.NullDecimal
	NetMargin         decimal.NullDecimal
	WaivedAmount      decimal.Decimal
	DisputeCount      int
	SuppressedCount   int
	NoticeDueCount    int
	NoticeOnTimeCount int
}

type FacilityIntelligence struct {
	LocationID      pulid.ID
	LocationName    string
	LocationCity    string
	LocationState   string
	StopCount       int
	DetainedCount   int
	PastFreeMinutes int64
	// The minutes past free time split by who caused the wait. A stop counts once, in the
	// first of: arrived late, no appointment (an open dock), arrived before its appointment, or
	// on time with the dock slow.
	DockMinutes        int64
	OpenMinutes        int64
	EarlyMinutes       int64
	LateMinutes        int64
	AvgDwellMinutes    int
	MedianDwellMinutes float64
	P90DwellMinutes    float64
	MedianFreeMinutes  float64
	BilledAmount       decimal.Decimal
	DriverPayAmount    decimal.NullDecimal
	NetMargin          decimal.NullDecimal
	WaivedAmount       decimal.Decimal
	DisputeCount       int
	SuppressedCount    int
	NoticeDueCount     int
	NoticeOnTimeCount  int
}

type CustomerIntelligence struct {
	CustomerID        pulid.ID
	CustomerName      string
	StopCount         int
	DetainedCount     int
	PastFreeMinutes   int64
	BilledAmount      decimal.Decimal
	DriverPayAmount   decimal.NullDecimal `json:"driverPayAmount"`
	NetMargin         decimal.NullDecimal `json:"netMargin"`
	WaivedAmount      decimal.Decimal
	DisputeCount      int
	NoticeDueCount    int
	NoticeOnTimeCount int
}

// FacilityMinutes is one facility's hours past free time in the window before, to show who
// moved.
type FacilityMinutes struct {
	LocationID      pulid.ID
	PastFreeMinutes int64
}

// RecoveryBucket is where the money detention accrued ended up, one bucket per occurrence:
// RecoveryLate, RecoveryNotice, RecoveryWaived, RecoveryDisputed, RecoveryPending,
// RecoveryBilled or RecoveryOther.
type RecoveryBucket struct {
	Bucket          string
	OccurrenceCount int
	GrossAmount     decimal.Decimal
	BillableAmount  decimal.Decimal
	InvoicedAmount  decimal.Decimal
	CollectedAmount decimal.Decimal
}

const (
	RecoveryLate     = "Late"
	RecoveryNotice   = "Notice"
	RecoveryWaived   = "Waived"
	RecoveryDisputed = "Disputed"
	RecoveryPending  = "Pending"
	RecoveryBilled   = "Billed"
	RecoveryOther    = "Other"
	// RecoveryCapped is what billed charges lost to caps, minimums and rounding: accrued less
	// billed on the charges that were billed. It is worked out, never a bucket a row falls in.
	RecoveryCapped = "Capped"
)

// ArrivalCell counts stops by the ISO weekday (1 is Monday) and the hour, in the organization's
// time zone, that the truck arrived.
type ArrivalCell struct {
	Weekday       int
	Hour          int
	StopCount     int
	DetainedCount int
}

type SplitFigure struct {
	StopCount       int
	DetainedCount   int
	PastFreeMinutes int64
}

// Splits compares the stops that wait: pickups against deliveries, appointments against open
// docks, live stops against drops.
type Splits struct {
	Pickup      SplitFigure
	Delivery    SplitFigure
	Appointment SplitFigure
	Open        SplitFigure
	Live        SplitFigure
	Drop        SplitFigure
}

// WeekBucket is one week of the window. WeekStart is the local Monday at midnight, written as
// if it were UTC, so it names the week without a time zone.
type WeekBucket struct {
	WeekStart       int64
	StopCount       int
	DetainedCount   int
	PastFreeMinutes int64
}

// PolicyUse is one policy's use at one facility inside the window; a facility with more than one
// changed policy partway through.
type PolicyUse struct {
	LocationID      pulid.ID
	PolicyID        pulid.ID
	PolicyName      string
	FirstClockStart int64
	StopCount       int
}

// DayFigure is one organization-local day of the window: Day is its midnight.
type DayFigure struct {
	Day             time.Time
	StopCount       int
	DetainedCount   int
	PastFreeMinutes int64
}

type IntelligenceRollup struct {
	Totals             IntelligenceTotals
	Previous           IntelligenceTotals
	Facilities         []FacilityIntelligence
	PreviousFacilities []FacilityMinutes
	Customers          []CustomerIntelligence
	Waivers            []WaiverLeakageStat
	Recovery           []RecoveryBucket
	Arrivals           []ArrivalCell
	Splits             Splits
	Days               []DayFigure
	PolicyUses         []PolicyUse
}

type DetentionIntelligenceRepository interface {
	Intelligence(ctx context.Context, req *IntelligenceRequest) (*IntelligenceRollup, error)
	// OrganizationTimezone is the IANA zone the organization's days are counted in.
	OrganizationTimezone(ctx context.Context, tenant pagination.TenantInfo) (string, error)
}
