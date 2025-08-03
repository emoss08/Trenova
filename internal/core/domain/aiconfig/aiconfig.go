/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package aiconfig

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
	_ bun.BeforeAppendModelHook = (*AIConfig)(nil)
	_ domain.Validatable        = (*AIConfig)(nil)
)

type AIConfig struct {
	bun.BaseModel `bun:"table:ai_configs,alias:ac" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	APIKey    string `json:"apiKey"    bun:"api_key,type:VARCHAR(255),notnull"`
	Version   int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// * Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (ac *AIConfig) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, ac,
		validation.Field(&ac.APIKey, validation.Required.Error("API Key is required")),
	)

	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (ac *AIConfig) GetID() string {
	return ac.ID.String()
}

func (ac *AIConfig) GetTableName() string {
	return "ai_configs"
}

func (ac *AIConfig) GetVersion() int64 {
	return ac.Version
}

func (ac *AIConfig) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ac.ID.IsNil() {
			ac.ID = pulid.MustNew("ac_")
		}

		ac.CreatedAt = now
	case *bun.UpdateQuery:
		ac.UpdatedAt = now
	}

	return nil
}
