package dedicatedlane

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*DedicatedLaneSuggestion)(nil)
	_ domain.Validatable        = (*DedicatedLaneSuggestion)(nil)
)

type SuggestionStatus string

const (
	SuggestionStatusPending  = SuggestionStatus("Pending")
	SuggestionStatusAccepted = SuggestionStatus("Accepted")
	SuggestionStatusRejected = SuggestionStatus("Rejected")
	SuggestionStatusExpired  = SuggestionStatus("Expired")
)

//nolint:revive // This is a domain object
type DedicatedLaneSuggestion struct {
	bun.BaseModel `bun:"table:dedicated_lane_suggestions,alias:dls" json:"-"`

	ID                     pulid.ID             `json:"id"                      bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID         pulid.ID             `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID         pulid.ID             `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Status                 SuggestionStatus     `json:"status"                  bun:"status,type:suggestion_status_enum,notnull,default:'Pending'"`
	CustomerID             pulid.ID             `json:"customerId"              bun:"customer_id,type:VARCHAR(100),notnull"`
	OriginLocationID       pulid.ID             `json:"originLocationId"        bun:"origin_location_id,type:VARCHAR(100),notnull"`
	DestinationLocationID  pulid.ID             `json:"destinationLocationId"   bun:"destination_location_id,type:VARCHAR(100),notnull"`
	ServiceTypeID          *pulid.ID            `json:"serviceTypeId,omitzero"  bun:"service_type_id,type:VARCHAR(100),nullzero"`
	ShipmentTypeID         *pulid.ID            `json:"shipmentTypeId,omitzero" bun:"shipment_type_id,type:VARCHAR(100),nullzero"`
	TrailerTypeID          *pulid.ID            `json:"trailerTypeId,omitzero"  bun:"trailer_type_id,type:VARCHAR(100),nullzero"`
	TractorTypeID          *pulid.ID            `json:"tractorTypeId,omitzero"  bun:"tractor_type_id,type:VARCHAR(100),nullzero"`
	CreatedDedicatedLaneID *pulid.ID            `json:"createdDedicatedLaneId"  bun:"created_dedicated_lane_id,type:VARCHAR(100),nullzero"`
	ProcessedByID          *pulid.ID            `json:"processedById"           bun:"processed_by_id,type:VARCHAR(100),nullzero"`
	AverageFreightCharge   *decimal.NullDecimal `json:"averageFreightCharge"    bun:"average_freight_charge,type:NUMERIC(19,4),nullzero"`
	TotalFreightValue      *decimal.NullDecimal `json:"totalFreightValue"       bun:"total_freight_value,type:NUMERIC(19,4),nullzero"`
	ProcessedAt            *int64               `json:"processedAt"             bun:"processed_at,type:BIGINT,nullzero"`
	ConfidenceScore        decimal.Decimal      `json:"confidenceScore"         bun:"confidence_score,type:NUMERIC(5,4),notnull"`
	SuggestedName          string               `json:"suggestedName"           bun:"suggested_name,type:VARCHAR(200),notnull"`
	PatternDetails         map[string]any       `json:"patternDetails"          bun:"pattern_details,type:JSONB,notnull"`
	LastShipmentDate       int64                `json:"lastShipmentDate"        bun:"last_shipment_date,type:BIGINT,notnull"`
	FrequencyCount         int64                `json:"frequencyCount"          bun:"frequency_count,type:INTEGER,notnull"`
	FirstShipmentDate      int64                `json:"firstShipmentDate"       bun:"first_shipment_date,type:BIGINT,notnull"`
	AnalysisStartDate      int64                `json:"analysisStartDate"       bun:"analysis_start_date,type:BIGINT,notnull"`
	AnalysisEndDate        int64                `json:"analysisEndDate"         bun:"analysis_end_date,type:BIGINT,notnull"`
	ExpiresAt              int64                `json:"expiresAt"               bun:"expires_at,type:BIGINT,notnull"`
	Version                int64                `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt              int64                `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit         *businessunit.BusinessUnit   `json:"businessUnit,omitzero"         bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization         *organization.Organization   `json:"organization,omitzero"         bun:"rel:belongs-to,join:organization_id=id"`
	Customer             *customer.Customer           `json:"customer,omitzero"             bun:"rel:belongs-to,join:customer_id=id"`
	OriginLocation       *location.Location           `json:"originLocation,omitzero"       bun:"rel:belongs-to,join:origin_location_id=id"`
	DestinationLocation  *location.Location           `json:"destinationLocation,omitzero"  bun:"rel:belongs-to,join:destination_location_id=id"`
	ServiceType          *servicetype.ServiceType     `json:"serviceType,omitzero"          bun:"rel:belongs-to,join:service_type_id=id"`
	ShipmentType         *shipmenttype.ShipmentType   `json:"shipmentType,omitzero"         bun:"rel:belongs-to,join:shipment_type_id=id"`
	TractorType          *equipmenttype.EquipmentType `json:"tractorType,omitzero"          bun:"rel:belongs-to,join:tractor_type_id=id"`
	TrailerType          *equipmenttype.EquipmentType `json:"trailerType,omitzero"          bun:"rel:belongs-to,join:trailer_type_id=id"`
	ProcessedBy          *user.User                   `json:"processedBy,omitzero"          bun:"rel:belongs-to,join:processed_by_id=id"`
	CreatedDedicatedLane *DedicatedLane               `json:"createdDedicatedLane,omitzero" bun:"rel:belongs-to,join:created_dedicated_lane_id=id"`
}

func (dls *DedicatedLaneSuggestion) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(
		ctx,
		dls,
		validation.Field(
			&dls.CustomerID,
			validation.Required.Error("Customer is required"),
		),
		validation.Field(
			&dls.OriginLocationID,
			validation.Required.Error("Origin Location is required"),
		),
		validation.Field(
			&dls.DestinationLocationID,
			validation.Required.Error("Destination Location is required"),
			validation.When(
				pulid.Equals(dls.OriginLocationID, dls.DestinationLocationID),
				validation.Required.Error("Origin and Destination cannot be the same location"),
			),
		),
		validation.Field(
			&dls.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				SuggestionStatusPending,
				SuggestionStatusAccepted,
				SuggestionStatusRejected,
				SuggestionStatusExpired,
			).Error("Status must be a valid suggestion status"),
		),
		validation.Field(
			&dls.ConfidenceScore,
			validation.Required.Error("Confidence Score is required"),
			validation.Min(decimal.NewFromFloat(0.0)).Error("Confidence Score must be >= 0"),
			validation.Max(decimal.NewFromFloat(1.0)).Error("Confidence Score must be <= 1"),
		),
		validation.Field(
			&dls.FrequencyCount,
			validation.Required.Error("Frequency Count is required"),
			validation.Min(int64(1)).Error("Frequency Count must be >= 1"),
		),
		validation.Field(
			&dls.SuggestedName,
			validation.Required.Error("Suggested Name is required"),
			validation.Length(2, 200).Error("Suggested Name must be between 2 & 200 characters"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (dls *DedicatedLaneSuggestion) GetID() string {
	return dls.ID.String()
}

func (dls *DedicatedLaneSuggestion) GetTableName() string {
	return "dedicated_lane_suggestions"
}

func (dls *DedicatedLaneSuggestion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dls.ID.IsNil() {
			dls.ID = pulid.MustNew("dls_")
		}

		dls.CreatedAt = now
	case *bun.UpdateQuery:
		dls.UpdatedAt = now
	}

	return nil
}

func (dls *DedicatedLaneSuggestion) IsExpired() bool {
	return timeutils.NowUnix() > dls.ExpiresAt
}

func (dls *DedicatedLaneSuggestion) IsProcessed() bool {
	return dls.Status == SuggestionStatusAccepted || dls.Status == SuggestionStatusRejected
}
