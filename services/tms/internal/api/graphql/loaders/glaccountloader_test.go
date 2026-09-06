package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGLAccountBatchFunc_PreservesOrderAndReportsMissingIDs(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("gla_")
	secondID := pulid.MustNew("gla_")
	missingID := pulid.MustNew("gla_")
	repo := &stubGLAccountRepository{
		getByIDs: func(_ context.Context, req repositories.GetGLAccountsByIDsRequest) ([]*glaccount.GLAccount, error) {
			assert.Equal(t, []pulid.ID{secondID, firstID, missingID}, req.GLAccountIDs)
			return []*glaccount.GLAccount{
				{ID: firstID, AccountCode: "1000"},
				{ID: secondID, AccountCode: "2000"},
			}, nil
		},
	}
	factory := &GLAccountByIDLoaderFactory{glAccountRepo: repo}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		secondID.String(),
		"bad",
		firstID.String(),
		missingID.String(),
		secondID.String(),
	})

	require.Len(t, values, 5)
	require.NoError(t, errs[0])
	assert.Equal(t, "2000", values[0].AccountCode)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Equal(t, "1000", values[2].AccountCode)
	require.Error(t, errs[3])
	assert.True(t, errortypes.IsNotFoundError(errs[3]))
	require.NoError(t, errs[4])
	assert.Equal(t, "2000", values[4].AccountCode)
}

func TestGLAccountBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	accountID := pulid.MustNew("gla_")
	repoErr := errors.New("repository failed")
	repo := &stubGLAccountRepository{
		getByIDs: func(context.Context, repositories.GetGLAccountsByIDsRequest) ([]*glaccount.GLAccount, error) {
			return nil, repoErr
		},
	}
	factory := &GLAccountByIDLoaderFactory{glAccountRepo: repo}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		accountID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], repoErr)
}

type stubGLAccountRepository struct {
	getByIDs func(context.Context, repositories.GetGLAccountsByIDsRequest) ([]*glaccount.GLAccount, error)
}

func (s *stubGLAccountRepository) List(
	context.Context,
	*repositories.ListGLAccountsRequest,
) (*pagination.ListResult[*glaccount.GLAccount], error) {
	return nil, nil
}

func (s *stubGLAccountRepository) GetByID(
	context.Context,
	repositories.GetGLAccountByIDRequest,
) (*glaccount.GLAccount, error) {
	return nil, nil
}

func (s *stubGLAccountRepository) GetByIDs(
	ctx context.Context,
	req repositories.GetGLAccountsByIDsRequest,
) ([]*glaccount.GLAccount, error) {
	return s.getByIDs(ctx, req)
}

func (s *stubGLAccountRepository) Create(
	context.Context,
	*glaccount.GLAccount,
) (*glaccount.GLAccount, error) {
	return nil, nil
}

func (s *stubGLAccountRepository) Update(
	context.Context,
	*glaccount.GLAccount,
) (*glaccount.GLAccount, error) {
	return nil, nil
}

func (s *stubGLAccountRepository) BulkUpdateStatus(
	context.Context,
	*repositories.BulkUpdateGLAccountStatusRequest,
) ([]*glaccount.GLAccount, error) {
	return nil, nil
}

func (s *stubGLAccountRepository) SelectOptions(
	context.Context,
	*repositories.GLAccountSelectOptionsRequest,
) (*pagination.ListResult[*glaccount.GLAccount], error) {
	return nil, nil
}

func (s *stubGLAccountRepository) Delete(
	context.Context,
	repositories.DeleteGLAccountRequest,
) error {
	return nil
}
