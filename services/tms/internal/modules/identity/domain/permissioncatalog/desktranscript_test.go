package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/agentceiling"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeskTranscript_IsAConfidentialAIResourceToViewAndExport(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	def, ok := reg.Get(permission.ResourceDeskTranscript.String())
	require.True(t, ok, "desk_transcript must be registered")

	assert.Equal(t, "AI & agents", def.Category)
	assert.Equal(t, permission.SensitivityConfidential, def.DefaultSensitivity)
	assert.Equal(t, platformcatalog.FeatureAIDesk, def.Feature)
	assert.Empty(t, def.ParentResource, "no other grant may reach other people's conversations")

	operations := make([]permission.Operation, 0, len(def.Operations))
	for _, op := range def.Operations {
		operations = append(operations, op.Operation)
	}
	assert.Equal(t, []permission.Operation{permission.OpRead, permission.OpExport}, operations)
}

func TestDeskTranscript_IsNoTemplatesAndOneBundleGrantsIt(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	for _, template := range JobTemplates() {
		assert.Empty(t,
			effectiveGrant(reg, reg.TemplateGrants(&template), permission.ResourceDeskTranscript),
			"template %q must not read other people's conversations", template.ID)
	}

	var granting []string
	for _, bundle := range Bundles() {
		for _, grant := range bundle.Grants {
			if grant.Resource == permission.ResourceDeskTranscript {
				granting = append(granting, bundle.ID)
				assert.ElementsMatch(t,
					[]permission.Operation{permission.OpRead, permission.OpExport},
					grant.Operations)
			}
		}
	}
	assert.Equal(t, []string{"desk.transcript-oversight"}, granting)
}

func TestDeskTranscript_NoAgentOrSupportSessionReachesIt(t *testing.T) {
	t.Parallel()

	for _, op := range []permission.Operation{permission.OpRead, permission.OpExport} {
		assert.False(t, agentceiling.Allows(permission.ResourceDeskTranscript, op),
			"an agent must never read other people's conversations")
		assert.NotZero(t, supportaccess.DeniedLimit(permission.ResourceDeskTranscript.String(), op),
			"Trenova support must never read other people's conversations")
		assert.False(t, supportaccess.Allows(
			supportaccess.ScopeFull,
			permission.ResourceDeskTranscript.String(),
			op,
		))
	}
}

func TestDeskTranscript_ExportImpliesView(t *testing.T) {
	t.Parallel()

	assert.Contains(t, permission.Implied([]permission.Operation{permission.OpExport}),
		permission.OpRead,
		"Export already means View everywhere; the page and the server follow that")
}
