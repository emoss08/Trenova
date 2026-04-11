package accountsreceivableservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountsreceivable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAgingSummaryAggregatesTotals(t *testing.T) {
	t.Parallel()

	repo := fakeARRepo{rows: []*accountsreceivable.CustomerAgingRow{{CustomerID: pulid.MustNew("cus_"), Buckets: accountsreceivable.AgingBucketTotals{CurrentMinor: 100, Days1To30Minor: 200, TotalOpenMinor: 300}}, {CustomerID: pulid.MustNew("cus_"), Buckets: accountsreceivable.AgingBucketTotals{Days31To60Minor: 400, TotalOpenMinor: 400}}}}
	svc := &Service{repo: repo}

	summary, err := svc.GetAgingSummary(t.Context(), pagination.TenantInfo{}, 123)

	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, int64(100), summary.Totals.CurrentMinor)
	assert.Equal(t, int64(200), summary.Totals.Days1To30Minor)
	assert.Equal(t, int64(400), summary.Totals.Days31To60Minor)
	assert.Equal(t, int64(700), summary.Totals.TotalOpenMinor)
}

type fakeARRepo struct {
	ledger []*accountsreceivable.LedgerEntry
	rows   []*accountsreceivable.CustomerAgingRow
}

func (f fakeARRepo) ListCustomerLedger(context.Context, repositories.ListCustomerLedgerRequest) ([]*accountsreceivable.LedgerEntry, error) {
	return f.ledger, nil
}

func (f fakeARRepo) ListARAging(context.Context, repositories.ListARAgingRequest) ([]*accountsreceivable.CustomerAgingRow, error) {
	return f.rows, nil
}
