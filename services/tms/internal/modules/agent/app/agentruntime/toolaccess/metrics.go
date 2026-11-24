package toolaccess

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/metricfind"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func (s *Access) findMetrics(
	ctx context.Context,
	set *ToolSet,
	actor *permissionports.RequestActor,
	need string,
) metricfind.Found {
	reads := slices.Contains(set.Allowed, agenttoolports.MetricReadTool)
	found := metricfind.Found{Section: metricfind.Find(ctx, &metricfind.Request{
		Tools:  s.queryTools,
		Actor:  actor,
		Access: set.DataAccess,
		Need:   need,
		Reads:  reads,
		Draws:  set.Offers(runtimetools.ShowAnswerTool),
	})}
	if found.Text != "" && reads {
		found.Loaded = set.Offers(agenttoolports.MetricReadTool) ||
			s.Load(set, agenttoolports.MetricReadTool)
	}

	return found
}

func (s *Access) openMetrics(set *ToolSet, selected []string, text string) {
	if slices.Contains(selected, agenttoolports.MetricReadTool) && metricfind.Mentions(text) {
		s.Load(set, agenttoolports.MetricReadTool)
	}
}
