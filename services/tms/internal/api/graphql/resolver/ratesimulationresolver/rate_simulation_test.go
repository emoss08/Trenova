package ratesimulationresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratesimulation"
	pkgsimulation "github.com/emoss08/trenova/pkg/ratesimulation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateSimulationOperations_CheckTheSimulationPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("rsim_").String()
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"rateSimulations", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).RateSimulations(ctx, nil, nil)
			return err
		}},
		{"rateSimulation", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).RateSimulation(ctx, id)
			return err
		}},
		{"rateSimulationResults", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).RateSimulationResults(ctx, id, nil, nil)
			return err
		}},
		{"createRateSimulation", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateRateSimulation(
				ctx,
				gqlmodel.RateSimulationInput{RateAgreementID: pulid.MustNew("rag_").String()},
			)
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
				permission.ResourceRateSimulation,
				call.operation,
			)
		})
	}
}

func TestRateSimulationToModel_StatesTheRunAndItsCoverage(t *testing.T) {
	t.Parallel()

	lostTo := pulid.MustNew("ragr_")
	entity := &ratesimulation.RateSimulation{
		ID:     pulid.MustNew("rsim_"),
		Name:   "Acme renewal",
		Status: ratesimulation.StatusCompleted,
		Summary: &pkgsimulation.Summary{
			ShipmentCount: 40,
			BeforeTotal:   decimal.RequireFromString("1000.50"),
			TotalDeltaPct: decimal.RequireFromString("4.2"),
		},
		RuleCoverage: []*ratesimulation.RuleCoverage{{
			RuleID:      pulid.MustNew("ragr_"),
			Outcome:     ratesimulation.RuleOutcomeLost,
			LostCount:   3,
			LostTo:      &lostTo,
			LostToLabel: "Southeast flat",
		}},
	}

	model := rateSimulationToModel(entity)

	assert.Equal(t, ratesimulation.StatusCompleted, model.Status)
	require.NotNil(t, model.Summary)
	assert.Equal(t, 40, model.Summary.ShipmentCount)
	assert.Equal(t, "1000.5", model.Summary.BeforeTotal)
	assert.Equal(t, "4.2", model.Summary.TotalDeltaPct)
	require.Len(t, model.RuleCoverage, 1)
	assert.Equal(t, ratesimulation.RuleOutcomeLost, model.RuleCoverage[0].Outcome)
	assert.Equal(t, lostTo.String(), *model.RuleCoverage[0].LostTo)
	assert.Nil(t, model.StartedAt)
}

func TestRateSimulationResultsToModel_PricesEachShipmentBothWays(t *testing.T) {
	t.Parallel()

	results := rateSimulationResultsToModel([]*ratesimulation.RateSimulationResult{{
		ShipmentID:   pulid.MustNew("shp_"),
		ProNumber:    "PRO-1",
		BeforeAmount: decimal.RequireFromString("1500"),
		AfterAmount:  decimal.RequireFromString("1575"),
		Delta:        decimal.RequireFromString("75"),
		DeltaPercent: decimal.RequireFromString("5"),
		Outcome:      ratequote.OutcomeRated,
	}})

	require.Len(t, results, 1)
	assert.Equal(t, "1575", results[0].AfterAmount)
	assert.Equal(t, gqlmodel.RateQuoteOutcomeRated, results[0].Outcome)
	assert.Nil(t, results[0].CustomerID)
}
