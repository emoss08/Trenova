package agentextension

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type registered struct {
	fx.In

	Connectors []agentextensionports.ExtensionConnector `group:"agent_extension_connectors"`
}

func TestEveryExtensionHasAConnectorForItsCapability(t *testing.T) {
	t.Parallel()

	var got registered
	app := fxtest.New(t, Module, fx.Populate(&got))
	app.RequireStart()
	app.RequireStop()

	byType := make(map[agentextension.Type]agentextensionports.ExtensionConnector, len(got.Connectors))
	for _, connector := range got.Connectors {
		_, duplicate := byType[connector.Type()]
		assert.False(t, duplicate, "%s is registered twice", connector.Type())
		byType[connector.Type()] = connector
	}

	for _, typ := range agentextension.AllTypes() {
		connector, ok := byType[typ]
		require.True(t, ok, "%s has no connector", typ)

		spec, _ := agentextension.SpecFor(typ)
		switch spec.Capability {
		case agentextension.CapabilityWebResearch:
			_, searches := connector.(agentextensionports.WebSearchProvider)
			assert.True(t, searches, "%s provides web research but cannot search", typ)
		case agentextension.CapabilityPortalAutomation:
			_, browses := connector.(agentextensionports.BrowserProvider)
			assert.True(t, browses, "%s provides portal automation but cannot browse", typ)
		case agentextension.CapabilityDataAnalysis:
			_, runs := connector.(agentextensionports.SandboxProvider)
			assert.True(t, runs, "%s provides data analysis but cannot run code", typ)
		default:
			t.Errorf("%s provides %s, which this test does not know how to check", typ,
				spec.Capability)
		}
	}
}
