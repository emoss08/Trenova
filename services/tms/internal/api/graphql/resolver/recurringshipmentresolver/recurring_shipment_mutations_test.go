package recurringshipmentresolver

import (
	"context"
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecurringShipmentResolvers_CheckTheRecurringShipmentPermission(t *testing.T) {
	t.Parallel()

	seriesID := pulid.MustNew("rsh_").String()
	input := gqlmodel.RecurringShipmentInput{
		SourceShipmentID: pulid.MustNew("shp_").String(),
		Name:             "Weekly Dallas run",
		CronExpression:   "0 8 * * 1",
		Timezone:         "America/Chicago",
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"recurringShipmentRuns", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).RecurringShipmentRuns(ctx, seriesID, nil)
			return err
		}},
		{"matchRecurringShipments", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).MatchRecurringShipments(
				ctx,
				gqlmodel.MatchRecurringShipmentsInput{},
			)
			return err
		}},
		{
			"createRecurringShipment",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateRecurringShipment(ctx, input)
				return err
			},
		},
		{
			"updateRecurringShipment",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateRecurringShipment(ctx, seriesID, input)
				return err
			},
		},
		{
			"updateRecurringShipmentStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateRecurringShipmentStatus(
					ctx,
					seriesID,
					recurringshipment.StatusPaused,
					2,
				)
				return err
			},
		},
		{
			"generateRecurringShipment",
			permission.OpDuplicate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).GenerateRecurringShipment(ctx, seriesID, nil)
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

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceRecurringShipment,
				call.operation,
			)
		})
	}
}

func TestApplyRecurringShipmentInput_KeepsTheSystemFieldsItDoesNotCarry(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("shp_")
	enteredBy := pulid.MustNew("usr_")
	next := int64(1_700_100_000)
	end := 1_800_000_000
	maxOccurrences := 12
	description := "Mondays to the DC"
	version := 7
	policy := recurringshipment.ExceptionPolicyNextBusinessDay

	entity := &recurringshipment.RecurringShipment{
		EnteredByID:         enteredBy,
		Status:              recurringshipment.StatusActive,
		NextOccurrenceAt:    &next,
		GenerationCount:     9,
		ConsecutiveFailures: 1,
	}
	err := applyRecurringShipmentInput(entity, &gqlmodel.RecurringShipmentInput{
		SourceShipmentID: sourceID.String(),
		Name:             "Weekly DC",
		Description:      &description,
		CronExpression:   "0 6 * * 1",
		Timezone:         "America/Chicago",
		EndDate:          &end,
		MaxOccurrences:   &maxOccurrences,
		LeadTimeDays:     2,
		SkipWeekends:     true,
		ExceptionPolicy:  &policy,
		BlackoutDates:    []string{"2026-12-25"},
		AutoGenerate:     true,
		Version:          &version,
	})
	require.NoError(t, err)

	assert.Equal(t, sourceID, entity.SourceShipmentID)
	assert.Equal(t, "Weekly DC", entity.Name)
	assert.Equal(t, "Mondays to the DC", entity.Description)
	assert.Equal(t, recurringshipment.StatusActive, entity.Status)
	assert.Equal(t, "0 6 * * 1", entity.CronExpression)
	require.NotNil(t, entity.EndDate)
	assert.Equal(t, int64(end), *entity.EndDate)
	require.NotNil(t, entity.MaxOccurrences)
	assert.Equal(t, int32(12), *entity.MaxOccurrences)
	assert.Equal(t, int16(2), entity.LeadTimeDays)
	assert.True(t, entity.SkipWeekends)
	assert.Equal(t, recurringshipment.ExceptionPolicyNextBusinessDay, entity.ExceptionPolicy)
	assert.Equal(t, []string{"2026-12-25"}, entity.BlackoutDates)
	assert.Equal(t, int64(7), entity.Version)
	assert.Equal(t, enteredBy, entity.EnteredByID)
	assert.Equal(t, &next, entity.NextOccurrenceAt)
	assert.Equal(t, int64(9), entity.GenerationCount)
	assert.Equal(t, int32(1), entity.ConsecutiveFailures)
}

func TestApplyRecurringShipmentInput_ClampsOutOfRangeCounts(t *testing.T) {
	t.Parallel()

	huge := math.MaxInt64
	entity := &recurringshipment.RecurringShipment{}
	err := applyRecurringShipmentInput(entity, &gqlmodel.RecurringShipmentInput{
		SourceShipmentID: pulid.MustNew("shp_").String(),
		MaxOccurrences:   &huge,
		LeadTimeDays:     math.MaxInt32,
	})
	require.NoError(t, err)

	require.NotNil(t, entity.MaxOccurrences)
	assert.Equal(t, int32(math.MaxInt32), *entity.MaxOccurrences)
	assert.Equal(t, int16(math.MaxInt16), entity.LeadTimeDays)
}

func TestRunsLimit_DefaultsAndCapsTheRunsRead(t *testing.T) {
	t.Parallel()

	zero, ten, thousand := 0, 10, 1000
	assert.Equal(t, defaultRunsLimit, runsLimit(nil))
	assert.Equal(t, defaultRunsLimit, runsLimit(&zero))
	assert.Equal(t, 10, runsLimit(&ten))
	assert.Equal(t, maxRunsLimit, runsLimit(&thousand))
}
