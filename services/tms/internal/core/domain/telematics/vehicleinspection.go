package telematics

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

func (i *VehicleInspection) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&i.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&i.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		validation.Field(&i.ProviderDvirID,
			validation.Required.Error("Provider inspection id is required"),
		),
		validation.Field(&i.InspectionType,
			validation.Required.Error("Inspection type is required"),
			validation.Length(1, maxInspectionTypeLength).
				Error("Inspection type cannot be longer than 32 characters"),
		),
		// The safety status is what takes a tractor out of service, so it is
		// never left blank on a stored inspection.
		validation.Field(&i.SafetyStatus,
			validation.Required.Error("Safety status is required"),
			validation.Length(1, maxSafetyStatusLength).
				Error("Safety status cannot be longer than 16 characters"),
		),
		validation.Field(&i.StartedAt,
			validation.Required.Error("Started at is required"),
			validation.Min(int64(1)).Error("Started at must be a valid timestamp"),
		),
		validation.Field(&i.EndedAt,
			validation.Required.Error("Ended at is required"),
			validation.Min(i.StartedAt).Error("Ended at cannot precede the start"),
		),
		validation.Field(&i.OdometerMeters,
			validation.Min(int64(0)).Error("Odometer cannot be negative"),
		),
		validation.Field(&i.DefectCount,
			validation.Min(0).Error("Defect count cannot be negative"),
		),
		// More unresolved defects than defects would show a tractor as worse
		// than the inspection it came from says it is.
		validation.Field(&i.UnresolvedDefectCount,
			validation.Min(0).Error("Unresolved defect count cannot be negative"),
			validation.Max(i.DefectCount).
				Error("Unresolved defects cannot exceed the total defects"),
		),
	))
}

const (
	maxInspectionTypeLength = 32
	maxSafetyStatusLength   = 16
)
