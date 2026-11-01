package fiscalyearresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func fiscalYearInput() gqlmodel.FiscalYearInput {
	description := "Calendar 2026"
	isCurrent := true
	isCalendarYear := true
	allowAdjusting := true
	version := 4

	return gqlmodel.FiscalYearInput{
		Status:                fiscalyear.StatusOpen,
		Year:                  2026,
		Name:                  "FY 2026",
		Description:           &description,
		StartDate:             1767225600,
		EndDate:               1798761599,
		IsCurrent:             &isCurrent,
		IsCalendarYear:        &isCalendarYear,
		AllowAdjustingEntries: &allowAdjusting,
		Version:               &version,
	}
}

func TestFiscalYearOperations_CheckTheFiscalYearPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("fyr_").String()
	input := fiscalYearInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"fiscalYearClosePreview", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).FiscalYearClosePreview(ctx, id)
			return err
		}},
		{"createFiscalYear", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateFiscalYear(ctx, input)
			return err
		}},
		{"updateFiscalYear", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateFiscalYear(ctx, id, input)
			return err
		}},
		{"activateFiscalYear", permission.OpActivate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ActivateFiscalYear(ctx, id)
			return err
		}},
		{"closeFiscalYear", permission.OpClose, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CloseFiscalYear(ctx, id)
			return err
		}},
		{"reopenFiscalYear", permission.OpReopen, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ReopenFiscalYear(ctx, id, "Audit adjustment")
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
				permission.ResourceFiscalYear,
				call.operation,
			)
		})
	}
}

func TestFiscalYearFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	input := fiscalYearInput()

	entity := fiscalYearFromInput(&input, authCtx)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, fiscalyear.StatusOpen, entity.Status)
	assert.Equal(t, 2026, entity.Year)
	assert.Equal(t, "FY 2026", entity.Name)
	assert.Equal(t, "Calendar 2026", entity.Description)
	assert.Equal(t, int64(1767225600), entity.StartDate)
	assert.Equal(t, int64(1798761599), entity.EndDate)
	assert.True(t, entity.IsCurrent)
	assert.True(t, entity.IsCalendarYear)
	assert.True(t, entity.AllowAdjustingEntries)
	assert.Equal(t, int64(4), entity.Version)
}

func TestApplyFiscalYearInput_KeepsTheCloseHistoryTheInputDoesNotCarry(t *testing.T) {
	t.Parallel()

	closedAt := int64(1798761600)
	closedBy := pulid.MustNew("usr_")
	existing := &fiscalyear.FiscalYear{
		ID:         pulid.MustNew("fyr_"),
		Version:    9,
		ClosedAt:   &closedAt,
		ClosedByID: closedBy,
	}
	input := fiscalYearInput()
	input.Description = nil
	input.Version = nil

	applyFiscalYearInput(existing, &input)

	assert.Equal(t, "FY 2026", existing.Name)
	assert.Empty(t, existing.Description)
	assert.Equal(t, int64(9), existing.Version)
	assert.Equal(t, &closedAt, existing.ClosedAt)
	assert.Equal(t, closedBy, existing.ClosedByID)
}
