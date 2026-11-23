package workeroverviewservice

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/shopspring/decimal"
)

const ConcernCap = 30

type DrugAlcoholSection struct {
	Standing           *workerdot.DrugAlcoholStanding
	NextQueryDaysUntil *int64
}

type EmploymentSection struct {
	EventCount         int
	HiredAt            *int64
	OpenChecklistCount *int
}

type ScheduleSection struct {
	CurrentShiftName *string
}

type HOSSection struct {
	Link             telematics.HOSLink
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
	TractorCode      *string
	RecordedAt       *int64
}

type PayRate struct {
	Kind       driverpay.ComponentKind
	Method     driverpay.CalcMethod
	Rate       decimal.Decimal
	Overridden bool
}

type PaySection struct {
	ProfileName    string
	Classification driverpay.PayeeClassification
	EffectiveFrom  int64
	Headline       *PayRate
}

type BenefitsSection struct {
	CoveredCount               int
	NotStartedCount            int
	EmployeeCostPerPeriodMinor int64
}

type PortalAccess string

const (
	PortalAccessLinked   = PortalAccess("Linked")
	PortalAccessInvited  = PortalAccess("Invited")
	PortalAccessNotSetUp = PortalAccess("NotSetUp")
)

func (a PortalAccess) String() string { return string(a) }

type PortalSection struct {
	Access                    PortalAccess
	PendingProfileChangeCount *int
	LastSignInAt              *int64
}

func paySectionOf(assignment *payassignment.WorkerPayAssignment) *PaySection {
	profile := assignment.PayProfile
	section := &PaySection{
		ProfileName:    profile.Name,
		Classification: profile.Classification,
		EffectiveFrom:  assignment.EffectiveFrom,
	}
	overrides := make(map[string]decimal.Decimal, len(assignment.RateOverrides))
	for _, override := range assignment.RateOverrides {
		overrides[override.ComponentID.String()] = override.Rate
	}
	var first *driverpay.PayProfileComponent
	for _, component := range profile.Components {
		if component == nil || !component.IsActive {
			continue
		}
		if rate, ok := overrides[component.ID.String()]; ok {
			section.Headline = &PayRate{
				Kind:       component.Kind,
				Method:     component.Method,
				Rate:       rate,
				Overridden: true,
			}
			return section
		}
		if first == nil {
			first = component
		}
	}
	if first != nil {
		section.Headline = &PayRate{Kind: first.Kind, Method: first.Method, Rate: first.Rate}
	}
	return section
}

func planYearOf(now int64) int16 {
	return int16(time.Unix(now, 0).UTC().Year()) //nolint:gosec // a calendar year fits int16
}
