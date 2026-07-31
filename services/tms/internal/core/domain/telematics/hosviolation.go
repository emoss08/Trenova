package telematics

import (
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type WorkerHOSViolation struct {
	bun.BaseModel `bun:"table:worker_hos_violations,alias:whv" json:"-"`

	OrganizationID   pulid.ID `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	WorkerID         pulid.ID `json:"workerId"         bun:"worker_id,pk,type:VARCHAR(100),notnull"`
	ViolationType    string   `json:"violationType"    bun:"violation_type,pk,type:VARCHAR(64),notnull"`
	ViolationStartAt int64    `json:"violationStartAt" bun:"violation_start_at,pk,type:BIGINT,notnull"`
	Description      string   `json:"description"      bun:"description,type:TEXT,nullzero"`
	DurationMs       int64    `json:"durationMs"       bun:"duration_ms,type:BIGINT,notnull,default:0"`
	DayStartAt       *int64   `json:"dayStartAt"       bun:"day_start_at,type:BIGINT,nullzero"`
	DayEndAt         *int64   `json:"dayEndAt"         bun:"day_end_at,type:BIGINT,nullzero"`
	DetectedAt       int64    `json:"detectedAt"       bun:"detected_at,type:BIGINT,notnull"`

	Worker *worker.Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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
