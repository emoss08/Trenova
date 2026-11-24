package agentevalgate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/stretchr/testify/require"
)

func TestCapabilitySearchOnTheSelectionSuites(t *testing.T) {
	capabilityHarnessEnabled(t)

	kit := newKit(t)
	index, err := agenttoolcatalog.BuildCapabilityIndex(kit.Queries, kit.Actions, kit.Runtime)
	require.NoError(t, err)
	for _, path := range []string{"evals/toolselection.yaml", "evals/toolselection.paraphrase.yaml"} {
		suite, err := agentevalgate.LoadSelectionSuite(path)
		require.NoError(t, err)
		top1, top3 := 0, 0
		for _, c := range suite.Cases {
			want := make(map[string]bool, len(c.Expect))
			for _, name := range c.Expect {
				want[agentcapability.CapabilityIDForTool(name)] = true
			}
			for rank, hit := range index.Search(c.Request, nil).Hits {
				if !want[hit.Capability.ID] {
					continue
				}
				if rank == 0 {
					top1++
				}
				if rank < agentevalgate.CapabilityTop {
					top3++
				}
				break
			}
		}
		total := float64(len(suite.Cases))
		t.Logf("%s: %d cases, top 1 %.1f%%, top 3 %.1f%%", path, len(suite.Cases),
			100*float64(top1)/total, 100*float64(top3)/total)
	}
}
