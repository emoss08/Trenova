package telematics

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type WorkerHOSLog struct {
	bun.BaseModel `bun:"table:worker_hos_logs,alias:whl" json:"-"`
	keel.Table    `keel:"worker_hos_logs"`

	OrganizationID    pulid.ID   `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID   `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	WorkerID          pulid.ID   `json:"workerId"          bun:"worker_id,pk,type:VARCHAR(100),notnull" keel:"worker_id,pk"`
	LogStartAt        int64      `json:"logStartAt"        bun:"log_start_at,pk,type:BIGINT,notnull" keel:"log_start_at,pk"`
	DutyStatus        DutyStatus `json:"dutyStatus"        bun:"duty_status,type:VARCHAR(32),notnull" keel:"duty_status"`
	LogEndAt          *int64     `json:"logEndAt"          bun:"log_end_at,type:BIGINT,nullzero" keel:"log_end_at"`
	Remark            string     `json:"remark"            bun:"remark,type:TEXT,nullzero" keel:"remark,nullzero"`
	Provider          string     `json:"provider"          bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'" keel:"provider"`
	ProviderVehicleID string     `json:"providerVehicleId" bun:"provider_vehicle_id,type:TEXT,nullzero" keel:"provider_vehicle_id,nullzero"`
	ReceivedAt        int64      `json:"receivedAt"        bun:"received_at,type:BIGINT,notnull" keel:"received_at"`

	Worker *worker.Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:worker_id"`
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
