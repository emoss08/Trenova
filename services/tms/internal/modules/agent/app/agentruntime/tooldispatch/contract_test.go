package tooldispatch

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/recordnumbers"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordnumberports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var proNumbered = recordkind.Kind(permission.ResourceShipment)

type proNumbers struct {
	records map[string]pulid.ID
	reads   int
}

func (p *proNumbers) Numbering(kind recordkind.Kind) (recordnumberports.Numbering, bool) {
	if kind != proNumbered {
		return recordnumberports.Numbering{}, false
	}

	return recordnumberports.Numbering{Field: "proNumber", Name: "PRO number"}, true
}

func (p *proNumbers) FindNumbers(
	_ context.Context,
	_ pagination.TenantInfo,
	numbers map[recordkind.Kind][]string,
) (recordnumberports.NumberedRecords, error) {
	p.reads++
	found := make(map[string][]recordnumberports.NumberedRecord)
	for _, number := range numbers[proNumbered] {
		if id, ok := p.records[number]; ok {
			found[number] = []recordnumberports.NumberedRecord{{ID: id, Number: number}}
		}
	}

	return recordnumberports.NumberedRecords{proNumbered: found}, nil
}

type internalReader struct {
	*agenttest.StubPermissions
}

func (internalReader) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{
		Resource:       resource,
		MaxSensitivity: permission.SensitivityInternal,
	}, nil
}

type namedRecords struct{}

func (namedRecords) Labels(
	context.Context,
	pagination.TenantInfo,
	map[permission.Resource][]pulid.ID,
) (agentruntimeports.RecordLabels, error) {
	return agentruntimeports.RecordLabels{}, nil
}

func numberedDispatcher(
	p DispatcherParams,
	records *proNumbers,
) *Dispatcher {
	permissions := internalReader{StubPermissions: &agenttest.StubPermissions{}}
	p.Permissions = permissions
	p.Labels = namedRecords{}
	p.Numbers = recordnumbers.New(recordnumbers.Params{
		Records:     records,
		Permissions: permissions,
	})

	return testDispatcher(p)
}

// "Put S26110493143683 and S26110579415132 side by side": the numbers sent for
// the ids are read as the loads, the tool reads the ids, a table pages on the
// ids, and the model is told what was read.
func TestDispatch_AReadGivenNumbersRunsWithTheirRecords(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	tool := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Result:   map[string]any{"count": 2},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"shipmentIds": agenttoolschema.RecordIDs(permission.ResourceShipment,
					"The shipments' ids, from shipment_list.", 25),
			},
		},
	}
	records := &proNumbers{records: map[string]pulid.ID{
		"S26110493143683": first, "S26110579415132": second,
	}}
	d := numberedDispatcher(DispatcherParams{
		QueryTools: &agenttest.StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{tool}},
	}, records)

	sent := map[string]any{"shipmentIds": []any{"S26110493143683", "S26110579415132"}}
	outcome := d.dispatch(t.Context(), &DispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: runtimetest.Definition(tool.ToolName),
			Actor:      runtimetest.Actor(),
		},
		call:  llmports.ToolCall{ID: "call-1", Name: tool.ToolName, Arguments: sent},
		taint: &agent.RunTaint{},
	})

	require.False(t, outcome.Failed, outcome.Content)
	assert.Equal(t, 1, records.reads, "both numbers in one read")
	want := map[string]any{"shipmentIds": []any{first.String(), second.String()}}
	assert.Equal(t, want, tool.LastParams.Params)
	assert.Equal(t, want, outcome.RanWith, "what a table pages on is the ids")
	assert.Equal(t, []any{"S26110493143683", "S26110579415132"}, sent["shipmentIds"])
	assert.Contains(t, outcome.Content, `[Record numbers were read as the records they name: `+
		`shipmentIds[0] "S26110493143683" is the shipment with that PRO number, id `+
		first.String())
}

// A write given a number runs, and is recorded, with the record's id.
func TestDispatch_AWriteGivenANumberRunsAndIsRecordedWithTheID(t *testing.T) {
	t.Parallel()

	load := pulid.MustNew("shp_")
	tool := &agenttest.StubActionTool{
		ToolName: "shipment_add_comment",
		Tier:     agent.TierAutoExecute,
		Resource: permission.ResourceShipmentComment,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"shipmentId": agenttoolschema.RecordID(permission.ResourceShipment, "The shipment",
					"shipment_list"),
				"comment": map[string]any{"type": "string"},
			},
		},
	}
	d := numberedDispatcher(DispatcherParams{
		ActionTools: &agenttest.StubActionRegistry{Tools: []agenttoolports.AgentTool{tool}},
	}, &proNumbers{records: map[string]pulid.ID{"SEED-SHP-008": load}})

	outcome := d.dispatch(t.Context(), &DispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: runtimetest.AutoDefinition(tool.ToolName),
			Actor:      runtimetest.Actor(),
		},
		call: llmports.ToolCall{ID: "call-1", Name: tool.ToolName, Arguments: map[string]any{
			"shipmentId": "SEED-SHP-008", "comment": "Receiver pushed to tomorrow.",
		}},
		taint: &agent.RunTaint{},
	})

	require.False(t, outcome.Failed, outcome.Content)
	require.Equal(t, 1, tool.Calls)
	assert.Equal(t, load.String(), tool.LastParams.Params["shipmentId"])
	require.NotNil(t, outcome.Action)
	assert.Equal(t, load.String(), outcome.Action.Arguments["shipmentId"])
	assert.Contains(t, outcome.Content, `shipmentId "SEED-SHP-008" is the shipment with that `+
		`PRO number, id `+load.String()+`. A number is not a record id: send the id from now on.`)
}

func TestDispatch_ANumberNoRecordCarriesIsRefusedAsBefore(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"shipmentIds": agenttoolschema.RecordIDs(permission.ResourceShipment,
					"The shipments' ids, from shipment_list.", 25),
			},
		},
	}
	d := numberedDispatcher(DispatcherParams{
		QueryTools: &agenttest.StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{tool}},
	}, &proNumbers{})

	outcome := d.dispatch(t.Context(), &DispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: runtimetest.Definition(tool.ToolName),
			Actor:      runtimetest.Actor(),
		},
		call: llmports.ToolCall{ID: "call-1", Name: tool.ToolName, Arguments: map[string]any{
			"shipmentIds": []any{"S404"},
		}},
		taint: &agent.RunTaint{},
	})

	assert.True(t, outcome.Failed)
	assert.Zero(t, tool.Calls)
	assert.Contains(t, outcome.Content, `shipmentIds[0]: "S404" is not a record id.`)
}
