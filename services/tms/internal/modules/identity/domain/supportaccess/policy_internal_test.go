package supportaccess

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryLimitTheGrantsPageListsIsASentence(t *testing.T) {
	t.Parallel()

	limits := Limits()
	require.NotEmpty(t, limits)
	for _, limit := range limits {
		assert.NotEmpty(t, limit.Label(), "limit %d has no sentence for the grants page", limit)
	}
	for resource, r := range denied {
		assert.Contains(t, limits, r.limit, "%s is denied but the grants page would not say so", resource)
	}
	assert.Contains(t, limits, LimitPaymentDestinations)
	assert.Contains(t, limits, LimitExport)
	assert.Len(t, limitLabels, len(limits), "a sentence no rule stands behind would promise what nothing enforces")
}

func TestOneLimitIsOneKindOfDenial(t *testing.T) {
	t.Parallel()

	kinds := map[Limit]denial{}
	for resource, r := range denied {
		if held, ok := kinds[r.limit]; ok {
			assert.Equal(t, held, r.denial,
				"%s shares a sentence with a resource denied differently, so the sentence would be wrong for one", resource)
			continue
		}
		kinds[r.limit] = r.denial
	}
}

func TestEveryPaymentDestinationNamesColumnsOnce(t *testing.T) {
	t.Parallel()

	tables := map[string]bool{}
	for _, d := range PaymentDestinations() {
		assert.False(t, tables[d.Table], "%s is listed twice; the database holds one trigger per table", d.Table)
		tables[d.Table] = true
		require.NotEmpty(t, d.Columns, d.Table)
		sorted := slices.Clone(d.Columns)
		slices.Sort(sorted)
		assert.Len(t, slices.Compact(sorted), len(d.Columns), "%s names a column twice", d.Table)
		assert.NotZero(t, d.Limit.Label(), d.Table)
	}
	for _, table := range []string{"invoices", "billing_controls", "organizations", "carriers", "customer_billing_profiles", "factoring_companies"} {
		assert.True(t, tables[table], "%s holds where money is sent", table)
	}
}

func TestPaymentDestinationsCannotBeChangedThroughTheCopy(t *testing.T) {
	t.Parallel()

	copied := PaymentDestinations()
	copied[0].Columns[0] = "memo"
	assert.Equal(t, "remittance_instructions", paymentDestinations[0].Columns[0])
}

func TestNoGrantSendsToTheCustomersCustomers(t *testing.T) {
	t.Parallel()

	for _, scope := range []Scope{ScopeReadOnly, ScopeFull} {
		assert.False(t, Allows(scope, permission.ResourceCustomerCommunication.String(), permission.OpCreate))
		assert.True(t, Allows(scope, permission.ResourceCustomerCommunication.String(), permission.OpRead),
			"support still reads what was sent")
	}
	assert.Contains(t, Limits(), LimitOutbound)
}

func TestEveryOutboundGuardSaysWhatItRefuses(t *testing.T) {
	t.Parallel()

	tables := map[string]bool{}
	triggers := map[string]bool{}
	for _, g := range OutboundGuards() {
		key := g.Table + "/" + g.WhenColumn
		assert.False(t, triggers[key], "%s is listed twice; the database holds one trigger per table and condition", key)
		triggers[key] = true
		tables[g.Table] = true
		assert.True(t, g.OnInsert || g.OnUpdate, "%s guards no write", g.Table)
		sorted := slices.Clone(g.Columns)
		slices.Sort(sorted)
		assert.Len(t, slices.Compact(sorted), len(g.Columns), "%s names a column twice", g.Table)
	}
	for _, table := range []string{
		"email_messages", "edi_messages", "invoice_document_share_tokens", "customer_tracking_links",
		"report_schedules", "invoices", "customer_email_profiles", "invoice_lines", "order_charges",
	} {
		assert.True(t, tables[table], "%s carries what leaves the organization", table)
	}
}

func TestNoScopeHoldsTheAssistantByItself(t *testing.T) {
	t.Parallel()

	for _, scope := range []Scope{ScopeReadOnly, ScopeFull} {
		for _, op := range []permission.Operation{permission.OpRead, permission.OpCreate, permission.OpDelete} {
			assert.False(t, Allows(scope, permission.ResourceAssistant.String(), op),
				"a %s grant may not %s the assistant unless the grant allows it", scope, op)
		}
	}
	assert.ElementsMatch(t, []permission.Operation{permission.OpRead, permission.OpCreate}, AssistantOperations)
}
