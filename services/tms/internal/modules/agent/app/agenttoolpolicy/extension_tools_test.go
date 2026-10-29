package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/stretchr/testify/assert"
)

// Every tool an extension's capability names is registered, as a read or as
// a write, and only the reads that return outside content hold later writes.
func TestEveryExtensionToolIsRegistered(t *testing.T) {
	t.Parallel()

	tools := buildRegistered(t)
	reads := make(map[string]bool, len(tools.Queries))
	for _, tool := range tools.Queries {
		reads[tool.Name()] = tool.Policy().CapsLaterWrites
	}
	writes := make(map[string]struct{}, len(tools.Actions))
	for _, tool := range tools.Actions {
		writes[tool.Name()] = struct{}{}
	}

	for _, capability := range agentextension.AllCapabilities() {
		spec, _ := agentextension.CapabilitySpecFor(capability)
		for _, name := range spec.Tools {
			capsLater, isRead := reads[name]
			_, isWrite := writes[name]
			if !isRead && !isWrite {
				t.Errorf("%s declares %s but no tool is registered", capability, name)
				continue
			}
			assert.Equal(t, spec.ReturnsOutsideContent(name), isRead && capsLater,
				"%s: whether it holds later writes must match its capability", name)
			assert.Equal(t, spec.ReturnsOutsideContent(name), toolaccess.ReadsExternalContent(name),
				"%s: the runtime must hold later writes exactly when the capability says", name)
		}
	}
}
