package shipmenteffects

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/planfields"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func effectsOnPlan(gate platformcatalogports.FeatureGate) *Effects {
	return effectsWithTemplates(gate, plantest.AllPricingTemplates())
}

func effectsWithTemplates(
	gate platformcatalogports.FeatureGate,
	templates permissionports.PricingTemplatePlan,
) *Effects {
	registry := permissioncatalog.Default()
	return &Effects{
		planFields:       planfields.New(registry, onplan.New(registry, gate)),
		pricingTemplates: templates,
	}
}

type shipmentPair struct {
	stored, sent    *shipment.Shipment
	newStop         *shipment.Stop
	formulaTemplate pulid.ID
}

// storedAndSent is a shipment rated by a tariff, tendered, with a stop whose
// detention a person turned off and a detention charge; and the same
// shipment as a base-only form sends it back: each of those read null and
// came back empty, the PRO number changed and a stop was added.
func storedAndSent() shipmentPair {
	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	chargeID := pulid.MustNew("ac_")
	template := pulid.MustNew("ft_")
	tender := shipmentvocab.TenderStatusAccepted

	stored := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		ProNumber:         "PRO-1",
		FormulaTemplateID: template,
		TariffInputs:      tarifftypes.Inputs{"miles": rules.RawJSON(`"412"`)},
		TenderStatus:      &tender,
		Moves: []*shipment.ShipmentMove{{
			ID: moveID,
			Stops: []*shipment.Stop{{
				ID:                     stopID,
				CountDetentionOverride: new(false),
			}},
		}},
		AdditionalCharges: []*shipment.AdditionalCharge{{ID: chargeID, IsDetention: true}},
	}

	newStop := &shipment.Stop{CountDetentionOverride: new(true)}
	sent := &shipment.Shipment{
		ID:        stored.ID,
		ProNumber: "PRO-2",
		Moves: []*shipment.ShipmentMove{{
			ID:    moveID,
			Stops: []*shipment.Stop{{ID: stopID}, newStop},
		}},
		AdditionalCharges: []*shipment.AdditionalCharge{{ID: chargeID}},
	}

	return shipmentPair{
		stored:          stored,
		sent:            sent,
		newStop:         newStop,
		formulaTemplate: template,
	}
}

func TestKeepPlanFields_ABaseOnlyUpdateKeepsTheStoredAddOnValues(t *testing.T) {
	t.Parallel()

	pair := storedAndSent()
	err := effectsOnPlan(plantest.BaseOnly()).KeepPlanFields(t.Context(), pair.sent, pair.stored)
	require.NoError(t, err)

	sent := pair.sent
	assert.Equal(t, "PRO-2", sent.ProNumber, "the base change is written")
	assert.True(t, sent.FormulaTemplateID.IsNil(), "the rating method is base, so clearing it is written")
	require.NotNil(t, sent.TenderStatus)
	assert.Equal(t, shipmentvocab.TenderStatusAccepted, *sent.TenderStatus)
	stop := sent.Moves[0].Stops[0]
	require.NotNil(t, stop.CountDetentionOverride)
	assert.False(t, *stop.CountDetentionOverride, "the stop keeps its stored override")
	assert.Nil(t, pair.newStop.CountDetentionOverride, "a new stop takes nothing Detention owns")
	assert.True(t, sent.AdditionalCharges[0].IsDetention)
}

func TestKeepPlanFields_WithTheAddOnsTheUpdateWritesTheSentValues(t *testing.T) {
	t.Parallel()

	pair := storedAndSent()
	err := effectsOnPlan(plantest.AllFeatures()).
		KeepPlanFields(t.Context(), pair.sent, pair.stored)
	require.NoError(t, err)

	sent := pair.sent
	assert.True(t, sent.FormulaTemplateID.IsNil())
	assert.Empty(t, sent.TariffInputs)
	assert.Nil(t, sent.TenderStatus)
	assert.Nil(t, sent.Moves[0].Stops[0].CountDetentionOverride)
	require.NotNil(t, pair.newStop.CountDetentionOverride)
	assert.True(t, *pair.newStop.CountDetentionOverride)
	assert.False(t, sent.AdditionalCharges[0].IsDetention)
}

func TestKeepPlanFields_ABaseOnlyCreateTakesNoAddOnValue(t *testing.T) {
	t.Parallel()

	pair := storedAndSent()
	created := pair.stored
	err := effectsOnPlan(plantest.BaseOnly()).KeepPlanFields(t.Context(), created, nil)
	require.NoError(t, err)

	assert.Equal(t, pair.formulaTemplate, created.FormulaTemplateID, "a standard template is base")
	assert.Equal(t, tarifftypes.Inputs{"miles": rules.RawJSON(`"412"`)}, created.TariffInputs)
	assert.Nil(t, created.TenderStatus)
	assert.Nil(t, created.Moves[0].Stops[0].CountDetentionOverride)
	assert.False(t, created.AdditionalCharges[0].IsDetention)
}

func TestKeepPlanFields_ABaseOnlyCreateCannotChooseTheOrganizationsOwnTemplate(t *testing.T) {
	t.Parallel()

	pair := storedAndSent()
	err := effectsWithTemplates(
		plantest.BaseOnly(),
		plantest.OwnTemplatesOffPlan(pair.formulaTemplate),
	).KeepPlanFields(t.Context(), pair.stored, nil)

	require.Error(t, err)
	assert.True(t, errortypes.IsPlanRestrictionError(err), "a plan restriction, got %v", err)
}

func TestKeepPlanFields_ABaseOnlyUpdateCannotSwitchToTheOrganizationsOwnTemplate(t *testing.T) {
	t.Parallel()

	pair := storedAndSent()
	own := pulid.MustNew("ft_")
	pair.sent.FormulaTemplateID = own

	err := effectsWithTemplates(
		plantest.BaseOnly(),
		plantest.OwnTemplatesOffPlan(own),
	).KeepPlanFields(t.Context(), pair.sent, pair.stored)

	require.Error(t, err)
	assert.True(t, errortypes.IsPlanRestrictionError(err), "a plan restriction, got %v", err)
}

func TestKeepPlanFields_ABaseOnlyUpdateKeepsTheOwnTemplateAndItsInputsAsEntered(t *testing.T) {
	t.Parallel()

	pair := storedAndSent()
	pair.sent.FormulaTemplateID = pair.formulaTemplate
	pair.sent.TariffInputs = tarifftypes.Inputs{"miles": rules.RawJSON(`"999"`)}

	err := effectsWithTemplates(
		plantest.BaseOnly(),
		plantest.OwnTemplatesOffPlan(pair.formulaTemplate),
	).KeepPlanFields(t.Context(), pair.sent, pair.stored)

	require.NoError(t, err, "a shipment entered under Rating stays editable")
	assert.Equal(t, pair.formulaTemplate, pair.sent.FormulaTemplateID)
	assert.Equal(t, pair.stored.TariffInputs, pair.sent.TariffInputs,
		"the inputs of a template the plan does not price keep their stored values")
}

func TestKeepPlanFields_ABaseOnlyUpdateMayChangeAStandardTemplatesInputs(t *testing.T) {
	t.Parallel()

	pair := storedAndSent()
	changed := tarifftypes.Inputs{"miles": rules.RawJSON(`"999"`)}
	pair.sent.FormulaTemplateID = pair.formulaTemplate
	pair.sent.TariffInputs = changed

	err := effectsOnPlan(plantest.BaseOnly()).KeepPlanFields(t.Context(), pair.sent, pair.stored)

	require.NoError(t, err)
	assert.Equal(t, changed, pair.sent.TariffInputs)
}
