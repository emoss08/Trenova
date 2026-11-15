package responseguard

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type unreadablePlan struct{}

func (unreadablePlan) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, errors.New("the plan is unavailable")
}

type planned struct {
	body   map[string]any
	access string
	byPlan string
}

func servePlanned(
	t *testing.T,
	registry *permissioncatalog.Registry,
	gate platformcatalogports.FeatureGate,
	value any,
) planned {
	t.Helper()

	rec := serveWith(t, registry, gate, nil, value, guarded)
	out := planned{
		body:   map[string]any{},
		access: rec.Header().Get(WithheldHeader),
		byPlan: rec.Header().Get(WithheldByPlanHeader),
	}
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &out.body))

	return out
}

func factoredInvoice() *invoice.Invoice {
	return &invoice.Invoice{
		ID:       pulid.MustNew("inv_"),
		Number:   "INV-1",
		Customer: client(),
		FactoringCompany: &factoringcompany.FactoringCompany{
			ID:          pulid.MustNew("fc_"),
			Code:        "FAC",
			Name:        "Fast Factor",
			Email:       "remit@factor.example",
			PhoneNumber: "800-555-0101",
		},
	}
}

func featureOf(t *testing.T, resource permission.Resource) platformcatalog.FeatureKey {
	t.Helper()

	feature, ok := fieldsensitivity.DefaultRegistry().FeatureOf(resource.String())
	require.True(t, ok)
	return feature
}

func TestALinkedRecordOfAKindOffThePlanIsWithheldAndNamedWithItsFeature(t *testing.T) {
	t.Parallel()

	feature := featureOf(t, permission.ResourceFactoringCompany)
	got := servePlanned(t, fieldsensitivity.DefaultRegistry(), plantest.Without(feature),
		factoredInvoice())

	factor := object(t, got.body, "factoringCompany")
	assert.NotEmpty(t, factor["id"], "a withheld record keeps its id")
	for _, field := range []string{"code", "name", "email", "phoneNumber"} {
		value, present := factor[field]
		assert.Truef(t, present, "%s should be answered, as null", field)
		assert.Nilf(t, value, "%s belongs to a kind the plan leaves out, view or not", field)
	}
	assert.Equal(t, "INV-1", got.body["number"])
	assert.Equal(t, "Acme Foods", object(t, got.body, "customer")["name"])
	assert.Equal(
		t,
		"factoringCompany="+string(feature)+
			", factoringOverride="+string(feature)+
			", factoringOverrideReason="+string(feature),
		got.byPlan,
	)
	assert.Empty(t, got.access, "a plan withholding is not an access withholding")
}

// moveRegistry has a base shipment whose moves are a part of its family that
// belongs to an add-on, under a base invoice.
func moveRegistry(t *testing.T) *permissioncatalog.Registry {
	t.Helper()

	registry := permissioncatalog.NewEmptyRegistry()
	for _, def := range []*permissionspec.ResourceDefinition{
		{
			Resource: permission.ResourceInvoice.String(),
			Feature:  platformcatalog.FeatureBilling,
			FieldFeatures: map[string]platformcatalog.FeatureKey{
				"factoringCompany": platformcatalog.FeatureFactoring,
			},
		},
		{
			Resource:      permission.ResourceShipment.String(),
			Feature:       platformcatalog.FeatureCoreTMS,
			ReferenceView: []string{"proNumber"},
		},
		{
			Resource:       permission.ResourceShipmentMove.String(),
			Feature:        platformcatalog.FeatureDetention,
			ParentResource: permission.ResourceShipment.String(),
		},
	} {
		require.NoError(t, registry.Register(def))
	}

	return registry
}

func TestAPartOfAKindOffThePlanIsWithheldUnderABaseCarrier(t *testing.T) {
	t.Parallel()

	inv := &invoice.Invoice{
		ID:     pulid.MustNew("inv_"),
		Number: "INV-2",
		Shipment: &shipment.Shipment{
			ID:        pulid.MustNew("shp_"),
			ProNumber: "PRO-7",
			Moves: []*shipment.ShipmentMove{{
				ID:       pulid.MustNew("smv_"),
				Status:   "Completed",
				Sequence: 1,
			}},
		},
	}

	got := servePlanned(t, moveRegistry(t), plantest.Without(platformcatalog.FeatureDetention),
		inv)

	shp := object(t, got.body, "shipment")
	assert.Equal(t, "PRO-7", shp["proNumber"], "the base carrier stays")
	move := object(t, got.body, "shipment", "moves", 0)
	assert.NotEmpty(t, move["id"])
	assert.Nil(t, move["status"])
	assert.Nil(t, move["sequence"])
	assert.Equal(t, "shipment.moves="+string(platformcatalog.FeatureDetention), got.byPlan)
	assert.Empty(t, got.access)
}

func TestAFieldOfAFeatureOffThePlanIsWithheldOnTheRootRecord(t *testing.T) {
	t.Parallel()

	inv := factoredInvoice()
	inv.Customer = nil
	got := servePlanned(t, moveRegistry(t), plantest.Without(platformcatalog.FeatureFactoring),
		inv)

	value, present := got.body["factoringCompany"]
	assert.True(t, present)
	assert.Nil(t, value, "the invoice's own field belongs to factoring")
	assert.Equal(t, "INV-1", got.body["number"])
	assert.Equal(t, "factoringCompany="+string(platformcatalog.FeatureFactoring), got.byPlan)
	assert.Empty(t, got.access)
}

func TestEveryFeatureOnLeavesTheResponseAsItWas(t *testing.T) {
	t.Parallel()

	got := servePlanned(t, fieldsensitivity.DefaultRegistry(), plantest.AllFeatures(),
		factoredInvoice())

	factor := object(t, got.body, "factoringCompany")
	assert.Equal(t, "Fast Factor", factor["name"])
	assert.Equal(t, "remit@factor.example", factor["email"])
	assert.Empty(t, got.byPlan)
	assert.Empty(t, got.access)

	inv := factoredInvoice()
	inv.Customer = nil
	got = servePlanned(t, moveRegistry(t), plantest.AllFeatures(), inv)
	assert.Equal(t, "Fast Factor", object(t, got.body, "factoringCompany")["name"])
	assert.Empty(t, got.byPlan)
}

func TestAPlanThatCannotBeReadWithholdsAsAccess(t *testing.T) {
	t.Parallel()

	got := servePlanned(t, fieldsensitivity.DefaultRegistry(), unreadablePlan{},
		factoredInvoice())

	assert.Nil(t, object(t, got.body, "factoringCompany")["name"])
	assert.Empty(t, got.byPlan, "a plan that could not be read is never named as the reason")
	assert.Contains(t, got.access, "factoringCompany")
}
