package resolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type subsetProposalTool struct{}

func (subsetProposalTool) Name() string        { return "transfer_to_billing" }
func (subsetProposalTool) Description() string { return "Transfer shipments." }
func (subsetProposalTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentIds": toolschema.RecordSubset(permission.ResourceShipment.String(),
				map[string]any{"type": "array", "items": map[string]any{"type": "string"}}),
			"billType": map[string]any{"type": "string"},
		},
		"required": []string{"shipmentIds"},
	}
}
func (subsetProposalTool) Policy() services.ToolPolicy { return services.ToolPolicy{} }
func (subsetProposalTool) Execute(context.Context, services.ToolExecuteParams) error {
	return nil
}

type subsetToolRegistry struct{}

func (subsetToolRegistry) Get(string) (services.AgentTool, bool) {
	return subsetProposalTool{}, true
}
func (subsetToolRegistry) All() []services.AgentTool                   { return nil }
func (subsetToolRegistry) Descriptors() []services.AgentToolDescriptor { return nil }

// A client builds the approval form from parameterFields: a record-subset
// parameter comes back as RecordSubset naming the resource its ids belong to,
// and every other field names none.
func TestAgentProposalParameterFields_ExposeARecordSubset(t *testing.T) {
	t.Parallel()

	r := &Resolver{agentTools: subsetToolRegistry{}}
	fields, err := (&agentProposalResolver{r}).ParameterFields(t.Context(), &agent.AgentProposal{
		Status:     agent.ProposalStatusPending,
		ToolName:   "transfer_to_billing",
		ToolParams: map[string]any{"shipmentIds": []any{"shp_a", "shp_b"}},
	})
	require.NoError(t, err)
	require.Len(t, fields, 2)

	byName := map[string]*toolschema.Field{}
	for _, field := range fields {
		byName[field.Name] = field
	}

	subset := byName["shipmentIds"]
	assert.Equal(t, toolschema.KindRecordSubset, subset.Kind)
	resource, err := (&agentProposalFieldResolver{r}).Resource(t.Context(), subset)
	require.NoError(t, err)
	require.NotNil(t, resource)
	assert.Equal(t, "shipment", *resource)

	none, err := (&agentProposalFieldResolver{r}).Resource(t.Context(), byName["billType"])
	require.NoError(t, err)
	assert.Nil(t, none)
}
