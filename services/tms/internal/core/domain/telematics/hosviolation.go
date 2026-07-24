package telematics

import (
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
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
