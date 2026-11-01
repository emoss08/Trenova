package permitresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermitResolvers_CheckTheRESTPermissions(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_").String()
	input := gqlmodel.PermitInput{
		StateID:      pulid.MustNew("us_").String(),
		PermitNumber: "GA-1",
		Status:       permit.StatusActive,
	}

	calls := []struct {
		name      string
		resource  permission.Resource
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"shipmentPermitAssessment",
			permission.ResourceShipment,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).ShipmentPermitAssessment(ctx, shipmentID)
				return err
			},
		},
		{
			"shipmentPermits",
			permission.ResourcePermit,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).ShipmentPermits(ctx, shipmentID)
				return err
			},
		},
		{
			"shipmentPermitRequirements",
			permission.ResourcePermit,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).ShipmentPermitRequirements(ctx, shipmentID)
				return err
			},
		},
		{
			"createShipmentPermit",
			permission.ResourcePermit,
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateShipmentPermit(ctx, shipmentID, input)
				return err
			},
		},
		{
			"updateShipmentPermit",
			permission.ResourcePermit,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateShipmentPermit(
					ctx,
					shipmentID,
					pulid.MustNew("pmt_").String(),
					input,
				)
				return err
			},
		},
		{
			"waiveShipmentPermitRequirement",
			permission.ResourcePermit,
			permission.OpApprove,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).WaiveShipmentPermitRequirement(
					ctx,
					pulid.MustNew("pmr_").String(),
					"Escort arranged with the state patrol",
				)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, call.resource, call.operation)
		})
	}
}

func TestPermitFromInput_TakesTheShipmentFromThePathAndTheTenantFromTheSession(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	shipmentID := pulid.MustNew("shp_")
	stateID := pulid.MustNew("us_")
	issued, expires := 1_700_000_000, 1_700_864_000
	cost := "125.50"
	notes := "Oversize width"
	version := 4

	entity, err := permitFromInput(&gqlmodel.PermitInput{
		StateID:      stateID.String(),
		PermitNumber: "TX-778",
		Status:       permit.StatusActive,
		IssuedAt:     &issued,
		ExpiresAt:    &expires,
		Cost:         &cost,
		Notes:        &notes,
		Version:      &version,
	}, shipmentID.String(), authCtx)
	require.NoError(t, err)

	assert.Equal(t, shipmentID, entity.ShipmentID)
	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, stateID, entity.StateID)
	assert.Equal(t, "TX-778", entity.PermitNumber)
	assert.Equal(t, permit.StatusActive, entity.Status)
	require.NotNil(t, entity.IssuedAt)
	assert.Equal(t, int64(issued), *entity.IssuedAt)
	require.NotNil(t, entity.ExpiresAt)
	assert.Equal(t, int64(expires), *entity.ExpiresAt)
	assert.True(t, entity.Cost.Valid)
	assert.Equal(t, "125.5", entity.Cost.Decimal.String())
	assert.Equal(t, "Oversize width", entity.Notes)
	assert.Equal(t, int64(4), entity.Version)
}

func TestPermitFromInput_NamesAMalformedCost(t *testing.T) {
	t.Parallel()

	cost := "a lot"
	_, err := permitFromInput(&gqlmodel.PermitInput{
		StateID:      pulid.MustNew("us_").String(),
		PermitNumber: "TX-1",
		Status:       permit.StatusPending,
		Cost:         &cost,
	}, pulid.MustNew("shp_").String(), resolvertest.UserAuthContext())

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "cost", fieldErr.Field)
}

func TestPermitDecimalFields_AreDecimalStringsOrNull(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cost, err := (&PermitResolver{}).Cost(ctx, &permit.Permit{
		Cost: decimal.NewNullDecimal(decimal.RequireFromString("310.25")),
	})
	require.NoError(t, err)
	require.NotNil(t, cost)
	assert.Equal(t, "310.25", *cost)

	fee, err := (&PermitRequirementResolver{}).EstimatedFee(ctx, &permit.Requirement{})
	require.NoError(t, err)
	assert.Nil(t, fee)

	total, err := (&PermitAssessmentResolver{}).TotalEstimatedFee(
		ctx,
		&permitports.PermitAssessment{
			TotalEstimatedFee: decimal.RequireFromString("90"),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "90.00", total)
}
