package servicefailure

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                    pulid.ID          `json:"id"                    bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID        pulid.ID          `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID        pulid.ID          `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentID            pulid.ID          `json:"shipmentId"            bun:"shipment_id,type:VARCHAR(100),notnull"`
	ShipmentMoveID        pulid.ID          `json:"shipmentMoveId"        bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	StopID                pulid.ID          `json:"stopId"                bun:"stop_id,type:VARCHAR(100),notnull"`
	ReasonCodeID          *pulid.ID         `json:"reasonCodeId"          bun:"reason_code_id,type:VARCHAR(100),nullzero"`
	Number                string            `json:"number"                bun:"number,type:VARCHAR(64),notnull"`
	Type                  Type              `json:"type"                  bun:"type,type:service_failure_type_enum,notnull"`
	Source                Source            `json:"source"                bun:"source,type:service_failure_source_enum,notnull,default:'Detected'"`
	Status                Status            `json:"status"                bun:"status,type:service_failure_status_enum,notnull,default:'Open'"`
	StopType              shipment.StopType `json:"stopType"              bun:"stop_type,type:stop_type_enum,notnull"`
	ScheduledCutoff       int64             `json:"scheduledCutoff"       bun:"scheduled_cutoff,type:BIGINT,notnull"`
	ActualArrival         int64             `json:"actualArrival"         bun:"actual_arrival,type:BIGINT,notnull"`
	GracePeriodMinutes    int               `json:"gracePeriodMinutes"    bun:"grace_period_minutes,type:INTEGER,notnull,default:30"`
	LateMinutes           int64             `json:"lateMinutes"           bun:"late_minutes,type:BIGINT,notnull"`
	Notes                 string            `json:"notes"                 bun:"notes,type:TEXT,nullzero"`
	InternalNotes         string            `json:"internalNotes"         bun:"internal_notes,type:TEXT,nullzero"`
	X12StatusCodeOverride string            `json:"x12StatusCodeOverride" bun:"x12_status_code_override,type:VARCHAR(3),nullzero"`
	X12ReasonCodeOverride string            `json:"x12ReasonCodeOverride" bun:"x12_reason_code_override,type:VARCHAR(3),nullzero"`
	X12ExceptionCode      string            `json:"x12ExceptionCode"      bun:"x12_exception_code,type:VARCHAR(3),nullzero"`
	DetectedAt            int64             `json:"detectedAt"            bun:"detected_at,type:BIGINT,notnull"`
	ReviewedAt            *int64            `json:"reviewedAt"            bun:"reviewed_at,type:BIGINT,nullzero"`
	ReviewedByID          *pulid.ID         `json:"reviewedById"          bun:"reviewed_by_id,type:VARCHAR(100),nullzero"`
	ResolvedAt            *int64            `json:"resolvedAt"            bun:"resolved_at,type:BIGINT,nullzero"`
	ResolvedByID          *pulid.ID         `json:"resolvedById"          bun:"resolved_by_id,type:VARCHAR(100),nullzero"`
	VoidedAt              *int64            `json:"voidedAt"              bun:"voided_at,type:BIGINT,nullzero"`
	VoidedByID            *pulid.ID         `json:"voidedById"            bun:"voided_by_id,type:VARCHAR(100),nullzero"`
	VoidReason            string            `json:"voidReason"            bun:"void_reason,type:TEXT,nullzero"`
	CreatedByID           *pulid.ID         `json:"createdById"           bun:"created_by_id,type:VARCHAR(100),nullzero"`
	Version               int64             `json:"version"               bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt             int64             `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64             `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector          string            `json:"-"                     bun:"search_vector,type:TSVECTOR,scanonly"`

	ReasonCode   *ReasonCode            `json:"reasonCode,omitempty" bun:"rel:belongs-to,join:reason_code_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Shipment     *shipment.Shipment     `json:"shipment,omitempty"   bun:"rel:belongs-to,join:shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	ShipmentMove *shipment.ShipmentMove `json:"shipmentMove,omitempty" bun:"rel:belongs-to,join:shipment_move_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Stop         *shipment.Stop         `json:"stop,omitempty"       bun:"rel:belongs-to,join:stop_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	CreatedBy    *tenant.User           `json:"createdBy,omitempty"  bun:"rel:belongs-to,join:created_by_id=id"`
	ReviewedBy   *tenant.User           `json:"reviewedBy,omitempty" bun:"rel:belongs-to,join:reviewed_by_id=id"`
	ResolvedBy   *tenant.User           `json:"resolvedBy,omitempty" bun:"rel:belongs-to,join:resolved_by_id=id"`
	VoidedBy     *tenant.User           `json:"voidedBy,omitempty"   bun:"rel:belongs-to,join:voided_by_id=id"`
	BusinessUnit *tenant.BusinessUnit   `json:"-"                    bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization   `json:"-"                    bun:"rel:belongs-to,join:organization_id=id"`
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
	err := validation.ValidateStruct(sf,
		validation.Field(&sf.OrganizationID, validation.Required.Error("Organization ID is required")),
		validation.Field(&sf.BusinessUnitID, validation.Required.Error("Business unit ID is required")),
		validation.Field(&sf.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&sf.ShipmentMoveID, validation.Required.Error("Shipment move ID is required")),
		validation.Field(&sf.StopID, validation.Required.Error("Stop ID is required")),
		validation.Field(&sf.Number, validation.Required.Error("Service failure number is required")),
		validation.Field(&sf.Type,
			validation.Required.Error("Service failure type is required"),
			validation.By(func(value any) error {
				failureType, _ := value.(Type)
				if !failureType.IsValid() {
					return errors.New("service failure type is invalid")
				}
				return nil
			}),
		),
		validation.Field(&sf.Source,
			validation.Required.Error("Source is required"),
			validation.By(func(value any) error {
				source, _ := value.(Source)
				if !source.IsValid() {
					return errors.New("source is invalid")
				}
				return nil
			}),
		),
		validation.Field(&sf.Status,
			validation.Required.Error("Status is required"),
			validation.By(func(value any) error {
				status, _ := value.(Status)
				if !status.IsValid() {
					return errors.New("status is invalid")
				}
				return nil
			}),
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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
			{Name: "internal_notes", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
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
