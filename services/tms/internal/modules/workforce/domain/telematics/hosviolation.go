package telematics

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type WorkerHOSViolation struct {
	keel.Table `keel:"worker_hos_violations"`

	OrganizationID   pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	WorkerID         pulid.ID `json:"workerId" keel:"worker_id,pk"`
	ViolationType    string   `json:"violationType" keel:"violation_type,pk"`
	ViolationStartAt int64    `json:"violationStartAt" keel:"violation_start_at,pk"`
	Description      string   `json:"description" keel:"description,nullzero"`
	DurationMs       int64    `json:"durationMs" keel:"duration_ms"`
	DayStartAt       *int64   `json:"dayStartAt" keel:"day_start_at"`
	DayEndAt         *int64   `json:"dayEndAt" keel:"day_end_at"`
	DetectedAt       int64    `json:"detectedAt" keel:"detected_at"`

	Worker *worker.Worker `json:"worker,omitempty" keel:"belongs:worker_id"`
}

func (v *WorkerHOSViolation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&v.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&v.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&v.ViolationType,
			validation.Required.Error("Violation type is required"),
			validation.Length(1, maxViolationTypeLength).
				Error("Violation type cannot be longer than 64 characters"),
		),
		validation.Field(&v.ViolationStartAt,
			validation.Required.Error("Violation start is required"),
			validation.Min(int64(1)).Error("Violation start must be a valid timestamp"),
		),
		validation.Field(&v.DurationMs,
			validation.Min(int64(0)).Error("Duration cannot be negative"),
		),
		// The day bounds are what the violation is reported inside of on a log
		// audit, so an inverted pair would place it in no day at all.
		validation.Field(&v.DayEndAt,
			validation.When(v.DayStartAt != nil,
				validation.Min(dayStartOrZero(v.DayStartAt)).
					Error("Day end cannot precede the day start"),
			),
		),
		validation.Field(&v.DetectedAt,
			validation.Required.Error("Detected at is required"),
			validation.Min(int64(1)).Error("Detected at must be a valid timestamp"),
		),
	))
}

func dayStartOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

const maxViolationTypeLength = 64
