package telematics

import (
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type WorkerHOSLog struct {
	bun.BaseModel `bun:"table:worker_hos_logs,alias:whl" json:"-"`

	OrganizationID    pulid.ID   `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID   `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	WorkerID          pulid.ID   `json:"workerId"          bun:"worker_id,pk,type:VARCHAR(100),notnull"`
	LogStartAt        int64      `json:"logStartAt"        bun:"log_start_at,pk,type:BIGINT,notnull"`
	DutyStatus        DutyStatus `json:"dutyStatus"        bun:"duty_status,type:VARCHAR(32),notnull"`
	LogEndAt          *int64     `json:"logEndAt"          bun:"log_end_at,type:BIGINT,nullzero"`
	Remark            string     `json:"remark"            bun:"remark,type:TEXT,nullzero"`
	Provider          string     `json:"provider"          bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'"`
	ProviderVehicleID string     `json:"providerVehicleId" bun:"provider_vehicle_id,type:TEXT,nullzero"`
	ReceivedAt        int64      `json:"receivedAt"        bun:"received_at,type:BIGINT,notnull"`

	Worker *worker.Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}
