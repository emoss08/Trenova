package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceBatchFunc_PreservesOrderAndReportsMissingIDs(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("inv_")
	secondID := pulid.MustNew("inv_")
	missingID := pulid.MustNew("inv_")
	getter := &stubInvoicesByIDsGetter{
		getByIDs: func(_ context.Context, req repositories.GetInvoicesByIDsRequest) ([]*invoice.Invoice, error) {
			assert.Equal(t, []pulid.ID{secondID, firstID, missingID}, req.InvoiceIDs)
			return []*invoice.Invoice{
				{ID: firstID, Number: "INV-1"},
				{ID: secondID, Number: "INV-2"},
			}, nil
		},
	}
	factory := &InvoiceByIDLoaderFactory{invoices: getter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		secondID.String(),
		"bad",
		firstID.String(),
		missingID.String(),
		secondID.String(),
	})

	require.Len(t, values, 5)
	require.NoError(t, errs[0])
	assert.Equal(t, "INV-2", values[0].Number)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Equal(t, "INV-1", values[2].Number)
	require.Error(t, errs[3])
	assert.True(t, errortypes.IsNotFoundError(errs[3]))
	require.NoError(t, errs[4])
	assert.Equal(t, "INV-2", values[4].Number)
}

func TestInvoiceBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	invoiceID := pulid.MustNew("inv_")
	repoErr := errors.New("repository failed")
	getter := &stubInvoicesByIDsGetter{
		getByIDs: func(context.Context, repositories.GetInvoicesByIDsRequest) ([]*invoice.Invoice, error) {
			return nil, repoErr
		},
	}
	factory := &InvoiceByIDLoaderFactory{invoices: getter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		invoiceID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], repoErr)
}

type stubInvoicesByIDsGetter struct {
	getByIDs func(context.Context, repositories.GetInvoicesByIDsRequest) ([]*invoice.Invoice, error)
}

func (s *stubInvoicesByIDsGetter) GetByIDs(
	ctx context.Context,
	req repositories.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	return s.getByIDs(ctx, req)
}
