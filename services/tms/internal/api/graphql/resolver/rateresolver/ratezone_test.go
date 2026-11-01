package rateresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateZoneOperations_CheckTheRateZonePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("rzn_").String()
	input := gqlmodel.RateZoneInput{Code: "SE", Name: "Southeast", Kind: "Regional"}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"rateZone", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).RateZone(ctx, id)
			return err
		}},
		{"createRateZone", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateRateZone(ctx, input)
			return err
		}},
		{"updateRateZone", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateRateZone(ctx, id, input)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, permission.ResourceRateZone, call.operation)
		})
	}
}

func TestRateZoneFromInput_StampsTheZoneAndEveryMember(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	zoneID := pulid.MustNew("rzn_")
	memberID := pulid.MustNew("rzm_")
	memberRaw := memberID.String()
	georgia, florida, miami := "us_ga", "us_fl", "Miami"
	version := 4

	entity, err := rateZoneFromInput(&gqlmodel.RateZoneInput{
		Code:    "SE",
		Name:    "Southeast",
		Kind:    "Regional",
		Status:  domaintypes.StatusActive,
		Version: &version,
		Members: []*gqlmodel.RateZoneMemberInput{
			{ID: &memberRaw, ScopeType: "State", ScopeValue: &georgia},
			nil,
			{ScopeType: "CityState", ScopeValue: &florida, City: &miami},
		},
	}, zoneID, authCtx)
	require.NoError(t, err)

	assert.Equal(t, zoneID, entity.ID)
	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, ratezone.ZoneKind("Regional"), entity.Kind)
	assert.Empty(t, entity.Description)
	assert.Equal(t, int64(4), entity.Version)
	require.Len(t, entity.Members, 2)
	assert.Equal(t, memberID, entity.Members[0].ID)
	assert.Equal(t, rategeo.ScopeTypeState, entity.Members[0].ScopeType)
	assert.Equal(t, "us_ga", entity.Members[0].ScopeValue)
	assert.True(t, entity.Members[1].ID.IsNil())
	assert.Equal(t, "Miami", entity.Members[1].City)
	for _, member := range entity.Members {
		assert.Equal(t, zoneID, member.RateZoneID)
		assert.Equal(t, authCtx.BusinessUnitID, member.BusinessUnitID)
	}
}

func TestRateZoneFromInput_NamesAMalformedMember(t *testing.T) {
	t.Parallel()

	bad := "nope"
	_, err := rateZoneFromInput(&gqlmodel.RateZoneInput{
		Members: []*gqlmodel.RateZoneMemberInput{{ID: &bad, ScopeType: "State"}},
	}, pulid.Nil, resolvertest.UserAuthContext())

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "members[0].id", valErr.Field)
}

func TestRateZoneToModel_ShowsMembersOnlyWhenTheyWereRead(t *testing.T) {
	t.Parallel()

	zoneID := pulid.MustNew("rzn_")
	memberID := pulid.MustNew("rzm_")

	listed := rateZoneToModel(&ratezone.RateZone{ID: zoneID, Kind: ratezone.ZoneKindKMA})
	assert.Equal(t, "KMA", listed.Kind)
	assert.Nil(t, listed.Members)

	read := rateZoneToModel(&ratezone.RateZone{
		ID: zoneID,
		Members: []*ratezone.RateZoneMember{{
			ID:         memberID,
			RateZoneID: zoneID,
			ScopeType:  rategeo.ScopeTypeZip3,
			ScopeValue: "606",
			MatchKey:   "Z3:606",
		}},
	})
	assert.Equal(t, []*gqlmodel.RateZoneMember{{
		ID:         memberID.String(),
		RateZoneID: zoneID.String(),
		ScopeType:  "Zip3",
		ScopeValue: "606",
		MatchKey:   "Z3:606",
	}}, read.Members)

	empty := rateZoneToModel(&ratezone.RateZone{ID: zoneID, Members: []*ratezone.RateZoneMember{}})
	assert.Equal(t, []*gqlmodel.RateZoneMember{}, empty.Members)
}
