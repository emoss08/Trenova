package routingguideresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func guideInput(carrierID string) gqlmodel.RoutingGuideInput {
	city, state := "Atlanta", "GA"
	destCity, destState := "Chicago", "IL"
	useContract := true
	version := 3

	return gqlmodel.RoutingGuideInput{
		Name:             "ATL to CHI",
		Status:           domaintypes.StatusActive,
		OriginCity:       &city,
		OriginState:      &state,
		DestinationCity:  &destCity,
		DestinationState: &destState,
		Version:          &version,
		Entries: []*gqlmodel.RoutingGuideEntryInput{{
			CarrierID:       carrierID,
			Rank:            1,
			RateMethod:      shipmentvocab.CarrierRateMethodFlat,
			Rate:            "1850.50",
			UseContractRate: &useContract,
			OfferTTLSeconds: 3600,
			Channel:         tender.ChannelEmail,
		}},
	}
}

func TestRoutingGuideMutations_CheckTheRoutingGuidePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("rg_").String()
	input := guideInput(pulid.MustNew("car_").String())
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createRoutingGuide", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateRoutingGuide(ctx, input)
			return err
		}},
		{"updateRoutingGuide", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateRoutingGuide(ctx, id, input)
			return err
		}},
		{"deleteRoutingGuide", permission.OpDelete, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DeleteRoutingGuide(ctx, id)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			resolvertest.RequireRefused(
				t,
				call.run(ctx, deps),
				engine,
				permission.ResourceRoutingGuide,
				call.operation,
			)
		})
	}
}

func TestRoutingGuideFromInput_MapsTheWaterfallIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	carrierID := pulid.MustNew("car_")

	input := guideInput(carrierID.String())
	entity, err := routingGuideFromInput(&input, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, "Atlanta", entity.OriginCity)
	assert.Nil(t, entity.OriginLocationID)
	assert.Equal(t, int64(3), entity.Version)
	require.Len(t, entity.Entries, 1)
	entry := entity.Entries[0]
	assert.Equal(t, carrierID, entry.CarrierID)
	assert.Equal(t, authCtx.OrganizationID, entry.OrganizationID)
	assert.True(t, entry.ID.IsNil())
	assert.True(t, entry.UseContractRate)
	assert.Equal(t, "1850.5", entry.Rate.String())
	assert.Equal(t, int32(3600), entry.OfferTTLSeconds)
}

func TestRoutingGuideFromInput_NamesAMissingCarrier(t *testing.T) {
	t.Parallel()

	input := guideInput("")
	_, err := routingGuideFromInput(&input, resolvertest.UserAuthContext())

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "carrierId", valErr.Field)
}
