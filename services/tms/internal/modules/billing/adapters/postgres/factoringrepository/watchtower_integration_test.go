//go:build integration

package factoringrepository

import (
	"testing"
	"time"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func (e *factoringEnv) watchtowerReader() factoringports.WatchtowerReader {
	return NewWatchtowerReader(Params{DB: postgres.NewTestConnection(e.db), Logger: zap.NewNop()})
}

func (e *factoringEnv) nameFactorOnProfile(t *testing.T, factorID *pulid.ID) {
	t.Helper()
	cbp := customerdb.CustomerBillingProfiles
	_, err := cbp.Where(cbp.CustomerID.Eq(e.customerID)).
		Update(e.ctx, e.db, cbp.FactoringCompanyID.ToExpr(keel.Val(factorID)))
	require.NoError(t, err)
}

func (e *factoringEnv) setNoaDates(t *testing.T, id pulid.ID, acknowledgedAt, expiresAt int64) {
	t.Helper()
	noa := factoringdb.Noas
	_, err := noa.Where(noa.ID.Eq(id)).
		Update(e.ctx, e.db, noa.AcknowledgedAt.To(acknowledgedAt), noa.ExpiresAt.To(expiresAt))
	require.NoError(t, err)
}

func missingFor(
	rows []*factoringports.CustomerMissingNoa,
	customerID, factorID pulid.ID,
) *factoringports.CustomerMissingNoa {
	for _, row := range rows {
		if row.CustomerID == customerID && row.FactoringCompanyID == factorID {
			return row
		}
	}
	return nil
}

func expiringFor(rows []*factoringports.ExpiringNoa, id pulid.ID) *factoringports.ExpiringNoa {
	for _, row := range rows {
		if row.NoaID == id {
			return row
		}
	}
	return nil
}

func TestWatchtowerReaderFollowsTheNoticeOfAssignment(t *testing.T) {
	env := setupFactoringEnv(t)
	factor := env.createFactor(t, "WTCH")
	reader := env.watchtowerReader()
	now := time.Now().Unix()
	window := now + int64(factoring.NoaExpiryWarningDays)*86400

	listMissing := func() []*factoringports.CustomerMissingNoa {
		rows, err := reader.ListCustomersMissingNoa(
			env.ctx,
			&factoringports.ListCustomersMissingNoaRequest{TenantInfo: env.tenantInfo, At: now},
		)
		require.NoError(t, err)
		return rows
	}
	listExpiring := func() []*factoringports.ExpiringNoa {
		rows, err := reader.ListNoasExpiringSoon(
			env.ctx,
			&factoringports.ListNoasExpiringSoonRequest{
				TenantInfo: env.tenantInfo,
				At:         now,
				Before:     window,
			},
		)
		require.NoError(t, err)
		return rows
	}

	assert.Nil(t, missingFor(listMissing(), env.customerID, factor.ID),
		"a customer whose profile names no factor needs no notice")

	env.nameFactorOnProfile(t, &factor.ID)

	row := missingFor(listMissing(), env.customerID, factor.ID)
	require.NotNil(t, row, "the profile names a factor with no notice")
	assert.True(t, row.NoaID.IsNil())
	assert.Equal(t, factor.Name, row.FactorName)
	assert.NotEmpty(t, row.CustomerName)
	assert.Empty(t, row.NoaStatus)

	noa, err := env.noas.Create(env.ctx, &factoring.Noa{
		OrganizationID:     env.tenantInfo.OrgID,
		BusinessUnitID:     env.tenantInfo.BuID,
		CustomerID:         env.customerID,
		FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)

	row = missingFor(listMissing(), env.customerID, factor.ID)
	require.NotNil(t, row, "a drafted notice is not a signed one")
	assert.Equal(t, noa.ID, row.NoaID)
	assert.Equal(t, factoring.NoaStatusDraft, row.NoaStatus)
	require.NotNil(t, row.NoaCreatedAt)

	far := now + 365*86400
	noa.Status = factoring.NoaStatusAcknowledged
	noa.AcknowledgedAt = new(now - 86400)
	noa.SignedDocumentID = pulid.MustNew("doc_")
	noa.ExpiresAt = &far
	noa, err = env.noas.Update(env.ctx, noa)
	require.NoError(t, err)

	assert.Nil(t, missingFor(listMissing(), env.customerID, factor.ID),
		"a signed notice that has not expired covers the customer")
	assert.Nil(t, expiringFor(listExpiring(), noa.ID), "a year out is not expiring")

	env.setNoaDates(t, noa.ID, now-86400, now+10*86400)
	expiring := expiringFor(listExpiring(), noa.ID)
	require.NotNil(t, expiring, "ten days out is inside the warning window")
	assert.Equal(t, now+10*86400, expiring.ExpiresAt)
	assert.Equal(t, env.customerID, expiring.CustomerID)
	assert.Equal(t, factor.Name, expiring.FactorName)
	assert.Nil(t, missingFor(listMissing(), env.customerID, factor.ID))

	env.setNoaDates(t, noa.ID, now-100*86400, now-86400)
	assert.Nil(t, expiringFor(listExpiring(), noa.ID), "an expired notice is no longer expiring")
	row = missingFor(listMissing(), env.customerID, factor.ID)
	require.NotNil(t, row, "an expired notice leaves the customer without one")
	assert.Equal(t, factoring.NoaStatusAcknowledged, row.NoaStatus)
	require.NotNil(t, row.NoaExpiresAt)
	assert.Equal(t, now-86400, *row.NoaExpiresAt)

	other := env.createFactor(t, "WTCX")
	env.nameFactorOnProfile(t, &other.ID)
	assert.Nil(t, missingFor(listMissing(), env.customerID, factor.ID),
		"a notice for a factor the profile no longer names is not asked for")
	require.NotNil(t, missingFor(listMissing(), env.customerID, other.ID))
}

func TestWatchtowerReaderLabelsInvoices(t *testing.T) {
	env := setupFactoringEnv(t)
	factor := env.createFactor(t, "LBL")
	reader := env.watchtowerReader()

	factored := env.postInvoice(t, 75000, &factor.ID)
	unfactored := env.postInvoice(t, 15000, nil)

	empty, err := reader.ListInvoiceLabels(env.ctx, &factoringports.ListInvoiceLabelsRequest{
		TenantInfo: env.tenantInfo,
	})
	require.NoError(t, err)
	assert.Empty(t, empty)

	labels, err := reader.ListInvoiceLabels(env.ctx, &factoringports.ListInvoiceLabelsRequest{
		TenantInfo: env.tenantInfo,
		InvoiceIDs: []pulid.ID{factored.ID, unfactored.ID, pulid.MustNew("inv_")},
	})
	require.NoError(t, err)
	require.Len(t, labels, 2)

	label := labels[factored.ID]
	require.NotNil(t, label)
	assert.Equal(t, factored.Number, label.Number)
	assert.Equal(t, "USD", label.CurrencyCode)
	assert.Equal(t, factor.Name, label.FactorName)
	assert.NotEmpty(t, label.CustomerName)

	require.NotNil(t, labels[unfactored.ID])
	assert.Empty(t, labels[unfactored.ID].FactorName)
}
