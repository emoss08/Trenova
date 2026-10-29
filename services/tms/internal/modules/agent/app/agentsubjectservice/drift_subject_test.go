package agentsubjectservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeDriftRepo struct {
	accountingdriftports.AccountingDriftFindingRepository

	found  *accountingdrift.AccountingDriftFinding
	lastID pulid.ID
}

func (f *fakeDriftRepo) GetByID(
	_ context.Context,
	req accountingdriftports.GetAccountingDriftFindingQuery,
) (*accountingdrift.AccountingDriftFinding, error) {
	f.lastID = req.ID

	return f.found, nil
}

func TestService_DescribesADriftFindingWithBothValuesAndItsFixes(t *testing.T) {
	t.Parallel()

	trenova, provider := int64(125_000), int64(120_000)
	found := accountingdrift.NewAccountingDriftFinding(&accountingdrift.DriftObservation{
		ConnectionID:       pulid.MustNew("acctc_"),
		ObjectType:         accountingsync.SyncObjectInvoice,
		ObjectID:           pulid.MustNew("inv_"),
		ObjectNumber:       "INV-1042",
		PartyName:          "Acme Foods",
		Kind:               accountingdrift.DriftAmountMismatch,
		CurrencyCode:       "USD",
		TrenovaMinor:       &trenova,
		ProviderMinor:      &provider,
		ProviderModifiedBy: "Pat Bookkeeper",
		At:                 1_700_000_000,
	})
	repo := &fakeDriftRepo{found: found}
	subjects := &Service{accountingDrift: repo, logger: zap.NewNop()}

	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectAccountingDrift,
		found.ID,
	)
	require.NoError(t, err)

	assert.Equal(t, found.ID, repo.lastID)
	assert.Equal(t, "Books differ from Trenova: INV-1042 Acme Foods", subject.Label)
	assert.Contains(t, subject.Notes, "AmountMismatch")
	assert.Contains(t, subject.Notes, "125000")
	assert.Contains(t, subject.Notes, "120000")
	assert.Contains(t, subject.Notes, "-5000")
	assert.Contains(t, subject.Notes, "Pat Bookkeeper")
	assert.Contains(t, subject.Notes, "AdjustTrenova")
}
