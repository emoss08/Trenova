package telematics

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type VehicleInspectionDefect struct {
	ID         string `json:"id"`
	DefectType string `json:"defectType"`
	Comment    string `json:"comment,omitempty"`
	Resolved   bool   `json:"resolved"`
	ResolvedAt *int64 `json:"resolvedAt,omitempty"`
}

type VehicleInspection struct {
	bun.BaseModel `bun:"table:vehicle_inspections,alias:vinsp" json:"-"`

	ID                    pulid.ID                  `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID        pulid.ID                  `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID                  `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Provider              string                    `json:"provider"              bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'"`
	ProviderDvirID        string                    `json:"providerDvirId"        bun:"provider_dvir_id,type:TEXT,notnull"`
	TractorID             pulid.ID                  `json:"tractorId"             bun:"tractor_id,type:VARCHAR(100),nullzero"`
	WorkerID              pulid.ID                  `json:"workerId"              bun:"worker_id,type:VARCHAR(100),nullzero"`
	InspectionType        string                    `json:"inspectionType"        bun:"inspection_type,type:VARCHAR(32),notnull"`
	SafetyStatus          string                    `json:"safetyStatus"          bun:"safety_status,type:VARCHAR(16),notnull"`
	StartedAt             int64                     `json:"startedAt"             bun:"started_at,type:BIGINT,notnull"`
	EndedAt               int64                     `json:"endedAt"               bun:"ended_at,type:BIGINT,notnull"`
	OdometerMeters        *int64                    `json:"odometerMeters"        bun:"odometer_meters,type:BIGINT,nullzero"`
	Location              string                    `json:"location"              bun:"location,type:TEXT,nullzero"`
	Signed                bool                      `json:"signed"                bun:"signed,type:BOOLEAN,notnull,default:false"`
	DefectCount           int                       `json:"defectCount"           bun:"defect_count,type:INT,notnull,default:0"`
	UnresolvedDefectCount int                       `json:"unresolvedDefectCount" bun:"unresolved_defect_count,type:INT,notnull,default:0"`
	Defects               []VehicleInspectionDefect `json:"defects"               bun:"defects,type:JSONB,nullzero"`
	CreatedAt             int64                     `json:"createdAt"             bun:"created_at,type:BIGINT,notnull"`
}

func NewVehicleInspectionID() pulid.ID {
	return pulid.MustNew("vinsp_")
}
