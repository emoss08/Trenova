package formulatemplateservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/formulatemplatetest"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func catalogKeys(t *testing.T, except ...string) []string {
	t.Helper()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)

	skip := make(map[string]struct{}, len(except))
	for _, key := range except {
		skip[key] = struct{}{}
	}
	keys := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		if _, skipped := skip[entry.Key]; !skipped {
			keys = append(keys, entry.Key)
		}
	}

	return keys
}

func selectRequest(tenant pagination.TenantInfo) *formulatemplateports.FormulaTemplateSelectOptionsRequest {
	return &formulatemplateports.FormulaTemplateSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: 20},
		},
	}
}

func expectSelectOptions(
	deps *testDeps,
	standardOnly bool,
) {
	deps.repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(
			func(req *formulatemplateports.FormulaTemplateSelectOptionsRequest) bool {
				return req.StandardOnly == standardOnly
			},
		)).
		Return(&pagination.ListResult[*formulatemplate.FormulaTemplate]{}, nil).
		Once()
}

func TestSelectOptions_ABasePlanListsOnlyTheStandardTemplates(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.features = plantest.BaseOnly()
	deps.repo.EXPECT().ListStandardKeys(mock.Anything, mock.Anything).
		Return(catalogKeys(t), nil).Once()
	expectSelectOptions(deps, true)

	_, err := deps.svc.SelectOptions(t.Context(), selectRequest(formulatemplatetest.NewTenantInfo()))

	require.NoError(t, err)
}

func TestSelectOptions_WithRatingListsEveryActiveTemplate(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.repo.EXPECT().ListStandardKeys(mock.Anything, mock.Anything).
		Return(catalogKeys(t), nil).Once()
	expectSelectOptions(deps, false)

	_, err := deps.svc.SelectOptions(t.Context(), selectRequest(formulatemplatetest.NewTenantInfo()))

	require.NoError(t, err)
}

func TestEnsureStandards_InstallsWhatTheOrganizationIsMissingOnce(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenant := formulatemplatetest.NewTenantInfo()
	deps.repo.EXPECT().ListStandardKeys(mock.Anything, mock.Anything).
		Return(catalogKeys(t, "flat_rate"), nil).Once()
	deps.repo.EXPECT().FindByNames(mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil).Once()

	var installed *formulatemplate.FormulaTemplate
	deps.repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *formulatemplate.FormulaTemplate,
		) (*formulatemplate.FormulaTemplate, error) {
			installed = entity
			return entity, nil
		}).Once()
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsVersion, nil).Once()
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	require.NoError(t, deps.svc.EnsureStandards(t.Context(), tenant))
	require.NoError(t, deps.svc.EnsureStandards(t.Context(), tenant), "the second call reads nothing")

	require.NotNil(t, installed)
	assert.Equal(t, "flat_rate", installed.StandardKey)
	assert.Equal(t, formulatemplate.StandardFlatRate, installed.Name)
	assert.Equal(t, formulatemplate.StatusActive, installed.Status)
	require.NotNil(t, installed.ApprovedByID)
	assert.Equal(t, tenant.UserID, *installed.ApprovedByID)
	assert.NotEmpty(t, installed.Inputs)
}

func TestEnsureStandards_InstallsBesideATemplateOfTheSameName(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenant := formulatemplatetest.NewTenantInfo()
	deps.repo.EXPECT().ListStandardKeys(mock.Anything, mock.Anything).
		Return(catalogKeys(t, "flat_rate"), nil).Once()
	deps.repo.EXPECT().FindByNames(mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{{Name: formulatemplate.StandardFlatRate}}, nil).
		Once()
	deps.repo.EXPECT().TakenNames(mock.Anything, mock.Anything).
		Return(map[string]struct{}{}, nil).Once()

	var installed *formulatemplate.FormulaTemplate
	deps.repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *formulatemplate.FormulaTemplate,
		) (*formulatemplate.FormulaTemplate, error) {
			installed = entity
			return entity, nil
		}).Once()
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsVersion, nil).Once()
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	require.NoError(t, deps.svc.EnsureStandards(t.Context(), tenant))

	require.NotNil(t, installed)
	assert.Equal(t, "Flat Rate (Standard)", installed.Name)
	assert.Equal(t, "flat_rate", installed.StandardKey)
}

func TestOffPlan_ReadsNothingWithRating(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)

	off, err := deps.svc.OffPlan(t.Context(), formulatemplatetest.NewTenantInfo(), pulid.MustNew("ft_"))

	require.NoError(t, err)
	assert.Empty(t, off, "with Rating every template prices, and the repository is not read")
}

func TestRefuseOffPlan_RefusesTheOrganizationsOwnTemplateOnABasePlan(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.features = plantest.BaseOnly()
	own := pulid.MustNew("ft_")
	standard := pulid.MustNew("ft_")
	deps.repo.EXPECT().NonStandardIDs(mock.Anything, mock.Anything).
		Return([]pulid.ID{own}, nil).Once()

	err := deps.svc.RefuseOffPlan(t.Context(), formulatemplatetest.NewTenantInfo(), own, standard)

	require.Error(t, err)
	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "a plan restriction, got %v", err)
	assert.Equal(t, string(platformcatalog.FeatureRating), restriction.Feature)
}

func TestRefuseOffPlan_AllowsAStandardTemplateOnABasePlan(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.features = plantest.BaseOnly()
	deps.repo.EXPECT().NonStandardIDs(mock.Anything, mock.Anything).
		Return([]pulid.ID{}, nil).Once()

	require.NoError(t, deps.svc.RefuseOffPlan(
		t.Context(),
		formulatemplatetest.NewTenantInfo(),
		pulid.MustNew("ft_"),
		pulid.Nil,
	))
}

func TestUpdate_RefusesAStandardTemplate(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	standard := formulatemplatetest.NewTemplate()
	standard.StandardKey = "per_mile"
	deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(standard, nil).Once()

	_, err := deps.svc.Update(t.Context(), standard, formulatemplatetest.NewTenantInfo().UserID)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	deps.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestBulkUpdateStatus_RefusesToArchiveAStandardTemplate(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	standard := formulatemplatetest.NewTemplate()
	standard.Status = formulatemplate.StatusActive
	standard.StandardKey = "per_mile"
	deps.repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{standard}, nil).Once()

	_, err := deps.svc.BulkUpdateStatus(t.Context(), &formulatemplateports.BulkUpdateFormulaTemplateStatusRequest{
		TenantInfo:  formulatemplatetest.NewTenantInfo(),
		TemplateIDs: []pulid.ID{standard.ID},
		Status:      formulatemplate.StatusInactive,
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestPricingInputs_AStandardTemplateIsReadOnABasePlan(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.features = plantest.BaseOnly()
	standard := formulatemplatetest.NewTemplate()
	standard.StandardKey = "per_mile"
	standard.Status = formulatemplate.StatusActive
	deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(standard, nil).Once()
	deps.versionRepo.EXPECT().ListApproved(mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplateVersion{}, nil).Maybe()

	inputs, err := deps.svc.PricingInputs(t.Context(), &PricingInputsRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		TemplateID: standard.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, nonNilInputs(standard.Inputs), inputs.Inputs)
}

func TestPricingInputs_TheOrganizationsOwnTemplateIsRefusedOnABasePlan(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.features = plantest.BaseOnly()
	own := formulatemplatetest.NewTemplate()
	deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(own, nil).Once()

	_, err := deps.svc.PricingInputs(t.Context(), &PricingInputsRequest{
		TenantInfo:    formulatemplatetest.NewTenantInfo(),
		TemplateID:    own.ID,
		MayReadCustom: true,
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsPlanRestrictionError(err), "a plan restriction, got %v", err)
}

func TestPricingInputs_TheOrganizationsOwnTemplateNeedsItsReadPermission(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	own := formulatemplatetest.NewTemplate()
	deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(own, nil).Once()

	_, err := deps.svc.PricingInputs(t.Context(), &PricingInputsRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		TemplateID: own.ID,
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err), "an authorization error, got %v", err)
}
