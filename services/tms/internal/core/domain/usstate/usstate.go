package usstate

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*UsState)(nil)

type Region string

const (
	RegionNortheast = Region("Northeast")
	RegionMidwest   = Region("Midwest")
	RegionSouth     = Region("South")
	RegionWest      = Region("West")
)

type UsState struct {
	bun.BaseModel `json:"-" bun:"table:us_states,alias:ust"`

	ID           pulid.ID `json:"id"           bun:"id,pk,type:VARCHAR(100)"`
	Name         string   `json:"name"         bun:"name,notnull"`
	Abbreviation string   `json:"abbreviation" bun:"abbreviation,notnull"`
	CountryName  string   `json:"countryName"  bun:"country_name,notnull"`
	CountryIso3  string   `json:"countryIso3"  bun:"country_iso3,notnull,default:'USA'"`
	CreatedAt    int64    `json:"createdAt"    bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64    `json:"updatedAt"    bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func RegionForStateAbbreviation(abbreviation string) (Region, bool) {
	region, ok := stateAbbreviationRegions[abbreviation]
	return region, ok
}

func (us *UsState) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if us.ID.IsNil() {
			us.ID = pulid.MustNew("us_")
		}

		us.CreatedAt = now
	case *bun.UpdateQuery:
		us.UpdatedAt = now
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
