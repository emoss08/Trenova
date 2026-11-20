package toolaccess_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/accesstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	shipmentTool = "shipment_list"
	ledgerTool   = "accounting_list_gl_accounts"
	forecastTool = "dispatch_forecast_demand"
)

func planTools() []agenttoolports.AgentQueryTool {
	return []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: shipmentTool, Resource: permission.ResourceShipment},
		&agenttest.StubQueryTool{
			ToolName: ledgerTool,
			Resource: permission.ResourceGeneralLedgerAccount,
		},
		&agenttest.StubQueryTool{
			ToolName: forecastTool,
			Resource: permission.ResourceShipmentMove,
			Feature:  platformcatalog.FeatureDispatchIntelligence,
		},
	}
}

// accessOn is tool access under a plan, with the permission engine refusing
// the resources the plan leaves out, as the real engine does.
func accessOn(gate *plantest.Gate, off ...platformcatalog.FeatureKey) *toolaccess.Access {
	registry := permissioncatalog.Default()
	offPlan := make(map[string]permissionports.PlanDenial)
	for _, tool := range planTools() {
		resource := tool.Policy().Resource.String()
		feature, _ := registry.FeatureOf(resource)
		for _, key := range off {
			if feature == key {
				offPlan[resource] = permissionports.NewPlanDenial(key)
			}
		}
	}

	return accesstest.New(&toolaccess.AccessParams{
		QueryTools:  &agenttest.StubQueryRegistry{Tools: planTools()},
		Permissions: &agenttest.StubPermissions{OffPlan: offPlan},
		Registry:    registry,
		Plan:        onplan.New(registry, gate),
	})
}

func openSet(t *testing.T, access *toolaccess.Access) *toolaccess.ToolSet {
	t.Helper()

	return access.NewToolSet(
		t.Context(),
		&toolaccess.ToolSetRequest{
			Definition: runtimetest.Definition(shipmentTool, ledgerTool, forecastTool),
			Actor:      runtimetest.Actor(),
			Input:      "what is owed on the ledger",
		},
	)
}

func TestToolSet_WithoutAccountingOffersNoAccountingToolAndNamesTheAddOn(t *testing.T) {
	t.Parallel()

	access := accessOn(
		plantest.Without(platformcatalog.FeatureAccounting, platformcatalog.FeatureFactoring),
		platformcatalog.FeatureAccounting,
	)
	set := openSet(t, access)

	assert.Contains(t, set.Allowed, shipmentTool)
	assert.NotContains(t, set.Allowed, ledgerTool)
	assert.False(t, set.Offers(ledgerTool))
	require.Equal(t, []agentturn.PlanAddOn{{
		Name:     "Accounting & Cash",
		Features: []string{"Accounting", "Factoring"},
	}}, set.NotOnPlan)

	refusal := access.UnheldOutcome(set, ledgerTool)
	assert.Equal(t, aitrace.OutcomeDenied, refusal.Verdict)
	assert.Equal(t, "tms.accounting is not on the organization's plan", refusal.Reason)
	assert.Contains(t, refusal.Content, `Tool "`+ledgerTool+`" is not available`)
	assert.Contains(t, refusal.Content, "Accounting & Cash add-on, which is not on your plan")
	assert.NotContains(t, refusal.Content, "AI Control")
}

func TestToolSet_WithEveryFeatureOnIsUnchanged(t *testing.T) {
	t.Parallel()

	set := openSet(t, accessOn(plantest.AllFeatures()))

	assert.ElementsMatch(t, []string{shipmentTool, ledgerTool, forecastTool}, set.Allowed)
	assert.Empty(t, set.NotOnPlan)
}

// The forecast needs Dispatch Intelligence as well as the shipment-move
// access every dispatcher holds: off the plan it is neither offered nor run.
func TestForecast_NeedsDispatchIntelligenceAsWellAsItsResource(t *testing.T) {
	t.Parallel()

	access := accessOn(plantest.Without(platformcatalog.FeatureDispatchIntelligence))
	set := openSet(t, access)
	assert.NotContains(t, set.Allowed, forecastTool)

	refusal, refused := access.PlanRefusal(t.Context(), &agentruntimeports.RunRequest{
		Actor: runtimetest.Actor(),
	}, forecastTool)
	require.True(t, refused)
	assert.Contains(t, refusal.Content, "Dispatch Intelligence add-on")

	held := openSet(t, accessOn(plantest.AllFeatures()))
	assert.Contains(t, held.Allowed, forecastTool)
	_, refused = accessOn(plantest.AllFeatures()).PlanRefusal(t.Context(),
		&agentruntimeports.RunRequest{Actor: runtimetest.Actor()}, forecastTool)
	assert.False(t, refused)
}

func TestPlanAddOns_BoundsEachAddOnsLine(t *testing.T) {
	t.Parallel()

	var hr []permissionports.PlanDenial
	for _, feature := range []platformcatalog.FeatureKey{
		platformcatalog.FeatureWorkforceTimeOff,
		platformcatalog.FeatureWorkforceTimeTracking,
		platformcatalog.FeatureWorkforceTalent,
		platformcatalog.FeatureWorkforceBenefits,
		platformcatalog.FeatureWorkforceSelfService,
	} {
		hr = append(hr, permissionports.NewPlanDenial(feature))
	}
	hr = append(hr, permissionports.NewPlanDenial(platformcatalog.FeatureCoreTMS))

	addOns := toolaccess.PlanAddOns(hr)

	require.Len(t, addOns, 1, "a base feature has no add-on to name")
	assert.Equal(t, "HR Suite", addOns[0].Name)
	assert.Len(t, addOns[0].Features, toolaccess.MaxPlanFeatureNames)
	assert.Equal(t, 1, addOns[0].MoreFeatures)
}

// Differences with the books and the payments they recorded are the
// Accounting add-on's; the outbound queue is every plan's accounting
// integration. Without the add-on the agent works the queue but is offered
// nothing that reconciles.
func TestToolSet_WithoutAccountingOffersNoReconciliationToolButKeepsTheQueue(t *testing.T) {
	t.Parallel()

	const (
		driftTool   = "accounting_resolve_drift_finding"
		inboundTool = "accounting_apply_inbound_change"
		queueTool   = "accounting_retry_sync_record"
	)
	tools := []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{
			ToolName: driftTool,
			Resource: permission.ResourceAccountingReconciliation,
		},
		&agenttest.StubQueryTool{
			ToolName: inboundTool,
			Resource: permission.ResourceAccountingReconciliation,
		},
		&agenttest.StubQueryTool{ToolName: queueTool, Resource: permission.ResourceAccountingSync},
	}

	registry := permissioncatalog.Default()
	reconciliation, _ := registry.FeatureOf(permission.ResourceAccountingReconciliation.String())
	queue, _ := registry.FeatureOf(permission.ResourceAccountingSync.String())
	require.Equal(t, platformcatalog.FeatureAccounting, reconciliation)
	require.NotEqual(t, platformcatalog.FeatureAccounting, queue)

	access := accesstest.New(&toolaccess.AccessParams{
		QueryTools: &agenttest.StubQueryRegistry{Tools: tools},
		Permissions: &agenttest.StubPermissions{OffPlan: map[string]permissionports.PlanDenial{
			permission.ResourceAccountingReconciliation.String(): permissionports.NewPlanDenial(
				platformcatalog.FeatureAccounting,
			),
		}},
		Registry: registry,
		Plan:     onplan.New(registry, plantest.Without(platformcatalog.FeatureAccounting)),
	})
	set := access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(driftTool, inboundTool, queueTool),
		Actor:      runtimetest.Actor(),
		Input:      "fix the differences with the books and retry what failed to send",
	})

	assert.Contains(t, set.Allowed, queueTool)
	for _, name := range []string{driftTool, inboundTool} {
		assert.NotContains(t, set.Allowed, name)
		assert.False(t, set.Offers(name), name)
	}
}
