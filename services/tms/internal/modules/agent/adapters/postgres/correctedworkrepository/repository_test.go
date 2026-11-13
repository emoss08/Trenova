package correctedworkrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(24 * 60 * 60)

func updated(from, to any) map[string]any {
	return map[string]any{"from": from, "to": to, "type": "updated"}
}

func agentWrite(
	agentID pulid.ID,
	record string,
	at int64,
	op permission.Operation,
	changes map[string]any,
) *audit.Entry {
	return &audit.Entry{
		Resource:   permission.ResourceShipment,
		ResourceID: record,
		Timestamp:  at,
		Operation:  op,
		Changes:    changes,
		Metadata:   map[string]any{audit.MetadataAgentID: agentID.String()},
	}
}

func handEdit(record string, at int64, changes map[string]any) *audit.Entry {
	return &audit.Entry{
		Resource:   permission.ResourceShipment,
		ResourceID: record,
		Timestamp:  at,
		Operation:  permission.OpUpdate,
		EntityRef:  "S-1042",
		Changes:    changes,
	}
}

func TestCorrected_KeepsTheFieldsOfTheAgentsThatAPersonChanged(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	writes := []*audit.Entry{
		agentWrite(agentID, "shp_1", 100, permission.OpUpdate, map[string]any{
			"deliveryWindowStart": updated("09:00", "14:00"),
		}),
	}
	edits := []*audit.Entry{
		handEdit("shp_1", 200, map[string]any{
			"deliveryWindowStart": updated("14:00", "08:00"),
			"notes":               updated("", "call ahead"),
			"updatedAt":           updated(100, 200),
		}),
	}

	got := Corrected(writes, edits)

	require.Len(t, got, 1)
	assert.Equal(t, agentID, got[0].AgentDefinitionID)
	assert.Equal(t, "S-1042", got[0].Label)
	assert.Equal(t, []agentreflectionports.CorrectedField{
		{Field: "deliveryWindowStart", Was: "14:00", Now: "08:00"},
	}, got[0].Fields, "a field the agent never set and bookkeeping fields are not corrections")
}

func TestCorrected_AnAgentThatCreatedTheRecordSetAllOfIt(t *testing.T) {
	t.Parallel()

	writes := []*audit.Entry{
		agentWrite(pulid.MustNew("agdef_"), "shp_1", 100, permission.OpCreate, map[string]any{}),
	}
	edits := []*audit.Entry{
		handEdit("shp_1", 200, map[string]any{"serviceType": updated("Standard", "Expedited")}),
	}

	got := Corrected(writes, edits)

	require.Len(t, got, 1)
	assert.Equal(t, "serviceType", got[0].Fields[0].Field)
}

func TestCorrected_LeavesOutWhatIsNotACorrectionOfAnAgent(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	window := map[string]any{"deliveryWindowStart": updated("09:00", "14:00")}
	changed := map[string]any{"deliveryWindowStart": updated("14:00", "08:00")}

	for name, tc := range map[string]struct {
		writes []*audit.Entry
		edits  []*audit.Entry
	}{
		"an edit made before the agent wrote": {
			writes: []*audit.Entry{agentWrite(agentID, "shp_1", 300, permission.OpUpdate, window)},
			edits:  []*audit.Entry{handEdit("shp_1", 200, changed)},
		},
		"an edit more than a week after the write, when the record has moved on": {
			writes: []*audit.Entry{agentWrite(agentID, "shp_1", 100, permission.OpUpdate, window)},
			edits:  []*audit.Entry{handEdit("shp_1", 100+8*day, changed)},
		},
		"an edit to another record": {
			writes: []*audit.Entry{agentWrite(agentID, "shp_1", 100, permission.OpUpdate, window)},
			edits:  []*audit.Entry{handEdit("shp_2", 200, changed)},
		},
		"an edit to a field the agent did not set": {
			writes: []*audit.Entry{agentWrite(agentID, "shp_1", 100, permission.OpUpdate, window)},
			edits: []*audit.Entry{
				handEdit("shp_1", 200, map[string]any{"notes": updated("", "call ahead")}),
			},
		},
		"a write that names no agent": {
			writes: []*audit.Entry{{
				Resource: permission.ResourceShipment, ResourceID: "shp_1", Timestamp: 100,
				Operation: permission.OpUpdate, Changes: window,
			}},
			edits: []*audit.Entry{handEdit("shp_1", 200, changed)},
		},
	} {
		assert.Empty(t, Corrected(tc.writes, tc.edits), name)
	}
}

func TestCorrected_PairsAnEditWithTheAgentsLatestWriteBeforeIt(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("agdef_"), pulid.MustNew("agdef_")
	window := map[string]any{"deliveryWindowStart": updated("09:00", "14:00")}
	writes := []*audit.Entry{
		agentWrite(first, "shp_1", 100, permission.OpUpdate, window),
		agentWrite(second, "shp_1", 150, permission.OpUpdate, window),
	}
	edits := []*audit.Entry{
		handEdit("shp_1", 200, map[string]any{"deliveryWindowStart": updated("14:00", "08:00")}),
	}

	got := Corrected(writes, edits)

	require.Len(t, got, 1)
	assert.Equal(t, second, got[0].AgentDefinitionID)
	assert.Equal(t, int64(150), got[0].WrittenAt)
}
