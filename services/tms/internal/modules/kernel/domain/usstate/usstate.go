package usstate

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Region string

const (
	RegionNortheast = Region("Northeast")
	RegionMidwest   = Region("Midwest")
	RegionSouth     = Region("South")
	RegionWest      = Region("West")
)

type UsState struct {
	keel.Table `keel:"us_states"`

	ID           pulid.ID `json:"id" keel:"id,pk,autoid:us_"`
	Name         string   `json:"name" keel:"name"`
	Abbreviation string   `json:"abbreviation" keel:"abbreviation"`
	CountryName  string   `json:"countryName" keel:"country_name"`
	CountryIso3  string   `json:"countryIso3" keel:"country_iso3"`
	CreatedAt    int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func RegionForStateAbbreviation(abbreviation string) (Region, bool) {
	region, ok := stateAbbreviationRegions[abbreviation]
	return region, ok
}

func (us *UsState) BeforeCreate(ctx context.Context) error {
	if us.CountryIso3 == "" {
		us.CountryIso3 = "USA"
	}

	return nil
}

var stateAbbreviationRegions = map[string]Region{
	"CT": RegionNortheast,
	"ME": RegionNortheast,
	"MA": RegionNortheast,
	"NH": RegionNortheast,
	"RI": RegionNortheast,
	"VT": RegionNortheast,
	"NJ": RegionNortheast,
	"NY": RegionNortheast,
	"PA": RegionNortheast,
	"IL": RegionMidwest,
	"IN": RegionMidwest,
	"MI": RegionMidwest,
	"OH": RegionMidwest,
	"WI": RegionMidwest,
	"IA": RegionMidwest,
	"KS": RegionMidwest,
	"MN": RegionMidwest,
	"MO": RegionMidwest,
	"NE": RegionMidwest,
	"ND": RegionMidwest,
	"SD": RegionMidwest,
	"DE": RegionSouth,
	"DC": RegionSouth,
	"FL": RegionSouth,
	"GA": RegionSouth,
	"MD": RegionSouth,
	"NC": RegionSouth,
	"SC": RegionSouth,
	"VA": RegionSouth,
	"WV": RegionSouth,
	"AL": RegionSouth,
	"KY": RegionSouth,
	"MS": RegionSouth,
	"TN": RegionSouth,
	"AR": RegionSouth,
	"LA": RegionSouth,
	"OK": RegionSouth,
	"TX": RegionSouth,
	"AZ": RegionWest,
	"CO": RegionWest,
	"ID": RegionWest,
	"MT": RegionWest,
	"NV": RegionWest,
	"NM": RegionWest,
	"UT": RegionWest,
	"WY": RegionWest,
	"AK": RegionWest,
	"CA": RegionWest,
	"HI": RegionWest,
	"OR": RegionWest,
	"WA": RegionWest,
}

func (s *UsState) GetID() pulid.ID { return s.ID }

func (s *UsState) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.Name, validation.Required.Error("Name is required")),
		// Addresses are stored and printed with the two letter abbreviation, so
		// anything else here would produce an undeliverable label.
		validation.Field(&s.Abbreviation,
			validation.Required.Error("Abbreviation is required"),
			validation.Length(stateAbbreviationLength, stateAbbreviationLength).
				Error("Abbreviation must be two characters"),
		),
		validation.Field(&s.CountryName, validation.Required.Error("Country name is required")),
		validation.Field(&s.CountryIso3,
			validation.Required.Error("Country ISO3 is required"),
			validation.Length(countryISO3Length, countryISO3Length).
				Error("Country ISO3 must be three characters"),
		),
	))
}

const (
	stateAbbreviationLength = 2
	countryISO3Length       = 3
)
