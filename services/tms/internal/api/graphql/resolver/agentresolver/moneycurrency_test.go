package agentresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type booksOf struct {
	tenantports.AccountingControlRepository

	currency string
	err      error
	asked    pulid.ID
}

func (b *booksOf) GetByOrgID(_ context.Context, orgID pulid.ID) (*tenant.AccountingControl, error) {
	b.asked = orgID
	if b.err != nil {
		return nil, b.err
	}

	return &tenant.AccountingControl{FunctionalCurrencyCode: b.currency}, nil
}

func TestAgentMoneyCurrency_IsTheOrganizationsFunctionalCurrency(t *testing.T) {
	t.Parallel()

	books := &booksOf{currency: " cad "}
	resolver := &AgentControlResolver{&Deps{AccountingControlRepo: books}}
	control := &tenant.AgentControl{OrganizationID: pulid.MustNew("org_")}

	currency, err := resolver.AgentMoneyCurrency(t.Context(), control)
	require.NoError(t, err)
	assert.Equal(t, "CAD", currency)
	assert.Equal(t, control.OrganizationID, books.asked)
}

func TestAgentMoneyCurrency_ReportsBooksThatCannotBeRead(t *testing.T) {
	t.Parallel()

	resolver := &AgentControlResolver{&Deps{AccountingControlRepo: &booksOf{err: errors.New("down")}}}

	_, err := resolver.AgentMoneyCurrency(t.Context(), &tenant.AgentControl{})
	require.Error(t, err, "a currency is never guessed")
}
