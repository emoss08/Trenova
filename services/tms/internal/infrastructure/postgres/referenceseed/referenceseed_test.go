package referenceseed

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemAgentDefinitionsAreDisabledAndShadowed(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	definitions := SystemAgentDefinitions(orgID, buID)
	require.Len(t, definitions, 2)

	keys := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		assert.Equal(t, orgID, definition.OrganizationID)
		assert.Equal(t, buID, definition.BusinessUnitID)
		assert.False(t, definition.Enabled)
		assert.True(t, definition.ShadowMode)
		keys = append(keys, definition.SystemKey)
	}
	assert.ElementsMatch(
		t,
		[]string{SystemAgentKeyBillingException, SystemAgentKeyDispatchAssignment},
		keys,
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
