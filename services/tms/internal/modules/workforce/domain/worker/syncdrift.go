package worker

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type WorkerSyncDrift struct {
	bun.BaseModel `bun:"table:samsara_worker_sync_drifts,alias:wsd"`

	OrganizationID  string `bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  string `bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	WorkerID        string `bun:"worker_id,pk,type:VARCHAR(100),notnull"`
	DriftType       string `bun:"drift_type,pk,type:VARCHAR(64),notnull"`
	WorkerName      string `bun:"worker_name,type:VARCHAR(255),notnull"`
	Message         string `bun:"message,type:TEXT,notnull"`
	LocalExternalID string `bun:"local_external_id,type:TEXT,nullzero"`
	RemoteDriverID  string `bun:"remote_driver_id,type:TEXT,nullzero"`
	DetectedAt      int64  `bun:"detected_at,type:BIGINT,notnull"`
}

func (d *WorkerSyncDrift) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&d.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&d.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&d.DriftType,
			validation.Required.Error("Drift type is required"),
			validation.Length(1, maxDriftTypeLength).
				Error("Drift type cannot be longer than 64 characters"),
		),
		validation.Field(&d.WorkerName,
			validation.Required.Error("Worker name is required"),
			validation.Length(1, maxDriftWorkerNameLength).
				Error("Worker name cannot be longer than 255 characters"),
		),
		// The message is what the drift report shows; a blank one would list a
		// discrepancy nobody can act on.
		validation.Field(&d.Message, validation.Required.Error("Message is required")),
		validation.Field(&d.DetectedAt,
			validation.Required.Error("Detected at is required"),
			validation.Min(int64(1)).Error("Detected at must be a valid timestamp"),
		),
	))
}

const (
	maxDriftTypeLength       = 64
	maxDriftWorkerNameLength = 255
)
