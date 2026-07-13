package ratetableservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testDeps struct {
	repo  *mocks.MockRateTableRepository
	audit *mocks.MockAuditService
	svc   *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockRateTableRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := &Service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    &Validator{repo: repo},
		auditService: auditSvc,
	}
	return &testDeps{repo: repo, audit: auditSvc, svc: svc}
}

//go:fix inline
func strPtr(s string) *string {
	return new(s)
}

func nullDec(v string) decimal.NullDecimal {
	return decimal.NullDecimal{Decimal: decimal.RequireFromString(v), Valid: true}
}

func newExactEntity() *ratetable.RateTable {
	return &ratetable.RateTable{
		ID:             pulid.MustNew("rt_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Name:           "Lane Rates",
		Key:            "lane_rate",
		LookupType:     ratetable.LookupTypeExact,
		Active:         true,
		Version:        1,
		Entries: []*ratetable.RateTableEntry{
			{MatchKey: new("ATL-MIA"), Value: decimal.RequireFromString("1450")},
			{MatchKey: new("ATL-JAX"), Value: decimal.RequireFromString("980")},
		},
	}
}

func newRangeEntity() *ratetable.RateTable {
	return &ratetable.RateTable{
		ID:             pulid.MustNew("rt_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Name:           "Fuel Surcharge",
		Key:            "fuel_surcharge",
		LookupType:     ratetable.LookupTypeRange,
		Active:         true,
		Version:        1,
		Entries: []*ratetable.RateTableEntry{
			{RangeMin: nullDec("0"), RangeMax: nullDec("3"), Value: decimal.Zero},
			{RangeMin: nullDec("3"), Value: decimal.RequireFromString("0.12")},
		},
	}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	entity.ID = pulid.Nil
	userID := pulid.MustNew("usr_")

	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{}, nil)
	deps.repo.On("Create", mock.Anything, entity).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, entity.Key, result.Key)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestCreate_DuplicateKey(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	entity.ID = pulid.Nil
	userID := pulid.MustNew("usr_")

	existing := newExactEntity()
	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{existing}, nil)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	entity.Name = ""
	entity.Key = ""
	userID := pulid.MustNew("usr_")

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "GetByKeys")
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_InvalidKeyPattern(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	entity.Key = "1invalid-key"
	userID := pulid.MustNew("usr_")

	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{}, nil)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_ExactEntriesMissingMatchKey(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	entity.Entries[0].MatchKey = nil
	userID := pulid.MustNew("usr_")

	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{}, nil)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_RangeEntriesOverlap(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newRangeEntity()
	entity.Entries = []*ratetable.RateTableEntry{
		{RangeMin: nullDec("0"), RangeMax: nullDec("5"), Value: decimal.Zero},
		{RangeMin: nullDec("3"), Value: decimal.RequireFromString("0.12")},
	}
	userID := pulid.MustNew("usr_")

	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{}, nil)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_RangeSuccess(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newRangeEntity()
	entity.ID = pulid.Nil
	userID := pulid.MustNew("usr_")

	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{}, nil)
	deps.repo.On("Create", mock.Anything, entity).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, ratetable.LookupTypeRange, result.LookupType)
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	userID := pulid.MustNew("usr_")

	original := newExactEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.Name = "Old Name"

	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{entity}, nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, entity.Name, result.Name)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestUpdate_DuplicateKey(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	userID := pulid.MustNew("usr_")

	other := newExactEntity()

	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{other}, nil)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestUpdate_GetOriginalError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	userID := pulid.MustNew("usr_")
	getErr := errors.New("not found")

	deps.repo.On("GetByKeys", mock.Anything, mock.Anything).
		Return([]*ratetable.RateTable{}, nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, getErr)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, getErr, err)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()
	userID := pulid.MustNew("usr_")

	req := &repositories.GetRateTableByIDRequest{
		RateTableID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)
	deps.repo.On("Delete", mock.Anything, req).Return(nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	err := deps.svc.Delete(ctx, req, userID)

	require.NoError(t, err)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestDelete_GetError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	getErr := errors.New("not found")

	req := &repositories.GetRateTableByIDRequest{
		RateTableID: pulid.MustNew("rt_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(nil, getErr)

	err := deps.svc.Delete(ctx, req, pulid.MustNew("usr_"))

	require.Error(t, err)
	assert.Equal(t, getErr, err)
	deps.repo.AssertNotCalled(t, "Delete")
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*ratetable.RateTable]{
		Items: []*ratetable.RateTable{newExactEntity()},
		Total: 1,
	}
	req := &repositories.ListRateTablesRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newExactEntity()

	req := &repositories.GetRateTableByIDRequest{
		RateTableID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)

	result, err := deps.svc.GetByID(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
}

func TestSelectOptions_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*ratetable.RateTable]{
		Items: []*ratetable.RateTable{newExactEntity()},
		Total: 1,
	}
	req := &repositories.RateTableSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{},
	}

	deps.repo.On("SelectOptions", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result.Items, 1)
}
