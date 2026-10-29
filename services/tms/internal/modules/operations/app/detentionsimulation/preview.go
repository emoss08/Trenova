package detentionsimulation

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
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
	Snapshot           detentionpolicy.PolicySnapshot `json:"policySnapshot"`
	RawDwellMinutes    int32                          `json:"rawDwellMinutes"`
	FreeMinutesGranted int32                          `json:"freeMinutesGranted"`
	BillableMinutes    int32                          `json:"billableMinutes"`
	RoundedMinutes     int32                          `json:"roundedMinutes"`
	BillableAmount     decimal.Decimal                `json:"billableAmount"`
	GrossAmount        decimal.Decimal                `json:"grossAmount"`
	DriverPayAmount    decimal.NullDecimal            `json:"driverPayAmount"`
	NetMargin          decimal.NullDecimal            `json:"netMargin"`
	ArrivedLate        bool                           `json:"arrivedLate"`
	CapApplied         detention.CapKind              `json:"capApplied"`
	Status             detention.OccurrenceStatus     `json:"status"`
	NotificationStatus detention.NotificationStatus   `json:"notificationStatus"`
	SuppressedByGate   bool                           `json:"suppressedByGate"`
	Trace              *detention.CalculationTrace    `json:"calculationTrace"`
	Receipt            string                         `json:"receipt"`
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
