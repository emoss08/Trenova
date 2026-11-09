package deskattentionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func due(at int64) *int64 { return &at }

func item(id string, about pulid.ID, severity watchtower.Severity, dueAt *int64) *watchtower.Item {
	entry := &watchtower.Item{ID: pulid.ID(id), Severity: severity, DueAt: dueAt, Seen: true}
	entry.SetAbout(about)
	return entry
}

const (
	shipmentA = pulid.ID("shp_01JABCDEFGHJKMNPQRSTVWXYZ0")
	shipmentB = pulid.ID("shp_01JABCDEFGHJKMNPQRSTVWXYZ1")
)

func TestGroup_OneRowPerRecordWithItsWorstSeverityAndSoonestDeadline(t *testing.T) {
	late := item("wt_1", shipmentA, watchtower.SeverityWarning, due(300))
	email := item("wt_2", shipmentA, watchtower.SeverityInfo, nil)
	plan := item("wt_3", shipmentA, watchtower.SeverityCritical, due(200))
	plan.Seen = false
	other := item("wt_4", shipmentB, watchtower.SeverityInfo, due(100))

	groups := group([]*watchtower.Item{late, email, plan, other})

	require.Len(t, groups, 2)
	assert.Equal(t, shipmentB, groups[0].Key, "the soonest deadline leads")
	row := groups[1]
	assert.Equal(t, shipmentA, row.Key)
	assert.Equal(t, watchtower.SeverityCritical, row.Severity)
	require.NotNil(t, row.DueAt)
	assert.Equal(t, int64(200), *row.DueAt)
	assert.True(t, row.Unseen)
	assert.Equal(t, []pulid.ID{"wt_3", "wt_1", "wt_2"}, idsOf(row.Items))
	require.NotNil(t, row.Record)
	assert.Equal(t, "shipment", string(row.Record.Kind))
}

func TestGroup_AnItemAboutNothingIsItsOwnRow(t *testing.T) {
	first := item("wt_1", pulid.Nil, watchtower.SeverityInfo, nil)
	second := item("wt_2", pulid.Nil, watchtower.SeverityInfo, nil)

	groups := group([]*watchtower.Item{first, second})

	require.Len(t, groups, 2)
	assert.Nil(t, groups[0].Record)
}

func TestFirstGroups_StopsAtTheFirstPageAndSaysMoreWait(t *testing.T) {
	items := []*watchtower.Item{
		item("wt_1", shipmentA, watchtower.SeverityInfo, due(1)),
		item("wt_2", shipmentA, watchtower.SeverityInfo, due(2)),
		item("wt_3", shipmentB, watchtower.SeverityInfo, due(3)),
	}

	keys, more := firstGroups(items, 1)

	assert.Equal(t, []pulid.ID{shipmentA}, keys)
	assert.True(t, more)
	assert.Equal(t, []pulid.ID{"wt_1", "wt_2"}, idsOf(keep(items, keys)))
}

func TestLensCounts_SumsEachKindIntoItsLens(t *testing.T) {
	counts := lensCounts(&watchtowerports.WatchtowerCounts{ByKind: map[watchtower.SourceKind]int{
		watchtower.SourceEtaAtRisk:      4,
		watchtower.SourceInboundMessage: 2,
		watchtower.SourceAgentProposal:  1,
		watchtower.SourceAgentPlan:      1,
	}})

	assert.Equal(t, 8, counts.Everything)
	assert.Equal(t, 4, counts.Watchtower)
	assert.Equal(t, 2, counts.Triage)
	assert.Equal(t, 2, counts.Decisions)
}

func TestMerge_AddsWhatTheSecondReadFoundOnce(t *testing.T) {
	first := []*watchtower.Item{item("wt_1", shipmentA, watchtower.SeverityInfo, nil)}
	more := []*watchtower.Item{
		item("wt_1", shipmentA, watchtower.SeverityInfo, nil),
		item("wt_9", shipmentA, watchtower.SeverityInfo, nil),
	}

	assert.Equal(t, []pulid.ID{"wt_1", "wt_9"}, idsOf(merge(first, more)))
}

func idsOf(items []*watchtower.Item) []pulid.ID {
	out := make([]pulid.ID, 0, len(items))
	for _, entry := range items {
		out = append(out, entry.ID)
	}
	return out
}
