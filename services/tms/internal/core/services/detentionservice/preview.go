package detentionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

// PreviewScenario is a hypothetical stop used to exercise a policy before it is
// activated.
type PreviewScenario struct {
	ArrivedAt        int64                     `json:"arrivedAt"`
	DepartedAt       *int64                    `json:"departedAt"`
	AppointmentStart *int64                    `json:"appointmentStart"`
	AppointmentEnd   *int64                    `json:"appointmentEnd"`
	StopType         shipment.StopType         `json:"stopType"`
	ScheduleType     shipment.StopScheduleType `json:"scheduleType"`
	NoticeSentAt     *int64                    `json:"noticeSentAt"`
	DriverPayRate    *string                   `json:"driverPayRate"`
}

// PreviewResult is what the configured terms would produce for the scenario.
type PreviewResult struct {
	Snapshot           detention.PolicySnapshot     `json:"policySnapshot"`
	RawDwellMinutes    int32                        `json:"rawDwellMinutes"`
	FreeMinutesGranted int32                        `json:"freeMinutesGranted"`
	BillableMinutes    int32                        `json:"billableMinutes"`
	RoundedMinutes     int32                        `json:"roundedMinutes"`
	BillableAmount     decimal.Decimal              `json:"billableAmount"`
	GrossAmount        decimal.Decimal              `json:"grossAmount"`
	DriverPayAmount    decimal.Decimal              `json:"driverPayAmount"`
	NetMargin          decimal.Decimal              `json:"netMargin"`
	ArrivedLate        bool                         `json:"arrivedLate"`
	CapApplied         detention.CapKind            `json:"capApplied"`
	Status             detention.OccurrenceStatus   `json:"status"`
	NotificationStatus detention.NotificationStatus `json:"notificationStatus"`
	SuppressedByGate   bool                         `json:"suppressedByGate"`
	Trace              *detention.CalculationTrace  `json:"calculationTrace"`
	Receipt            string                       `json:"receipt"`
}

// PreviewPolicy runs the real engine against a hypothetical stop.
//
// The preview deliberately reuses the production calculator rather than
// re-deriving the math for the UI. Configuration without an accurate feedback
// loop is how carriers end up with the invoice error rate the industry has.
func (s *Service) PreviewPolicy(
	ctx context.Context,
	policy *detention.DetentionPolicy,
	scenario PreviewScenario,
	tenantInfo pagination.TenantInfo,
) (*PreviewResult, error) {
	if policy == nil {
		return nil, errortypes.NewValidationError(
			"policy", errortypes.ErrRequired, "A policy is required to preview")
	}

	multiErr := errortypes.NewMultiError()
	policy.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if scenario.ArrivedAt <= 0 {
		return nil, errortypes.NewValidationError(
			"scenario.arrivedAt", errortypes.ErrRequired, "An arrival time is required")
	}

	if scenario.StopType == "" {
		scenario.StopType = shipment.StopTypeDelivery
	}
	if scenario.ScheduleType == "" {
		scenario.ScheduleType = shipment.StopScheduleTypeAppointment
	}

	accessorial, err := s.accessorialRepo.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
		ID:         policy.AccessorialChargeID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	policy.SpecificityScore = policy.ComputeSpecificity()

	now := s.now()
	if scenario.DepartedAt != nil && *scenario.DepartedAt > now {
		now = *scenario.DepartedAt
	}

	snapshot := detention.NewPolicySnapshot(detention.SnapshotInput{
		Policy:          policy,
		StopType:        string(scenario.StopType),
		FreeMinutes:     policy.FreeMinutesForStopType(scenario.StopType),
		AccessorialCode: accessorial.Code,
		FlatRate:        accessorial.Amount,
		FlatRateUnit:    tierUnitFromAccessorial(accessorial.RateUnit),
		CapturedAt:      now,
	})

	computed := Compute(ComputeInput{
		Snapshot:         snapshot,
		StopType:         scenario.StopType,
		ScheduleType:     scenario.ScheduleType,
		ArrivedAt:        &scenario.ArrivedAt,
		DepartedAt:       scenario.DepartedAt,
		AppointmentStart: scenario.AppointmentStart,
		AppointmentEnd:   scenario.AppointmentEnd,
		NoticeSentAt:     scenario.NoticeSentAt,
		Now:              now,
		Location:         s.tenantLocation(ctx, tenantInfo.OrgID),
		DriverPayRate:    parsePayRate(scenario.DriverPayRate),
		ShipmentAccrued:  decimal.Zero,
	})

	result := &PreviewResult{
		Snapshot:           snapshot,
		RawDwellMinutes:    computed.RawDwellMinutes,
		FreeMinutesGranted: computed.FreeMinutesGranted,
		BillableMinutes:    computed.BillableMinutes,
		RoundedMinutes:     computed.RoundedMinutes,
		BillableAmount:     computed.BillableAmount,
		GrossAmount:        computed.GrossAmount,
		DriverPayAmount:    computed.DriverPayAmount,
		NetMargin:          computed.NetMargin,
		ArrivedLate:        computed.ArrivedLate,
		CapApplied:         computed.CapApplied,
		Status:             computed.Status,
		NotificationStatus: computed.NotificationStatus,
		SuppressedByGate:   computed.SuppressedByGate,
		Trace:              computed.Trace,
	}

	if computed.Trace != nil {
		result.Receipt = computed.Trace.Receipt()
	}

	return result, nil
}

func parsePayRate(raw *string) decimal.NullDecimal {
	if raw == nil || *raw == "" {
		return decimal.NullDecimal{}
	}

	parsed, err := decimal.NewFromString(*raw)
	if err != nil {
		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(parsed)
}
