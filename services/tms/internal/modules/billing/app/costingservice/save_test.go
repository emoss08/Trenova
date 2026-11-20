package costingservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/costingcontrolportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type inTransaction struct{}

type noFieldGroups struct{}

func (noFieldGroups) Visible(
	context.Context,
	*permissionports.RequestActor,
) protectedfield.GroupSet {
	return 0
}

type recordingTransactor struct {
	calls int
}

func (r *recordingTransactor) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	r.calls++
	return fn(context.WithValue(ctx, inTransaction{}, true))
}

func requireInTransaction(t *testing.T, ctx context.Context) {
	t.Helper()
	require.Equal(t, true, ctx.Value(inTransaction{}), "write ran outside the save's transaction")
}

type saveFixture struct {
	repo    *costingcontrolportsmock.MockCostingControlRepository
	audit   *auditportsmock.MockAuditService
	tx      *recordingTransactor
	service *costingservice.Service
	stored  *costingcontrol.CostingControl
	userID  pulid.ID
}

func newSaveFixture(t *testing.T) *saveFixture {
	t.Helper()

	stored := benchmarkControl()
	stored.Version = 4
	for i, category := range stored.Categories {
		category.ID = pulid.MustNew("ccat_")
		category.Version = int64(i + 1)
	}

	f := &saveFixture{
		repo:   costingcontrolportsmock.NewMockCostingControlRepository(t),
		audit:  auditportsmock.NewMockAuditService(t),
		tx:     &recordingTransactor{},
		stored: stored,
		userID: pulid.MustNew("usr_"),
	}
	f.service = costingservice.NewTestService(costingservice.TestServiceParams{
		DB:           f.tx,
		Repo:         f.repo,
		AuditService: f.audit,
	})
	return f
}

func (f *saveFixture) request() *costingservice.SaveRequest {
	return &costingservice.SaveRequest{
		TenantInfo: testTenantInfo(),
		Actor:      &permissionports.RequestActor{UserID: f.userID},
	}
}

func (f *saveFixture) expectReads(times int) {
	f.repo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(f.stored, nil).Times(times)
}

func settingsOf(control *costingcontrol.CostingControl) *costingcontrol.CostingControl {
	return &costingcontrol.CostingControl{
		FuelIndexID:          control.FuelIndexID,
		UseLiveFuelPrice:     control.UseLiveFuelPrice,
		MilesPerGallon:       control.MilesPerGallon,
		IncludeDeadheadMiles: control.IncludeDeadheadMiles,
		GLActualsEnabled:     control.GLActualsEnabled,
		GLRollingMonths:      control.GLRollingMonths,
		PlannedMonthlyMiles:  control.PlannedMonthlyMiles,
		TargetMarginPercent:  control.TargetMarginPercent,
		Version:              control.Version,
	}
}

func TestSave_WritesTheControlAndChangedCategoriesInOneTransaction(t *testing.T) {
	t.Parallel()

	f := newSaveFixture(t)
	f.expectReads(2)

	glAccountID := pulid.MustNew("gla_")
	wages := f.stored.Categories[0]
	fuel := f.stored.Categories[2]

	req := f.request()
	req.Control = settingsOf(f.stored)
	req.Control.GLActualsEnabled = true
	req.Control.GLRollingMonths = 6
	req.Categories = []*costingservice.CategoryChange{
		{
			ID:           fuel.ID,
			RateSource:   costingcontrol.RateSourceGLActual,
			IsActive:     true,
			GLAccountIDs: []pulid.ID{glAccountID, glAccountID},
			Version:      fuel.Version,
		},
		{
			ID:                  wages.ID,
			RateSource:          costingcontrol.RateSourceOverride,
			OverrideRatePerMile: decimal.NewNullDecimal(decimal.RequireFromString("0.91")),
			IsActive:            true,
			Version:             wages.Version,
		},
	}

	f.repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, entity *costingcontrol.CostingControl) (*costingcontrol.CostingControl, error) {
			requireInTransaction(t, ctx)
			assert.Equal(t, f.stored.ID, entity.ID)
			assert.Equal(t, testOrgID, entity.OrganizationID)
			assert.Equal(t, testBuID, entity.BusinessUnitID)
			assert.True(t, entity.GLActualsEnabled)
			assert.Equal(t, int16(6), entity.GLRollingMonths)
			assert.Equal(t, int64(4), entity.Version)
			assert.Nil(t, entity.Categories)
			return entity, nil
		}).
		Once()
	f.repo.EXPECT().
		UpdateCategories(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, categories []*costingcontrol.CostCategory) error {
			requireInTransaction(t, ctx)
			require.Len(t, categories, 2)
			assert.Equal(t, fuel.ID, categories[0].ID)
			assert.Equal(t, costingcontrol.RateSourceGLActual, categories[0].RateSource)
			assert.Equal(t, fuel.Version, categories[0].Version)
			assert.Equal(t, fuel.Name, categories[0].Name)
			assert.Equal(t, wages.ID, categories[1].ID)
			assert.Equal(t, costingcontrol.RateSourceOverride, categories[1].RateSource)
			assert.Equal(t, "0.91", categories[1].OverrideRatePerMile.Decimal.String())
			return nil
		}).
		Once()
	f.repo.EXPECT().
		ReplaceCategoryGLAccounts(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, req *costingcontrolports.ReplaceCategoryGLAccountsRequest) error {
			requireInTransaction(t, ctx)
			assert.Equal(t, testTenantInfo(), req.TenantInfo)
			assert.Equal(t, []costingcontrolports.CategoryGLAccounts{
				{CostCategoryID: fuel.ID, GLAccountIDs: []pulid.ID{glAccountID}},
				{CostCategoryID: wages.ID, GLAccountIDs: []pulid.ID{}},
			}, req.Categories)
			return nil
		}).
		Once()
	f.audit.EXPECT().
		LogActions(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, entries []auditports.BulkLogEntry) error {
			requireInTransaction(t, ctx)
			require.Len(t, entries, 3)
			assert.Equal(t, f.stored.ID.String(), entries[0].Params.ResourceID)
			assert.Equal(t, fuel.ID.String(), entries[1].Params.ResourceID)
			assert.Equal(t, []string{glAccountID.String()},
				entries[1].Params.CurrentState["glAccountIds"])
			assert.Equal(t, []string{}, entries[1].Params.PreviousState["glAccountIds"])
			assert.Equal(t, wages.ID.String(), entries[2].Params.ResourceID)
			for _, entry := range entries {
				assert.Equal(t, f.userID, entry.Params.UserID)
				assert.Equal(t, testOrgID, entry.Params.OrganizationID)
			}
			return nil
		}).
		Once()

	saved, err := f.service.Save(t.Context(), req)
	require.NoError(t, err)
	assert.Same(t, f.stored, saved)
	assert.Equal(t, 1, f.tx.calls)
}

func TestSave_ACategoryOnlySaveLeavesTheControlAlone(t *testing.T) {
	t.Parallel()

	f := newSaveFixture(t)
	f.expectReads(2)
	tolls := f.stored.Categories[5]

	req := f.request()
	req.Categories = []*costingservice.CategoryChange{
		{ID: tolls.ID, RateSource: costingcontrol.RateSourceBenchmark, Version: tolls.Version},
	}

	f.repo.EXPECT().UpdateCategories(mock.Anything, mock.Anything).Return(nil).Once()
	f.repo.EXPECT().ReplaceCategoryGLAccounts(mock.Anything, mock.Anything).Return(nil).Once()
	f.audit.EXPECT().
		LogActions(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entries []auditports.BulkLogEntry) error {
			require.Len(t, entries, 1)
			assert.Equal(t, tolls.ID.String(), entries[0].Params.ResourceID)
			assert.Equal(t, false, entries[0].Params.CurrentState["isActive"])
			return nil
		}).
		Once()

	_, err := f.service.Save(t.Context(), req)
	require.NoError(t, err)
	f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestSave_RefusesACategoryOnGLActualsWithoutAnAccountAndWritesNothing(t *testing.T) {
	t.Parallel()

	f := newSaveFixture(t)
	f.stored.GLActualsEnabled = true
	f.expectReads(1)
	fuel := f.stored.Categories[2]
	tolls := f.stored.Categories[5]

	req := f.request()
	req.Categories = []*costingservice.CategoryChange{
		{ID: tolls.ID, RateSource: costingcontrol.RateSourceBenchmark, IsActive: true},
		{ID: fuel.ID, RateSource: costingcontrol.RateSourceGLActual, IsActive: true},
	}

	_, err := f.service.Save(t.Context(), req)

	multiErr := refusal(t, err)
	assert.Equal(t, []string{"categories[1].glAccountIds"}, refusedFields(multiErr))
	assert.Equal(t, 0, f.tx.calls)
}

func TestSave_RefusesTurningGLActualsOnOverAnUnmappedCategory(t *testing.T) {
	t.Parallel()

	f := newSaveFixture(t)
	f.expectReads(1)
	f.stored.Categories[3].RateSource = costingcontrol.RateSourceGLActual

	req := f.request()
	req.Control = settingsOf(f.stored)
	req.Control.GLActualsEnabled = true

	_, err := f.service.Save(t.Context(), req)

	multiErr := refusal(t, err)
	assert.Equal(t, []string{"control.glActualsEnabled"}, refusedFields(multiErr))
	assert.Equal(t, 0, f.tx.calls)
}

func TestSave_KeepsARollingWindowOutsideOneToTwelveWhileGLActualsAreOff(t *testing.T) {
	t.Parallel()

	f := newSaveFixture(t)
	f.expectReads(2)

	req := f.request()
	req.Control = settingsOf(f.stored)
	req.Control.GLRollingMonths = 0

	f.repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *costingcontrol.CostingControl) (*costingcontrol.CostingControl, error) {
			assert.Equal(t, int16(0), entity.GLRollingMonths)
			return entity, nil
		}).
		Once()
	f.audit.EXPECT().LogActions(mock.Anything, mock.Anything).Return(nil).Once()

	_, err := f.service.Save(t.Context(), req)
	require.NoError(t, err)
}

func TestSave_ReturnsAFailedCategoryWriteWithoutAuditing(t *testing.T) {
	t.Parallel()

	f := newSaveFixture(t)
	f.expectReads(1)
	tolls := f.stored.Categories[5]
	conflict := errors.New("version mismatch")

	req := f.request()
	req.Control = settingsOf(f.stored)
	req.Control.IncludeDeadheadMiles = false
	req.Categories = []*costingservice.CategoryChange{
		{ID: tolls.ID, RateSource: costingcontrol.RateSourceBenchmark, Version: 99},
	}

	f.repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *costingcontrol.CostingControl) (*costingcontrol.CostingControl, error) {
			return entity, nil
		}).
		Once()
	f.repo.EXPECT().UpdateCategories(mock.Anything, mock.Anything).Return(conflict).Once()

	_, err := f.service.Save(t.Context(), req)
	require.ErrorIs(t, err, conflict)
	assert.Equal(t, 1, f.tx.calls)
	f.audit.AssertNotCalled(t, "LogActions", mock.Anything, mock.Anything)
}

func TestSave_NothingToSaveReturnsTheStoredControl(t *testing.T) {
	t.Parallel()

	f := newSaveFixture(t)
	f.expectReads(1)

	saved, err := f.service.Save(t.Context(), f.request())
	require.NoError(t, err)
	assert.Same(t, f.stored, saved)
	assert.Equal(t, 0, f.tx.calls)
}

func TestSave_KeepsAWithheldTargetMargin(t *testing.T) {
	t.Parallel()

	f := newSaveFixture(t)
	f.stored.TargetMarginPercent = decimal.NewNullDecimal(decimal.NewFromInt(18))
	f.expectReads(2)

	req := f.request()
	req.Control = settingsOf(f.stored)
	req.Control.TargetMarginPercent = decimal.NullDecimal{}

	f.repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *costingcontrol.CostingControl) (*costingcontrol.CostingControl, error) {
			assert.Equal(t, "18", entity.TargetMarginPercent.Decimal.String())
			return entity, nil
		}).
		Once()
	f.audit.EXPECT().LogActions(mock.Anything, mock.Anything).Return(nil).Once()

	service := costingservice.NewTestService(costingservice.TestServiceParams{
		DB:           f.tx,
		Repo:         f.repo,
		AuditService: f.audit,
		Fields:       noFieldGroups{},
	})

	_, err := service.Save(t.Context(), req)
	require.NoError(t, err)
}
