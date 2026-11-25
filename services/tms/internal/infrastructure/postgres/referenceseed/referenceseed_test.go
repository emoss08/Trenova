package referenceseed

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemAgentDefinitionsSeedTheDesksOnAndTheEventAgentsOffInShadow(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	definitions := SystemAgentDefinitions(orgID, buID)
	desks := agentpreset.DefaultDesks()
	require.Len(t, definitions, len(desks)+2)

	deskKeys := make([]string, 0, len(desks))
	for _, template := range desks {
		deskKeys = append(deskKeys, agentpreset.DeskKey(template))
	}
	var seededDesks, eventKeys []string
	for _, definition := range definitions {
		assert.Equal(t, orgID, definition.OrganizationID)
		assert.Equal(t, buID, definition.BusinessUnitID)
		if slices.Contains(deskKeys, definition.SystemKey) {
			assert.True(t, definition.Enabled, definition.SystemKey)
			assert.False(t, definition.ShadowMode, definition.SystemKey)
			seededDesks = append(seededDesks, definition.SystemKey)
			continue
		}
		assert.False(t, definition.Enabled, definition.SystemKey)
		assert.True(t, definition.ShadowMode, definition.SystemKey)
		eventKeys = append(eventKeys, definition.SystemKey)
	}
	assert.ElementsMatch(t, deskKeys, seededDesks)
	assert.ElementsMatch(
		t,
		[]string{SystemAgentKeyBillingException, SystemAgentKeyDispatchAssignment},
		eventKeys,
	)
}

func TestTemplateCodeFor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "INVOICE_PDF", templateCodeFor(documenttemplate.KindInvoicePDF))
	for _, kind := range documenttemplate.AllKinds() {
		assert.LessOrEqual(t, len(templateCodeFor(kind)), documenttemplate.MaxCodeLength)
	}
}

func TestDefaultServiceFailureReasonCodesAreUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, len(defaultServiceFailureReasonCodes))
	for _, def := range defaultServiceFailureReasonCodes {
		_, dup := seen[def.code]
		assert.False(t, dup, def.code)
		seen[def.code] = struct{}{}
	}
}
