package rateresolver

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func rateZoneFromInput(
	input *gqlmodel.RateZoneInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) (*ratezone.RateZone, error) {
	members := make([]*ratezone.RateZoneMember, 0, len(input.Members))
	for i, item := range input.Members {
		if item == nil {
			continue
		}
		memberID, err := gqlid.OptionalScopedID(fmt.Sprintf("members[%d].id", i), item.ID)
		if err != nil {
			return nil, err
		}

		members = append(members, &ratezone.RateZoneMember{
			ID:             memberID,
			OrganizationID: authCtx.OrganizationID,
			BusinessUnitID: authCtx.BusinessUnitID,
			RateZoneID:     id,
			ScopeType:      rategeo.ScopeType(item.ScopeType),
			ScopeValue:     gqlscalar.StringValue(item.ScopeValue),
			City:           gqlscalar.StringValue(item.City),
		})
	}

	entity := &ratezone.RateZone{
		ID:             id,
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		Code:           input.Code,
		Name:           input.Name,
		Description:    gqlscalar.StringValue(input.Description),
		Kind:           ratezone.ZoneKind(input.Kind),
		Status:         input.Status,
		Members:        members,
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return entity, nil
}
