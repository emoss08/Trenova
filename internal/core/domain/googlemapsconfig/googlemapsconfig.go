package googlemapsconfig

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*GoogleMapsConfig)(nil)
	_ domain.Validatable        = (*GoogleMapsConfig)(nil)
)

type GoogleMapsConfig struct {
	bun.BaseModel `bun:"table:googlemaps_config,alias:gmc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// API Key
	APIKey string `json:"apiKey" bun:"api_key,type:TEXT,notnull"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (gmc *GoogleMapsConfig) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, gmc,
		validation.Field(&gmc.APIKey, validation.Required.Error("API Key is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (gmc *GoogleMapsConfig) GetID() string {
	return gmc.ID.String()
}

func (gmc *GoogleMapsConfig) GetTableName() string {
	return "googlemaps_config"
}

func (gmc *GoogleMapsConfig) GetVersion() int64 {
	return gmc.Version
}

func (gmc *GoogleMapsConfig) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if gmc.ID.IsNil() {
			gmc.ID = pulid.MustNew("gmc_")
		}

		gmc.CreatedAt = now
	case *bun.UpdateQuery:
		gmc.UpdatedAt = now
	}

	return nil
}
