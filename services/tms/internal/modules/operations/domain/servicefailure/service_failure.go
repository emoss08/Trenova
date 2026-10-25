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
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*ServiceFailure)(nil)
	_ domaintypes.PostgresSearchable     = (*ServiceFailure)(nil)
)

type ServiceFailure struct {
	keel.Table `keel:"service_failures"`

	ID                    pulid.ID          `json:"id"                    keel:"id,pk,autoid:sf_"`
	BusinessUnitID        pulid.ID          `json:"businessUnitId"        keel:"business_unit_id,pk,tenant"`
	OrganizationID        pulid.ID          `json:"organizationId"        keel:"organization_id,pk,tenant"`
	ShipmentID            pulid.ID          `json:"shipmentId"            keel:"shipment_id"`
	ShipmentMoveID        pulid.ID          `json:"shipmentMoveId"        keel:"shipment_move_id"`
	StopID                pulid.ID          `json:"stopId"                keel:"stop_id"`
	ReasonCodeID          *pulid.ID         `json:"reasonCodeId"          keel:"reason_code_id"`
	Number                string            `json:"number"                keel:"number"`
	Type                  Type              `json:"type"                  keel:"type"`
	Source                Source            `json:"source"                keel:"source"`
	Status                Status            `json:"status"                keel:"status"`
	StopType              shipment.StopType `json:"stopType"              keel:"stop_type"`
	ScheduledCutoff       int64             `json:"scheduledCutoff"       keel:"scheduled_cutoff"`
	ActualArrival         int64             `json:"actualArrival"         keel:"actual_arrival"`
	GracePeriodMinutes    int               `json:"gracePeriodMinutes"    keel:"grace_period_minutes"`
	LateMinutes           int64             `json:"lateMinutes"           keel:"late_minutes"`
	Notes                 string            `json:"notes"                 keel:"notes,nullzero"`
	InternalNotes         string            `json:"internalNotes"         keel:"internal_notes,nullzero"`
	X12StatusCodeOverride string            `json:"x12StatusCodeOverride" keel:"x12_status_code_override,nullzero"`
	X12ReasonCodeOverride string            `json:"x12ReasonCodeOverride" keel:"x12_reason_code_override,nullzero"`
	X12ExceptionCode      string            `json:"x12ExceptionCode"      keel:"x12_exception_code,nullzero"`
	DetectedAt            int64             `json:"detectedAt"            keel:"detected_at"`
	ReviewedAt            *int64            `json:"reviewedAt"            keel:"reviewed_at"`
	ReviewedByID          *pulid.ID         `json:"reviewedById"          keel:"reviewed_by_id"`
	ResolvedAt            *int64            `json:"resolvedAt"            keel:"resolved_at"`
	ResolvedByID          *pulid.ID         `json:"resolvedById"          keel:"resolved_by_id"`
	VoidedAt              *int64            `json:"voidedAt"              keel:"voided_at"`
	VoidedByID            *pulid.ID         `json:"voidedById"            keel:"voided_by_id"`
	VoidReason            string            `json:"voidReason"            keel:"void_reason,nullzero"`
	CreatedByID           *pulid.ID         `json:"createdById"           keel:"created_by_id"`
	Version               int64             `json:"version"               keel:"version,version"`
	CreatedAt             int64             `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64             `json:"updatedAt"             keel:"updated_at,autoupdate"`
	SearchVector          string            `json:"-"                     keel:"search_vector,tsvector,readonly,noselect"`

	ReasonCode   *ReasonCode            `json:"reasonCode,omitempty"   keel:"belongs:reason_code_id"`
	Shipment     *shipment.Shipment     `json:"shipment,omitempty"     keel:"belongs:shipment_id"`
	ShipmentMove *shipment.ShipmentMove `json:"shipmentMove,omitempty" keel:"belongs:shipment_move_id"`
	Stop         *shipment.Stop         `json:"stop,omitempty"         keel:"belongs:stop_id"`
	CreatedBy    *tenant.User           `json:"createdBy,omitempty"    keel:"belongs:created_by_id"`
	ReviewedBy   *tenant.User           `json:"reviewedBy,omitempty"   keel:"belongs:reviewed_by_id"`
	ResolvedBy   *tenant.User           `json:"resolvedBy,omitempty"   keel:"belongs:resolved_by_id"`
	VoidedBy     *tenant.User           `json:"voidedBy,omitempty"     keel:"belongs:voided_by_id"`
	BusinessUnit *tenant.BusinessUnit   `json:"-"`
	Organization *tenant.Organization   `json:"-"`
}

func (sf *ServiceFailure) BeforeUpdate(_ context.Context) error {
	sf.Normalize()

	return nil
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

func (sf *ServiceFailure) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	sf.Normalize()
	sf.applyCreateDefaults(now)
	if sf.Source == "" {
		sf.Source = SourceDetected
	}
	if sf.Status == "" {
		sf.Status = StatusOpen
	}
	if sf.GracePeriodMinutes == 0 {
		sf.GracePeriodMinutes = 30
	}
	sf.Normalize()
	sf.applyCreateDefaults(timeutils.NowUnix())

	return nil
}

func (sf *ServiceFailure) applyCreateDefaults(now int64) {
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
}

func (sf *ServiceFailure) ensureIdentity() {
	if sf.ID.IsNil() {
		sf.ID = pulid.MustNew("sf_")
	}
	if sf.Number == "" {
		sf.Number = serviceFailureNumber(sf.ID)
	}
}

func (ServiceFailure) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_sf_reason_code",
			Message: "Reason code does not exist in your organization",
		},
		{
			Name:    "ux_sf_active_stop_type",
			Message: "An unresolved service failure already exists for this stop",
		},
		{
			Name:    "ux_sf_tenant_number",
			Message: "Service failure number already exists",
		},
	}
}

func serviceFailureNumber(id pulid.ID) string {
	value := strings.TrimPrefix(id.String(), "sf_")
	if len(value) > 12 {
		value = value[len(value)-12:]
	}
	return "SF-" + strings.ToUpper(value)
}
