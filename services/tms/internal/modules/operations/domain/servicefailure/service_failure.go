package servicefailure

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ServiceFailure)(nil)
	_ validationframework.TenantedEntity = (*ServiceFailure)(nil)
	_ domaintypes.PostgresSearchable     = (*ServiceFailure)(nil)
)

type ServiceFailure struct {
	bun.BaseModel             `bun:"table:service_failures,alias:sf" json:"-"`
	keel.Table                `keel:"service_failures"`
	pagination.CursorValueSet `bun:",embed"                          json:"-"`

	ID                    pulid.ID          `json:"id"                    bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk"`
	BusinessUnitID        pulid.ID          `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID        pulid.ID          `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	ShipmentID            pulid.ID          `json:"shipmentId"            bun:"shipment_id,type:VARCHAR(100),notnull" keel:"shipment_id"`
	ShipmentMoveID        pulid.ID          `json:"shipmentMoveId"        bun:"shipment_move_id,type:VARCHAR(100),notnull" keel:"shipment_move_id"`
	StopID                pulid.ID          `json:"stopId"                bun:"stop_id,type:VARCHAR(100),notnull" keel:"stop_id"`
	ReasonCodeID          *pulid.ID         `json:"reasonCodeId"          bun:"reason_code_id,type:VARCHAR(100),nullzero" keel:"reason_code_id"`
	Number                string            `json:"number"                bun:"number,type:VARCHAR(64),notnull" keel:"number"`
	Type                  Type              `json:"type"                  bun:"type,type:service_failure_type_enum,notnull" keel:"type"`
	Source                Source            `json:"source"                bun:"source,type:service_failure_source_enum,notnull,default:'Detected'" keel:"source"`
	Status                Status            `json:"status"                bun:"status,type:service_failure_status_enum,notnull,default:'Open'" keel:"status"`
	StopType              shipment.StopType `json:"stopType"              bun:"stop_type,type:stop_type_enum,notnull" keel:"stop_type"`
	ScheduledCutoff       int64             `json:"scheduledCutoff"       bun:"scheduled_cutoff,type:BIGINT,notnull" keel:"scheduled_cutoff"`
	ActualArrival         int64             `json:"actualArrival"         bun:"actual_arrival,type:BIGINT,notnull" keel:"actual_arrival"`
	GracePeriodMinutes    int               `json:"gracePeriodMinutes"    bun:"grace_period_minutes,type:INTEGER,notnull,default:30" keel:"grace_period_minutes"`
	LateMinutes           int64             `json:"lateMinutes"           bun:"late_minutes,type:BIGINT,notnull" keel:"late_minutes"`
	Notes                 string            `json:"notes"                 bun:"notes,type:TEXT,nullzero" keel:"notes,nullzero"`
	InternalNotes         string            `json:"internalNotes"         bun:"internal_notes,type:TEXT,nullzero" keel:"internal_notes,nullzero"`
	X12StatusCodeOverride string            `json:"x12StatusCodeOverride" bun:"x12_status_code_override,type:VARCHAR(3),nullzero" keel:"x12_status_code_override,nullzero"`
	X12ReasonCodeOverride string            `json:"x12ReasonCodeOverride" bun:"x12_reason_code_override,type:VARCHAR(3),nullzero" keel:"x12_reason_code_override,nullzero"`
	X12ExceptionCode      string            `json:"x12ExceptionCode"      bun:"x12_exception_code,type:VARCHAR(3),nullzero" keel:"x12_exception_code,nullzero"`
	DetectedAt            int64             `json:"detectedAt"            bun:"detected_at,type:BIGINT,notnull" keel:"detected_at"`
	ReviewedAt            *int64            `json:"reviewedAt"            bun:"reviewed_at,type:BIGINT,nullzero" keel:"reviewed_at"`
	ReviewedByID          *pulid.ID         `json:"reviewedById"          bun:"reviewed_by_id,type:VARCHAR(100),nullzero" keel:"reviewed_by_id"`
	ResolvedAt            *int64            `json:"resolvedAt"            bun:"resolved_at,type:BIGINT,nullzero" keel:"resolved_at"`
	ResolvedByID          *pulid.ID         `json:"resolvedById"          bun:"resolved_by_id,type:VARCHAR(100),nullzero" keel:"resolved_by_id"`
	VoidedAt              *int64            `json:"voidedAt"              bun:"voided_at,type:BIGINT,nullzero" keel:"voided_at"`
	VoidedByID            *pulid.ID         `json:"voidedById"            bun:"voided_by_id,type:VARCHAR(100),nullzero" keel:"voided_by_id"`
	VoidReason            string            `json:"voidReason"            bun:"void_reason,type:TEXT,nullzero" keel:"void_reason,nullzero"`
	CreatedByID           *pulid.ID         `json:"createdById"           bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id"`
	Version               int64             `json:"version"               bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt             int64             `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt             int64             `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	SearchVector          string            `json:"-"                     bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`

	ReasonCode   *ReasonCode            `json:"reasonCode,omitempty"   bun:"rel:belongs-to,join:reason_code_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:reason_code_id"`
	Shipment     *shipment.Shipment     `json:"shipment,omitempty"     bun:"rel:belongs-to,join:shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:shipment_id"`
	ShipmentMove *shipment.ShipmentMove `json:"shipmentMove,omitempty" bun:"rel:belongs-to,join:shipment_move_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:shipment_move_id"`
	Stop         *shipment.Stop         `json:"stop,omitempty"         bun:"rel:belongs-to,join:stop_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:stop_id"`
	CreatedBy    *tenant.User           `json:"createdBy,omitempty"    bun:"rel:belongs-to,join:created_by_id=id" keel:"belongs:created_by_id"`
	ReviewedBy   *tenant.User           `json:"reviewedBy,omitempty"   bun:"rel:belongs-to,join:reviewed_by_id=id" keel:"belongs:reviewed_by_id"`
	ResolvedBy   *tenant.User           `json:"resolvedBy,omitempty"   bun:"rel:belongs-to,join:resolved_by_id=id" keel:"belongs:resolved_by_id"`
	VoidedBy     *tenant.User           `json:"voidedBy,omitempty"     bun:"rel:belongs-to,join:voided_by_id=id" keel:"belongs:voided_by_id"`
	BusinessUnit *tenant.BusinessUnit   `json:"-"                      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization   `json:"-"                      bun:"rel:belongs-to,join:organization_id=id"`
}

func (sf *ServiceFailure) Normalize() {
	sf.ensureIdentity()
	sf.Number = strings.TrimSpace(sf.Number)
	sf.Notes = strings.TrimSpace(sf.Notes)
	sf.InternalNotes = strings.TrimSpace(sf.InternalNotes)
	sf.X12StatusCodeOverride = strings.ToUpper(strings.TrimSpace(sf.X12StatusCodeOverride))
	sf.X12ReasonCodeOverride = strings.ToUpper(strings.TrimSpace(sf.X12ReasonCodeOverride))
	sf.X12ExceptionCode = strings.ToUpper(strings.TrimSpace(sf.X12ExceptionCode))
	sf.VoidReason = strings.TrimSpace(sf.VoidReason)
}

func (sf *ServiceFailure) Validate(multiErr *errortypes.MultiError) {
	sf.Normalize()
	multiErr.AddOzzoError(validation.ValidateStruct(
		sf,
		validation.Field(
			&sf.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&sf.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(&sf.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(
			&sf.ShipmentMoveID,
			validation.Required.Error("Shipment move ID is required"),
		),
		validation.Field(&sf.StopID, validation.Required.Error("Stop ID is required")),
		validation.Field(
			&sf.Number,
			validation.Required.Error("Service failure number is required"),
		),
		validation.Field(&sf.Type,
			validation.Required.Error("Service failure type is required"),
			domainvalidation.ValidEnum[Type]("service failure type is invalid"),
		),
		validation.Field(&sf.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[Source]("source is invalid"),
		),
		validation.Field(&sf.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("status is invalid"),
		),
		validation.Field(&sf.StopType,
			validation.Required.Error("Stop type is required"),
			validation.In(
				shipment.StopTypePickup,
				shipment.StopTypeDelivery,
				shipment.StopTypeSplitPickup,
				shipment.StopTypeSplitDelivery,
			).Error("Stop type is invalid"),
		),
		validation.Field(&sf.ScheduledCutoff,
			validation.Required.Error("Scheduled cutoff is required"),
			validation.Min(int64(1)).Error("Scheduled cutoff must be greater than zero"),
		),
		validation.Field(&sf.ActualArrival,
			validation.Required.Error("Actual arrival is required"),
			validation.Min(int64(1)).Error("Actual arrival must be greater than zero"),
		),
		validation.Field(&sf.GracePeriodMinutes,
			validation.Required.Error("Grace period is required"),
			validation.Min(1).Error("Grace period must be greater than zero"),
		),
		validation.Field(&sf.LateMinutes,
			validation.Required.Error("Late minutes is required"),
			validation.Min(int64(1)).Error("Late minutes must be at least 1"),
		),
		validation.Field(&sf.X12StatusCodeOverride,
			validation.Length(0, 3).Error("X12 status code override must be at most 3 characters"),
		),
		validation.Field(&sf.X12ReasonCodeOverride,
			validation.Length(0, 3).Error("X12 reason code override must be at most 3 characters"),
		),
		validation.Field(&sf.X12ExceptionCode,
			validation.Length(0, 3).Error("X12 exception code must be at most 3 characters"),
		),
	))
}

func (sf *ServiceFailure) IsUnresolved() bool {
	return sf != nil && (sf.Status == StatusOpen || sf.Status == StatusReviewed)
}

func (sf *ServiceFailure) IsTerminal() bool {
	return sf != nil && (sf.Status == StatusResolved || sf.Status == StatusVoided)
}

func (sf *ServiceFailure) GetID() pulid.ID {
	return sf.ID
}

func (sf *ServiceFailure) GetCreatedAt() int64 {
	return sf.CreatedAt
}

func (sf *ServiceFailure) GetTableName() string {
	return "service_failures"
}

func (sf *ServiceFailure) GetOrganizationID() pulid.ID {
	return sf.OrganizationID
}

func (sf *ServiceFailure) GetBusinessUnitID() pulid.ID {
	return sf.BusinessUnitID
}

func (sf *ServiceFailure) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "sf",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "number", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "type", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "source", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "status", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "notes", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
			{
				Name:   "internal_notes",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (sf *ServiceFailure) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	sf.Normalize()

	switch query.(type) {
	case *bun.InsertQuery:
		sf.ensureIdentity()
		if sf.Source == "" {
			sf.Source = SourceDetected
		}
		if sf.Status == "" {
			sf.Status = StatusOpen
		}
		if sf.DetectedAt == 0 {
			sf.DetectedAt = now
		}
		sf.CreatedAt = now
	case *bun.UpdateQuery:
		sf.UpdatedAt = now
	}

	return nil
}

func (sf *ServiceFailure) ensureIdentity() {
	if sf.ID.IsNil() {
		sf.ID = pulid.MustNew("sf_")
	}
	if sf.Number == "" {
		sf.Number = serviceFailureNumber(sf.ID)
	}
}

func serviceFailureNumber(id pulid.ID) string {
	value := strings.TrimPrefix(id.String(), "sf_")
	if len(value) > 12 {
		value = value[len(value)-12:]
	}
	return "SF-" + strings.ToUpper(value)
}
