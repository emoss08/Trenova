package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryDefaultResourceDeclaresAFeatureTheCatalogKnows(t *testing.T) {
	t.Parallel()

	defs := Default().All()
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Resource)
	}
	require.Len(t, names, 199)

	for _, def := range defs {
		assert.NotEmpty(t, def.Feature, "resource %s declares no feature", def.Resource)
		_, known := platformcatalog.Lookup(def.Feature)
		assert.True(t, known, "resource %s declares unknown feature %q", def.Resource, def.Feature)
	}
}

func TestRegisterRefusesADefinitionWithoutAFeature(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	err := reg.Register(&permissionspec.ResourceDefinition{Resource: "featureless"})

	require.Error(t, err)
	assert.Equal(t, "resource featureless declares no feature", err.Error())
	assert.False(t, reg.HasResource("featureless"))
}

func TestRegisterRefusesADefinitionWithAnUnknownFeature(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	err := reg.Register(&permissionspec.ResourceDefinition{
		Resource: "mystery",
		Feature:  platformcatalog.FeatureKey("tms.nonexistent"),
	})

	require.Error(t, err)
	assert.Equal(t, `resource mystery declares unknown feature "tms.nonexistent"`, err.Error())
	assert.False(t, reg.HasResource("mystery"))
}

func TestFeatureOfReturnsTheFeatureOfAKnownResource(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	require.NoError(t, reg.Register(&permissionspec.ResourceDefinition{
		Resource: "probe",
		Feature:  platformcatalog.FeatureBilling,
	}))

	feature, ok := reg.FeatureOf("probe")
	assert.True(t, ok)
	assert.Equal(t, platformcatalog.FeatureBilling, feature)
}

func TestFeatureOfReportsAnUnknownResourceAsMissing(t *testing.T) {
	t.Parallel()

	feature, ok := NewEmptyRegistry().FeatureOf("not_a_resource")
	assert.False(t, ok)
	assert.Empty(t, feature)
}

func TestResourcesBelongToTheFeaturesTheCatalogueAssignsThem(t *testing.T) {
	t.Parallel()

	cases := []struct {
		resource permission.Resource
		feature  platformcatalog.FeatureKey
		why      string
	}{
		{permission.ResourceDriverPositioning, platformcatalog.FeatureDispatchIntelligence, "only the freight outlook sends a driver"},
		{permission.ResourceAIProvider, platformcatalog.FeatureAdministration, "base on purpose: Document Intelligence needs a provider"},
		{permission.ResourceAPIKey, platformcatalog.FeatureAPIKeys, "api keys are their own base feature"},
		{permission.ResourceGeneralLedgerAccount, platformcatalog.FeatureAccounting, "add-on ACC"},
		{permission.ResourceFactoringAgreement, platformcatalog.FeatureFactoring, "add-on ACC"},
		{permission.ResourceRateMatrix, platformcatalog.FeatureRating, "add-on RTE"},
		{permission.ResourceDetention, platformcatalog.FeatureDetention, "add-on RTE"},
		{permission.ResourceIFTAReturn, platformcatalog.FeatureFuelTax, "add-on FUL"},
		{permission.ResourceDriverSettlement, platformcatalog.FeatureSettlement, "add-on PAY"},
		{permission.ResourceDriverPortal, platformcatalog.FeatureDriverPortal, "base: the driver portal"},
		{permission.ResourceAccountingIntegration, platformcatalog.FeatureAccountingIntegration, "base: accounting integrations"},
		{permission.ResourceAccountingSync, platformcatalog.FeatureAccountingIntegration, "base: accounting integrations"},
		{permission.ResourceAccountingReconciliation, platformcatalog.FeatureAccounting, "add-on ACC"},
		{permission.ResourceWorkerPTO, platformcatalog.FeatureWorkforceTimeOff, "add-on HRS"},
		{permission.ResourceTimesheet, platformcatalog.FeatureWorkforceTimeTracking, "add-on HRS"},
		{permission.ResourceTrainingCourse, platformcatalog.FeatureWorkforceTalent, "add-on HRS"},
		{permission.ResourceBenefitPlan, platformcatalog.FeatureWorkforceBenefits, "add-on HRS"},
		{permission.ResourceWorkerPolicy, platformcatalog.FeatureWorkforceSelfService, "add-on HRS"},
		{permission.ResourceQualification, platformcatalog.FeatureWorkforceCompliance, "add-on SAF"},
		{permission.ResourceWorkerInjury, platformcatalog.FeatureWorkforceSafety, "add-on SAF"},
		{permission.ResourceTender, platformcatalog.FeatureCarrierBrokerage, "add-on BRK"},
		{permission.ResourceEDI, platformcatalog.FeatureEDIIntegration, "add-on EDI"},
		{permission.ResourceTableChangeAlert, platformcatalog.FeatureTableChangeAlerts, "add-on VIS"},
		{permission.ResourceAssistant, platformcatalog.FeatureAIDesk, "add-on AI"},
		{permission.ResourceWatchtower, platformcatalog.FeatureAIWatchtower, "add-on AI"},
		{permission.ResourceInboundMessage, platformcatalog.FeatureAITriage, "add-on AI"},
		{permission.ResourceAgentRun, platformcatalog.FeatureAgentAutomation, "add-on AI"},
		{permission.ResourceAIAuditTrail, platformcatalog.FeatureAIAuditTrail, "add-on AI"},
		{permission.ResourceIdentityProvider, platformcatalog.FeatureIdentity, "add-on SEC"},
		{permission.ResourceAuditLegalHold, platformcatalog.FeatureIdentity, "add-on SEC"},
		{permission.ResourceShipment, platformcatalog.FeatureCoreTMS, "base"},
	}

	reg := Default()
	for _, tc := range cases {
		feature, ok := reg.FeatureOf(tc.resource.String())
		assert.True(t, ok, "%s is not registered", tc.resource)
		assert.Equal(t, tc.feature, feature, "%s: %s", tc.resource, tc.why)
	}
}

func TestRegisterRefusesAFieldFeatureTheCatalogDoesNotKnow(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	err := reg.Register(&permissionspec.ResourceDefinition{
		Resource: "invoice_probe",
		Feature:  platformcatalog.FeatureBilling,
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"factoringCompany": "tms.nonexistent",
		},
	})

	require.Error(t, err)
	assert.Equal(t,
		`field factoringCompany of resource invoice_probe declares unknown feature "tms.nonexistent"`,
		err.Error())
	assert.False(t, reg.HasResource("invoice_probe"))
}

func TestRegisterRefusesAFieldFeatureThatIsTheResourcesOwn(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	err := reg.Register(&permissionspec.ResourceDefinition{
		Resource: "invoice_probe",
		Feature:  platformcatalog.FeatureBilling,
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"total": platformcatalog.FeatureBilling,
		},
	})

	require.Error(t, err)
	assert.Equal(t,
		`field total of resource invoice_probe declares "`+
			string(platformcatalog.FeatureBilling)+`", the resource's own feature`,
		err.Error())
	assert.False(t, reg.HasResource("invoice_probe"))
}

func TestFieldFeatureOfReturnsOnlyADeclaredFieldsFeature(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	require.NoError(t, reg.Register(&permissionspec.ResourceDefinition{
		Resource: "invoice_probe",
		Feature:  platformcatalog.FeatureBilling,
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"factoringCompany": platformcatalog.FeatureFactoring,
		},
	}))

	feature, ok := reg.FieldFeatureOf("invoice_probe", "factoringCompany")
	assert.True(t, ok)
	assert.Equal(t, platformcatalog.FeatureFactoring, feature)

	feature, ok = reg.FieldFeatureOf("invoice_probe", "total")
	assert.False(t, ok, "a field without an entry is the resource's own")
	assert.Empty(t, feature)

	feature, ok = reg.FieldFeatureOf("not_a_resource", "factoringCompany")
	assert.False(t, ok)
	assert.Empty(t, feature)
}
