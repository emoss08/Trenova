package agentaccessservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/assert"
)

func TestSensitiveTools_NamesToolsOnRestrictedResources(t *testing.T) {
	t.Parallel()

	registry := permission.NewEmptyRegistry()
	for resource, sensitivity := range map[permission.Resource]permission.FieldSensitivity{
		"shipment":   permission.SensitivityInternal,
		"worker_pay": permission.SensitivityConfidential,
		"api_key":    permission.SensitivityRestricted,
	} {
		assert.NoError(t, registry.Register(&permission.ResourceDefinition{
			Resource:           resource.String(),
			DisplayName:        resource.String(),
			DefaultSensitivity: sensitivity,
		}))
	}

	held := []HeldTool{
		{Name: "get_shipment", Resource: "shipment", Operation: permission.OpRead},
		{Name: "get_worker_pay", Resource: "worker_pay", Operation: permission.OpRead},
		{Name: "rotate_api_key", Resource: "api_key", Operation: permission.OpUpdate},
		{Name: "unknown", Resource: "nothing", Operation: permission.OpRead},
	}

	assert.Equal(t,
		[]string{"get_worker_pay", "rotate_api_key"},
		SensitiveTools(held, DefaultSensitiveRule(registry)),
	)
}

// Another reason a tool is sensitive is ORed in, never replacing the first.
func TestAnySensitive_OrsItsRules(t *testing.T) {
	t.Parallel()

	registry := permission.NewEmptyRegistry()
	assert.NoError(t, registry.Register(&permission.ResourceDefinition{
		Resource:           "worker_pay",
		DisplayName:        "worker_pay",
		DefaultSensitivity: permission.SensitivityConfidential,
	}))
	sendsOut := func(tool HeldTool) bool { return tool.Name == "send_email" }

	rule := AnySensitive(RestrictedResourceRule(registry), sendsOut, nil)
	held := []HeldTool{
		{Name: "send_email", Resource: "email", Operation: permission.OpCreate},
		{Name: "get_worker_pay", Resource: "worker_pay", Operation: permission.OpRead},
		{Name: "list_reports", Resource: "report", Operation: permission.OpRead},
	}

	assert.Equal(t, []string{"send_email", "get_worker_pay"}, SensitiveTools(held, rule))
	assert.Empty(t, SensitiveTools(held, nil))
}
