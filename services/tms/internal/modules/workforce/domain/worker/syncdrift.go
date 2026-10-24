package worker

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type WorkerSyncDrift struct {
	keel.Table `keel:"samsara_worker_sync_drifts"`

	OrganizationID  string `keel:"organization_id,pk,tenant"`
	BusinessUnitID  string `keel:"business_unit_id,pk,tenant"`
	WorkerID        string `keel:"worker_id,pk"`
	DriftType       string `keel:"drift_type,pk"`
	WorkerName      string `keel:"worker_name"`
	Message         string `keel:"message"`
	LocalExternalID string `keel:"local_external_id,nullzero"`
	RemoteDriverID  string `keel:"remote_driver_id,nullzero"`
	DetectedAt      int64  `keel:"detected_at"`
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
