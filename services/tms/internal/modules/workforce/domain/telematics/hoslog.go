package telematics

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type WorkerHOSLog struct {
	keel.Table `keel:"worker_hos_logs"`

	OrganizationID    pulid.ID   `json:"organizationId"    keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID   `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	WorkerID          pulid.ID   `json:"workerId"          keel:"worker_id,pk"`
	LogStartAt        int64      `json:"logStartAt"        keel:"log_start_at,pk"`
	DutyStatus        DutyStatus `json:"dutyStatus"        keel:"duty_status"`
	LogEndAt          *int64     `json:"logEndAt"          keel:"log_end_at"`
	Remark            string     `json:"remark"            keel:"remark,nullzero"`
	Provider          string     `json:"provider"          keel:"provider"`
	ProviderVehicleID string     `json:"providerVehicleId" keel:"provider_vehicle_id,nullzero"`
	ReceivedAt        int64      `json:"receivedAt"        keel:"received_at"`

	Worker *worker.Worker `json:"worker,omitempty" keel:"belongs:worker_id"`
}

func (l *WorkerHOSLog) BeforeCreate(_ context.Context) error {
	if l.Provider == "" {
		l.Provider = defaultProvider
	}

	return nil
}

func (l *WorkerHOSLog) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(&l.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&l.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&l.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&l.DutyStatus,
			validation.Required.Error("Duty status is required"),
			domainvalidation.ValidEnum[DutyStatus]("Duty status is invalid"),
		),
		validation.Field(&l.LogStartAt,
			validation.Required.Error("Log start is required"),
			validation.Min(int64(1)).Error("Log start must be a valid timestamp"),
		),
		// A log entry that ends before it starts would report negative time in
		// a duty status, which is the number an hours-of-service audit reads.
		validation.Field(&l.LogEndAt,
			validation.Min(l.LogStartAt).Error("Log end cannot precede the log start"),
		),
		validation.Field(&l.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		validation.Field(&l.ReceivedAt,
			validation.Required.Error("Received at is required"),
			validation.Min(int64(1)).Error("Received at must be a valid timestamp"),
		),
	))
}
