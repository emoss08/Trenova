package formula_test

import (
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingObserver struct {
	mu    sync.Mutex
	calls map[string]int
}

func (o *countingObserver) CallFinished(op, _ string, _ time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls[op]++
}

func (*countingObserver) InstanceAcquired()         {}
func (*countingObserver) InstanceReleased()         {}
func (*countingObserver) InstanceDiscarded(string) {}

func (o *countingObserver) snapshot() map[string]int {
	o.mu.Lock()
	defer o.mu.Unlock()
	copied := make(map[string]int, len(o.calls))
	for op, count := range o.calls {
		copied[op] = count
	}
	return copied
}

func observeModuleCalls(t *testing.T) *countingObserver {
	t.Helper()

	_, err := rules.Catalog(t.Context())
	require.NoError(t, err)

	observer := &countingObserver{calls: map[string]int{}}
	rules.Default().Instrument(nil, observer)
	t.Cleanup(func() { rules.Default().Instrument(nil, nil) })
	return observer
}

func TestRate_OneModuleCallPricesTheExpressionAndEveryBreakdownLine(t *testing.T) {
	svc := setupServiceWithMatrixData(t, freightClassMatrix())
	observer := observeModuleCalls(t)

	template := activeTemplate(`lookup("class_rates", "100") * 10 + totalStops`)
	template.OrganizationID = testTenant().OrgID
	template.BusinessUnitID = testTenant().BuID
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "linehaul", Label: "Linehaul", Expression: `lookup("class_rates", "100") * 10`},
		{Name: "stops", Label: "Stops", Expression: `totalStops`},
		{Name: "broken", Label: "Broken", Expression: `1 / (totalStops - totalStops)`},
	}

	entity := conformanceShipment()
	entity.Customer = &customer.Customer{Code: "ACME", Name: "Acme"}

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   entity,
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]int{rules.OpFormulaEvaluate: 1}, observer.snapshot(),
		"the main expression and every line are one evaluate call, and nothing is checked")
	assert.Equal(t, "317", resp.Amount.String())
	require.Len(t, resp.Breakdown, 3)
	assert.Equal(t, "315", resp.Breakdown[0].Amount.String())
	assert.Equal(t, "2", resp.Breakdown[1].Amount.String())
	assert.NotEmpty(t, resp.Breakdown[2].Error, "a failing line fails only itself")

	scopes := make([]string, 0, len(resp.Receipt.Lookups))
	for _, lookup := range resp.Receipt.Lookups {
		scopes = append(scopes, lookup.Scope)
	}
	assert.Equal(t, []string{formula.MainScope, "linehaul"}, scopes,
		"the receipt attributes each lookup to the line that made it")
}

func TestEvaluatePredicates_EveryConditionIsOneModuleCall(t *testing.T) {
	svc := setupService(t)
	observer := observeModuleCalls(t)

	outcomes, err := svc.EvaluatePredicates(
		t.Context(),
		&formulatemplateports.EvaluatePredicatesRequest{
			Expressions: []string{"totalStops > 1", "hasHazmat", `lookup("x", 1) == 0`},
			SchemaID:    "shipment",
			Entity:      conformanceShipment(),
			TenantInfo:  testTenant(),
		},
	)
	require.NoError(t, err)
	require.Len(t, outcomes, 3)
	assert.True(t, outcomes[0].Holds)
	assert.False(t, outcomes[1].Holds)
	require.NoError(t, outcomes[2].Err)
	assert.True(t, outcomes[2].Holds, "a predicate reads every rate table as zero")
	assert.Equal(t, map[string]int{rules.OpFormulaEvaluate: 1}, observer.snapshot())
}
