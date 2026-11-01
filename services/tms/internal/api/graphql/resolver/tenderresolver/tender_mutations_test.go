package tenderresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenderMutations_CheckTheTenderPermission(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_").String()
	tenderID := pulid.MustNew("tnd_").String()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createWaterfallTender", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateWaterfallTender(
				ctx,
				gqlmodel.CreateWaterfallTenderInput{ShipmentMoveID: moveID},
			)
			return err
		}},
		{"createSpotTender", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateSpotTender(
				ctx,
				gqlmodel.CreateSpotTenderInput{
					ShipmentMoveID: moveID,
					Mode:           tender.ModeSpotBroadcast,
				},
			)
			return err
		}},
		{"cancelTender", permission.OpCancel, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CancelTender(ctx, tenderID, "Covered internally")
			return err
		}},
		{"respondToTenderOffer", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).RespondToTenderOffer(
				ctx,
				pulid.MustNew("tof_").String(),
				gqlmodel.RespondToTenderOfferInput{Action: tender.ResponseActionAccept},
			)
			return err
		}},
		{"bounceTender", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).BounceTender(ctx, tenderID, "Truck broke down")
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

			resolvertest.RequireRefused(t, err, engine, permission.ResourceTender, call.operation)
		})
	}
}

func TestSpotTenderRequestFromInput_MapsEveryCarrierLine(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	firstCarrier := pulid.MustNew("car_")
	secondCarrier := pulid.MustNew("car_")
	email := "dispatch@carrier.test"
	override := true

	req, err := spotTenderRequestFromInput(&gqlmodel.CreateSpotTenderInput{
		ShipmentMoveID: moveID.String(),
		Mode:           tender.ModeSpotSequential,
		Lines: []*gqlmodel.SpotTenderLineInput{
			{
				CarrierID:       firstCarrier.String(),
				RateMethod:      shipmentvocab.CarrierRateMethodFlat,
				Rate:            "1500.50",
				OfferTTLSeconds: 7200,
				Channel:         tender.ChannelEmail,
				Email:           &email,
			},
			{
				CarrierID:       secondCarrier.String(),
				RateMethod:      shipmentvocab.CarrierRateMethodFlat,
				Rate:            "1400",
				OfferTTLSeconds: -5,
				Channel:         tender.ChannelEDI,
			},
		},
		OverrideInsuranceWarnings: &override,
	})
	require.NoError(t, err)

	assert.Equal(t, moveID, req.ShipmentMoveID)
	assert.Equal(t, tender.ModeSpotSequential, req.Mode)
	assert.True(t, req.OverrideInsuranceWarnings)
	require.Len(t, req.Lines, 2)
	assert.Equal(t, firstCarrier, req.Lines[0].CarrierID)
	assert.Equal(t, "1500.5", req.Lines[0].Rate.String())
	assert.Equal(t, int32(7200), req.Lines[0].OfferTTLSeconds)
	assert.Equal(t, "dispatch@carrier.test", req.Lines[0].Email)
	assert.Equal(t, secondCarrier, req.Lines[1].CarrierID)
	assert.Equal(t, int32(0), req.Lines[1].OfferTTLSeconds)
	assert.Empty(t, req.Lines[1].Email)
}

func TestSpotTenderRequestFromInput_NamesAMalformedRate(t *testing.T) {
	t.Parallel()

	_, err := spotTenderRequestFromInput(&gqlmodel.CreateSpotTenderInput{
		ShipmentMoveID: pulid.MustNew("smv_").String(),
		Mode:           tender.ModeSpotBroadcast,
		Lines: []*gqlmodel.SpotTenderLineInput{{
			CarrierID: pulid.MustNew("car_").String(),
			Rate:      "cheap",
		}},
	})

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "lines.rate", fieldErr.Field)
}
