package distancecontrol

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*DistanceControl)(nil)
	_ validationframework.TenantedEntity = (*DistanceControl)(nil)
	_ domaintypes.PostgresSearchable     = (*DistanceControl)(nil)
)

const (
	PurposeLoadedMove                  = "LoadedMove"
	PurposeEmptyMove                   = "EmptyMove"
	PurposePay                         = "Pay"
	PurposeBilling                     = "Billing"
	PurposeFuel                        = "Fuel"
	PurposeEtaOutOfRoute               = "EtaOutOfRoute"
	PurposeDistanceCalculatorShortest  = "DistanceCalculatorShortest"
	PurposeDistanceCalculatorPractical = "DistanceCalculatorPractical"
)

type DistanceControl struct {
	bun.BaseModel `bun:"table:distance_controls,alias:dc" json:"-"`

	ID                                           pulid.ID `json:"id"                                          bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID                               pulid.ID `json:"businessUnitId"                              bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID                               pulid.ID `json:"organizationId"                              bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	StoreMileage                                 bool     `json:"storeMileage"                                bun:"store_mileage,type:BOOLEAN,notnull"`
	StoredDistanceUnits                          string   `json:"storedDistanceUnits"                         bun:"stored_distance_units,type:VARCHAR(50),notnull"`
	PostalCodeFallbackToCity                     bool     `json:"postalCodeFallbackToCity"                    bun:"postal_code_fallback_to_city,type:BOOLEAN,notnull"`
	AutoCreateStoredMileage                      bool     `json:"autoCreateStoredMileage"                     bun:"auto_create_stored_mileage,type:BOOLEAN,notnull"`
	LoadedMoveDistanceProfileID                  pulid.ID `json:"loadedMoveDistanceProfileId"                 bun:"loaded_move_distance_profile_id,type:VARCHAR(100),notnull"`
	EmptyMoveDistanceProfileID                   pulid.ID `json:"emptyMoveDistanceProfileId"                  bun:"empty_move_distance_profile_id,type:VARCHAR(100),notnull"`
	PayDistanceProfileID                         pulid.ID `json:"payDistanceProfileId"                        bun:"pay_distance_profile_id,type:VARCHAR(100),notnull"`
	BillingDistanceProfileID                     pulid.ID `json:"billingDistanceProfileId"                    bun:"billing_distance_profile_id,type:VARCHAR(100),notnull"`
	FuelDistanceProfileID                        pulid.ID `json:"fuelDistanceProfileId"                       bun:"fuel_distance_profile_id,type:VARCHAR(100),notnull"`
	EtaOutOfRouteDistanceProfileID               pulid.ID `json:"etaOutOfRouteDistanceProfileId"              bun:"eta_out_of_route_distance_profile_id,type:VARCHAR(100),notnull"`
	DistanceCalculatorShortestDistanceProfileID  pulid.ID `json:"distanceCalculatorShortestDistanceProfileId" bun:"distance_calculator_shortest_distance_profile_id,type:VARCHAR(100),notnull"`
	DistanceCalculatorPracticalDistanceProfileID pulid.ID `json:"distanceCalculatorPracticalDistanceProfileId" bun:"distance_calculator_practical_distance_profile_id,type:VARCHAR(100),notnull"`
	Version                                      int64    `json:"version"                                     bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                                    int64    `json:"createdAt"                                   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                                    int64    `json:"updatedAt"                                   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector                                 string   `json:"-"                                           bun:"search_vector,type:TSVECTOR,scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
}

func NewDefault(orgID, buID pulid.ID, practicalProfileID, shortestProfileID pulid.ID) *DistanceControl {
	if shortestProfileID.IsNil() {
		shortestProfileID = practicalProfileID
	}
	return &DistanceControl{
		OrganizationID:                               orgID,
		BusinessUnitID:                               buID,
		StoreMileage:                                 true,
		StoredDistanceUnits:                          distanceprofile.DefaultDistanceUnits,
		PostalCodeFallbackToCity:                     true,
		AutoCreateStoredMileage:                      true,
		LoadedMoveDistanceProfileID:                  practicalProfileID,
		EmptyMoveDistanceProfileID:                   practicalProfileID,
		PayDistanceProfileID:                         practicalProfileID,
		BillingDistanceProfileID:                     practicalProfileID,
		FuelDistanceProfileID:                        practicalProfileID,
		EtaOutOfRouteDistanceProfileID:               practicalProfileID,
		DistanceCalculatorShortestDistanceProfileID:  shortestProfileID,
		DistanceCalculatorPracticalDistanceProfileID: practicalProfileID,
	}
}

func (d *DistanceControl) ApplyDefaults() {
	d.StoredDistanceUnits = strings.TrimSpace(d.StoredDistanceUnits)
	if d.StoredDistanceUnits == "" {
		d.StoredDistanceUnits = distanceprofile.DefaultDistanceUnits
	}
}

func (d *DistanceControl) ProfileIDForPurpose(purpose string) pulid.ID {
	switch purpose {
	case PurposeEmptyMove:
		return d.EmptyMoveDistanceProfileID
	case PurposePay:
		return d.PayDistanceProfileID
	case PurposeBilling:
		return d.BillingDistanceProfileID
	case PurposeFuel:
		return d.FuelDistanceProfileID
	case PurposeEtaOutOfRoute:
		return d.EtaOutOfRouteDistanceProfileID
	case PurposeDistanceCalculatorShortest:
		return d.DistanceCalculatorShortestDistanceProfileID
	case PurposeDistanceCalculatorPractical:
		return d.DistanceCalculatorPracticalDistanceProfileID
	default:
		return d.LoadedMoveDistanceProfileID
	}
}

func (d *DistanceControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(d,
		validation.Field(&d.StoredDistanceUnits, validation.Required.Error("Stored distance units are required")),
		validation.Field(&d.LoadedMoveDistanceProfileID, validation.Required.Error("Loaded move distance profile is required")),
		validation.Field(&d.EmptyMoveDistanceProfileID, validation.Required.Error("Empty move distance profile is required")),
		validation.Field(&d.PayDistanceProfileID, validation.Required.Error("Pay distance profile is required")),
		validation.Field(&d.BillingDistanceProfileID, validation.Required.Error("Billing distance profile is required")),
		validation.Field(&d.FuelDistanceProfileID, validation.Required.Error("Fuel distance profile is required")),
		validation.Field(&d.EtaOutOfRouteDistanceProfileID, validation.Required.Error("ETA/out-of-route distance profile is required")),
		validation.Field(&d.DistanceCalculatorShortestDistanceProfileID, validation.Required.Error("Shortest calculator distance profile is required")),
		validation.Field(&d.DistanceCalculatorPracticalDistanceProfileID, validation.Required.Error("Practical calculator distance profile is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
	if d.StoredDistanceUnits != "" &&
		d.StoredDistanceUnits != distanceprofile.DefaultDistanceUnits &&
		d.StoredDistanceUnits != "Kilometers" {
		multiErr.Add("storedDistanceUnits", errortypes.ErrInvalid, "Stored distance units must be Miles or Kilometers")
	}
}

func (d *DistanceControl) GetID() pulid.ID             { return d.ID }
func (d *DistanceControl) GetTableName() string        { return "distance_controls" }
func (d *DistanceControl) GetOrganizationID() pulid.ID { return d.OrganizationID }
func (d *DistanceControl) GetBusinessUnitID() pulid.ID { return d.BusinessUnitID }

func (d *DistanceControl) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dc",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "stored_distance_units", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (d *DistanceControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("dc_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}
	return nil
}
