package routing

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

type LocationIndex struct {
	bun.BaseModel `bun:"table:location_indices,alias:li" json:"-"`

	// Primary identifiers
	ID pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`

	// Relationship identifiers (Non-Primary-Keys)
	InternalStateID *pulid.ID `bun:"internal_state_id,type:VARCHAR(100),nullzero" json:"internalStateId"`

	// Core Fields
	Provider        Provider `bun:"provider,type:routing_provider_enum,notnull" json:"provider"`
	StreetAddress   string   `bun:"street_address,type:VARCHAR(255),notnull" json:"streetAddress"`
	LocalArea       string   `bun:"local_area,type:VARCHAR(255),notnull" json:"localArea"`
	City            string   `bun:"city,type:VARCHAR(255),notnull" json:"city"`
	State           string   `bun:"state,type:VARCHAR(255),notnull" json:"state"`
	StateName       string   `bun:"state_name,type:VARCHAR(255),notnull" json:"stateName"`
	PostalCode      string   `bun:"postal_code,type:VARCHAR(255),notnull" json:"postalCode"`
	Country         string   `bun:"country,type:VARCHAR(255),notnull" json:"country"`
	CountryFullName string   `bun:"country_full_name,type:VARCHAR(255),notnull" json:"countryFullName"`
	SPLC            *string  `bun:"splc,type:VARCHAR(255),nullzero" json:"splc"`
	Longitude       *float64 `json:"longitude" bun:"longitude,type:FLOAT,nullzero"`
	Latitude        *float64 `json:"latitude" bun:"latitude,type:FLOAT,nullzero"`
	ShortString     string   `bun:"short_string,type:VARCHAR(255),notnull" json:"shortString"`
	TimeZone        string   `bun:"time_zone,type:VARCHAR(255),notnull" json:"timeZone"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	InternalState *usstate.UsState `bun:"rel:belongs-to,join:internal_state_id=id" json:"internalState,omitempty"`
}

func (l *LocationIndex) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("li_")
		}

		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}

	return nil
}

func (l *LocationIndex) GetTableName() string {
	return "location_indices"
}
