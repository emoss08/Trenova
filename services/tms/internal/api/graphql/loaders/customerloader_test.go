package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerBatchFunc_PreservesOrderAndReportsMissingIDs(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("cus_")
	secondID := pulid.MustNew("cus_")
	missingID := pulid.MustNew("cus_")
	getter := &stubCustomersByIDsGetter{
		getByIDs: func(_ context.Context, req repositories.GetCustomersByIDsRequest) ([]*customer.Customer, error) {
			assert.Equal(t, []pulid.ID{secondID, firstID, missingID}, req.CustomerIDs)
			assert.True(t, req.IncludeState)
			return []*customer.Customer{
				{ID: firstID, Name: "First"},
				{ID: secondID, Name: "Second"},
			}, nil
		},
	}
	factory := &CustomerByIDLoaderFactory{customers: getter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		secondID.String(),
		"bad",
		firstID.String(),
		missingID.String(),
		secondID.String(),
	})

	require.Len(t, values, 5)
	require.NoError(t, errs[0])
	assert.Equal(t, "Second", values[0].Name)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Equal(t, "First", values[2].Name)
	require.Error(t, errs[3])
	assert.True(t, errortypes.IsNotFoundError(errs[3]))
	require.NoError(t, errs[4])
	assert.Equal(t, "Second", values[4].Name)
}

func TestCustomerBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	repoErr := errors.New("repository failed")
	getter := &stubCustomersByIDsGetter{
		getByIDs: func(context.Context, repositories.GetCustomersByIDsRequest) ([]*customer.Customer, error) {
			return nil, repoErr
		},
	}
	factory := &CustomerByIDLoaderFactory{customers: getter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		customerID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], repoErr)
}

type stubCustomersByIDsGetter struct {
	getByIDs func(context.Context, repositories.GetCustomersByIDsRequest) ([]*customer.Customer, error)
}

func (s *stubCustomersByIDsGetter) GetByIDs(
	ctx context.Context,
	req repositories.GetCustomersByIDsRequest,
) ([]*customer.Customer, error) {
	return s.getByIDs(ctx, req)
}
