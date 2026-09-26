package accountingconnlookup

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeConnections struct {
	repositories.AccountingConnectionRepository
	conn *accountingsync.AccountingConnection
	err  error
	asks []repositories.GetAccountingConnectionRequest
}

func (f *fakeConnections) GetByType(
	_ context.Context,
	req repositories.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	f.asks = append(f.asks, req)
	return f.conn, f.err
}

func TestByTypeReadsTheTenantsConnection(t *testing.T) {
	t.Parallel()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	conn := &accountingsync.AccountingConnection{ID: pulid.MustNew("acctc_")}
	repo := &fakeConnections{conn: conn}

	found, err := ByType(t.Context(), repo, tenant, integration.TypeQuickBooksOnline)

	require.NoError(t, err)
	assert.Same(t, conn, found)
	require.Len(t, repo.asks, 1)
	assert.Equal(t, tenant, repo.asks[0].TenantInfo)
	assert.Equal(t, integration.TypeQuickBooksOnline, repo.asks[0].IntegrationType)
}

func TestByTypeRefusesWhatIsNotAnAccountingSystem(t *testing.T) {
	t.Parallel()
	repo := &fakeConnections{}

	_, err := ByType(t.Context(), repo, pagination.TenantInfo{}, integration.Type("Samsara"))

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "integrationType", validation.Field)
	assert.Empty(t, repo.asks)
}

func TestByTypeSaysTheSystemIsNotConnected(t *testing.T) {
	t.Parallel()
	repo := &fakeConnections{err: errortypes.NewNotFoundError("Accounting connection not found")}

	_, err := ByType(t.Context(), repo, pagination.TenantInfo{}, integration.TypeQuickBooksOnline)

	require.True(t, errortypes.IsNotFoundError(err))
	assert.Contains(t, err.Error(), "has not been connected yet")

	failure := errors.New("connection refused")
	repo.err = failure
	_, err = ByType(t.Context(), repo, pagination.TenantInfo{}, integration.TypeQuickBooksOnline)
	assert.ErrorIs(t, err, failure)
}
