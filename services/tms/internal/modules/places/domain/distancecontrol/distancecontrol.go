package distancecontrol

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.Entity     = (*DistanceControl)(nil)
	_ domaintypes.PostgresSearchable = (*DistanceControl)(nil)
)

const (
	PurposeLoadedMove    = "LoadedMove"
	PurposeEmptyMove     = "EmptyMove"
	PurposePay           = "Pay"
	PurposeBilling       = "Billing"
	PurposeFuel          = "Fuel"
	PurposeEtaOutOfRoute = "EtaOutOfRoute"

	RatingDistanceUnitKilometers = "Kilometers"
)

type DistanceControl struct {
	keel.Table `keel:"distance_controls"`

	ID                             pulid.ID `json:"id"                                           keel:"id,pk,autoid:dc_"`
	BusinessUnitID                 pulid.ID `json:"businessUnitId"                               keel:"business_unit_id,pk,tenant"`
	OrganizationID                 pulid.ID `json:"organizationId"                               keel:"organization_id,pk,tenant"`
	StoreMileage                   bool     `json:"storeMileage"                                 keel:"store_mileage"`
	RatingDistanceUnit             string   `json:"ratingDistanceUnit"                           keel:"rating_distance_unit"`
	PostalCodeFallbackToCity       bool     `json:"postalCodeFallbackToCity"                     keel:"postal_code_fallback_to_city"`
	AutoCreateStoredMileage        bool     `json:"autoCreateStoredMileage"                      keel:"auto_create_stored_mileage"`
	LoadedMoveDistanceProfileID    pulid.ID `json:"loadedMoveDistanceProfileId"                  keel:"loaded_move_distance_profile_id"`
	EmptyMoveDistanceProfileID     pulid.ID `json:"emptyMoveDistanceProfileId"                   keel:"empty_move_distance_profile_id"`
	PayDistanceProfileID           pulid.ID `json:"payDistanceProfileId"                         keel:"pay_distance_profile_id"`
	BillingDistanceProfileID       pulid.ID `json:"billingDistanceProfileId"                     keel:"billing_distance_profile_id"`
	FuelDistanceProfileID          pulid.ID `json:"fuelDistanceProfileId"                        keel:"fuel_distance_profile_id"`
	EtaOutOfRouteDistanceProfileID pulid.ID `json:"etaOutOfRouteDistanceProfileId"               keel:"eta_out_of_route_distance_profile_id"`
	CaptureJurisdictionMiles       bool     `json:"captureJurisdictionMiles"                     keel:"capture_jurisdiction_miles"`
	Version                        int64    `json:"version"                                      keel:"version,version"`
	CreatedAt                      int64    `json:"createdAt"                                    keel:"created_at,autocreate"`
	UpdatedAt                      int64    `json:"updatedAt"                                    keel:"updated_at,autoupdate"`
	SearchVector                   string   `json:"-"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func NewDefault(
	orgID, buID pulid.ID,
	profileID pulid.ID,
) *DistanceControl {
	return &DistanceControl{
		OrganizationID:                 orgID,
		BusinessUnitID:                 buID,
		StoreMileage:                   true,
		RatingDistanceUnit:             distanceprofile.DefaultDistanceUnits,
		PostalCodeFallbackToCity:       true,
		AutoCreateStoredMileage:        true,
		LoadedMoveDistanceProfileID:    profileID,
		EmptyMoveDistanceProfileID:     profileID,
		PayDistanceProfileID:           profileID,
		BillingDistanceProfileID:       profileID,
		FuelDistanceProfileID:          profileID,
		EtaOutOfRouteDistanceProfileID: profileID,
	}
}

func (d *DistanceControl) ApplyDefaults() {
	d.RatingDistanceUnit = strings.TrimSpace(d.RatingDistanceUnit)
	if d.RatingDistanceUnit == "" {
		d.RatingDistanceUnit = distanceprofile.DefaultDistanceUnits
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
	default:
		return d.LoadedMoveDistanceProfileID
	}
}

func (d *DistanceControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		d,
		validation.Field(
			&d.RatingDistanceUnit,
			validation.Required.Error("Rating distance unit is required"),
		),
		validation.Field(
			&d.LoadedMoveDistanceProfileID,
			validation.Required.Error("Loaded move distance profile is required"),
		),
		validation.Field(
			&d.EmptyMoveDistanceProfileID,
			validation.Required.Error("Empty move distance profile is required"),
		),
		validation.Field(
			&d.PayDistanceProfileID,
			validation.Required.Error("Pay distance profile is required"),
		),
		validation.Field(
			&d.BillingDistanceProfileID,
			validation.Required.Error("Billing distance profile is required"),
		),
		validation.Field(
			&d.FuelDistanceProfileID,
			validation.Required.Error("Fuel distance profile is required"),
		),
		validation.Field(
			&d.EtaOutOfRouteDistanceProfileID,
			validation.Required.Error("ETA/out-of-route distance profile is required"),
		),
	))
	if d.RatingDistanceUnit != "" &&
		d.RatingDistanceUnit != distanceprofile.DefaultDistanceUnits &&
		d.RatingDistanceUnit != RatingDistanceUnitKilometers {
		multiErr.Add(
			"ratingDistanceUnit",
			errortypes.ErrInvalid,
			"Rating distance unit must be Miles or Kilometers",
		)
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
			{
				Name:   "rating_distance_unit",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}
