package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiscalPeriodsBatchFunc_GroupsByFiscalYearAndDefaultsToEmpty(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("fy_")
	secondID := pulid.MustNew("fy_")
	lister := &stubFiscalPeriodsByFiscalYearIDsLister{
		list: func(_ context.Context, req repositories.ListByFiscalYearIDsRequest) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
			assert.Equal(t, []pulid.ID{firstID, secondID}, req.FiscalYearIDs)
			return map[pulid.ID][]*fiscalperiod.FiscalPeriod{
				firstID: {
					{FiscalYearID: firstID, PeriodNumber: 1},
					{FiscalYearID: firstID, PeriodNumber: 2},
				},
			}, nil
		},
	}
	factory := &FiscalPeriodsByFiscalYearIDLoaderFactory{periods: lister}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		firstID.String(),
		"bad",
		secondID.String(),
		firstID.String(),
	})

	require.Len(t, values, 4)
	require.NoError(t, errs[0])
	require.Len(t, values[0], 2)
	assert.EqualValues(t, 1, values[0][0].PeriodNumber)
	assert.EqualValues(t, 2, values[0][1].PeriodNumber)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Empty(t, values[2])
	assert.NotNil(t, values[2])
	require.NoError(t, errs[3])
	assert.Len(t, values[3], 2)
}

func TestFiscalPeriodsBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	fiscalYearID := pulid.MustNew("fy_")
	repoErr := errors.New("repository failed")
	lister := &stubFiscalPeriodsByFiscalYearIDsLister{
		list: func(context.Context, repositories.ListByFiscalYearIDsRequest) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
			return nil, repoErr
		},
	}
	factory := &FiscalPeriodsByFiscalYearIDLoaderFactory{periods: lister}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		fiscalYearID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], repoErr)
}

type stubFiscalPeriodsByFiscalYearIDsLister struct {
	list func(context.Context, repositories.ListByFiscalYearIDsRequest) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error)
}

func (s *stubFiscalPeriodsByFiscalYearIDsLister) ListPeriodsByFiscalYearIDs(
	ctx context.Context,
	req repositories.ListByFiscalYearIDsRequest,
) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
	return s.list(ctx, req)
}
