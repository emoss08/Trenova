package customerpaymentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestListDelegatesToRepository(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomerPaymentRepository(t)
	svc := &Service{repo: repo}
	req := &repositories.ListCustomerPaymentsRequest{
		Filter: &pagination.QueryOptions{Pagination: pagination.Info{Limit: 10}},
	}
	expected := &pagination.ListResult[*customerpayment.Payment]{
		Items: []*customerpayment.Payment{{}},
		Total: 1,
	}

	repo.EXPECT().List(t.Context(), req).Return(expected, nil).Once()

	result, err := svc.List(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, expected, result)
}
