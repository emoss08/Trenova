package resolver

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

type choiceLabeler struct {
	mu     sync.Mutex
	labels services.RecordLabels
	reads  int
}

func (l *choiceLabeler) Labels(
	_ context.Context,
	_ pagination.TenantInfo,
	refs map[permission.Resource][]pulid.ID,
) (services.RecordLabels, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reads++

	out := make(services.RecordLabels, len(refs))
	for resource, ids := range refs {
		out[resource] = make(map[pulid.ID]string, len(ids))
		for _, id := range ids {
			if label := l.labels.Label(resource, id); label != "" {
				out[resource][id] = label
			}
		}
	}

	return out, nil
}

func (l *choiceLabeler) readCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.reads
}

type choicesHarness struct {
	resolver *agentProposalResolver
	fields   *agentProposalFieldResolver
	labeler  *choiceLabeler
	ctx      context.Context
	known    pulid.ID
	gone     pulid.ID
	other    pulid.ID
}

func newChoicesHarness(t *testing.T, granted bool) *choicesHarness {
	t.Helper()

	h := &choicesHarness{
		known: pulid.MustNew("shp_"),
		gone:  pulid.MustNew("shp_"),
		other: pulid.MustNew("shp_"),
	}
	h.labeler = &choiceLabeler{labels: services.RecordLabels{
		permission.ResourceShipment: {h.known: "PRO-1001", h.other: "PRO-2002"},
	}}
	auth := &authctx.AuthContext{
		PrincipalType:  string(services.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	grants := map[string]bool{}
	if granted {
		grants[permission.ResourceShipment.String()+"|"+string(permission.OpRead)] = true
	}
	r := &Resolver{
		l:                zap.NewNop(),
		agentTools:       subsetToolRegistry{},
		permissionEngine: &previewPermissions{granted: grants},
	}
	h.resolver = &agentProposalResolver{r}
	h.fields = &agentProposalFieldResolver{r}
	factory := loaders.NewSubsetLabelsLoaderFactory(loaders.SubsetLabelsLoaderFactoryParams{
		Labeler: h.labeler,
	})
	h.ctx = loaders.WithLoaders(gqlctx.WithAuthContext(t.Context(), auth), &loaders.Loaders{
		SubsetLabels: factory.NewForTenant(pagination.TenantInfo{
			OrgID: auth.OrganizationID,
			BuID:  auth.BusinessUnitID,
		}),
	})

	return h
}

func (h *choicesHarness) fieldsOf(
	t *testing.T,
	params map[string]any,
) map[string]*toolschema.Field {
	t.Helper()

	fields, err := h.resolver.ParameterFields(h.ctx, &agent.AgentProposal{
		Status:     agent.ProposalStatusPending,
		ToolName:   "transfer_to_billing",
		ToolParams: params,
	})
	require.NoError(t, err)

	byName := make(map[string]*toolschema.Field, len(fields))
	for _, field := range fields {
		byName[field.Name] = field
	}

	return byName
}

func choiceValues(choices []*toolschema.Choice) []toolschema.Choice {
	out := make([]toolschema.Choice, 0, len(choices))
	for _, choice := range choices {
		out = append(out, *choice)
	}

	return out
}

// A person unticks records from every one the agent proposed, named by its
// PRO number, in the order proposed. A record that is gone keeps its id, and
// a field that is not a subset lists nothing.
func TestAgentProposalFieldChoices_ListEveryProposedRecordByItsLabel(t *testing.T) {
	t.Parallel()

	h := newChoicesHarness(t, true)
	fields := h.fieldsOf(t, map[string]any{
		"shipmentIds": []any{h.gone.String(), h.known.String()},
		"billType":    "Invoice",
	})

	choices, err := h.fields.Choices(h.ctx, fields["shipmentIds"])
	require.NoError(t, err)
	assert.Equal(t, []toolschema.Choice{
		{ID: h.gone.String(), Label: h.gone.String()},
		{ID: h.known.String(), Label: "PRO-1001"},
	}, choiceValues(choices))

	none, err := h.fields.Choices(h.ctx, fields["billType"])
	require.NoError(t, err)
	assert.Nil(t, none, "only a subset field lists records")
}

// A page of proposals labels all of their records in one read: the loader
// batches every subset field asked for together.
func TestAgentProposalFieldChoices_LabelAPageOfProposalsInOneRead(t *testing.T) {
	t.Parallel()

	h := newChoicesHarness(t, true)
	first := h.fieldsOf(t, map[string]any{"shipmentIds": []any{h.known.String()}})
	second := h.fieldsOf(t, map[string]any{"shipmentIds": []any{h.other.String()}})

	var wg sync.WaitGroup
	results := make([][]*toolschema.Choice, 2)
	for idx, field := range []*toolschema.Field{first["shipmentIds"], second["shipmentIds"]} {
		wg.Go(func() {
			choices, err := h.fields.Choices(h.ctx, field)
			assert.NoError(t, err)
			results[idx] = choices
		})
	}
	wg.Wait()

	assert.Equal(t, []toolschema.Choice{{ID: h.known.String(), Label: "PRO-1001"}},
		choiceValues(results[0]))
	assert.Equal(t, []toolschema.Choice{{ID: h.other.String(), Label: "PRO-2002"}},
		choiceValues(results[1]))
	assert.Equal(t, 1, h.labeler.readCount())
}

// A reader who may not read shipments is not told their PRO numbers through a
// proposal: the records are listed by the ids the proposal already carries,
// and no label is read.
func TestAgentProposalFieldChoices_NameUnreadableRecordsByTheirIDs(t *testing.T) {
	t.Parallel()

	h := newChoicesHarness(t, false)
	fields := h.fieldsOf(t, map[string]any{"shipmentIds": []any{h.known.String()}})

	choices, err := h.fields.Choices(h.ctx, fields["shipmentIds"])
	require.NoError(t, err)
	assert.Equal(t, []toolschema.Choice{{ID: h.known.String(), Label: h.known.String()}},
		choiceValues(choices))
	assert.Zero(t, h.labeler.readCount())
}
