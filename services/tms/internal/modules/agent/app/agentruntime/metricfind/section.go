package metricfind

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	offeredLimit = 3
	drawnGroups  = 2
)

type Request struct {
	Tools  agenttoolports.AgentQueryToolRegistry
	Actor  *permissionports.RequestActor
	Access permission.FieldSensitivity
	Need   string
	Reads  bool
	Draws  bool
}

type Section struct {
	Text    string
	Metrics []string
}

type Found struct {
	Section
	Loaded bool
}

func (f *Found) WithTools(content string) string {
	if f.Text == "" {
		return content
	}

	return f.Text + "\n\n" + content
}

func (f *Found) Names(names []string) []string {
	if !f.Loaded || slices.Contains(names, agenttoolports.MetricReadTool) {
		return names
	}

	return append(names, agenttoolports.MetricReadTool)
}

func Find(ctx context.Context, req *Request) Section {
	if (!req.Reads && !req.Draws) || req.Actor == nil || req.Tools == nil {
		return Section{}
	}
	tool, registered := req.Tools.Get(agenttoolports.MetricReadTool)
	gate, gated := tool.(agenttoolports.MetricGate)
	if !registered || !gated {
		return Section{}
	}
	offered := make([]boardcatalog.Metric, 0, offeredLimit)
	matched := Matching(req.Need, offeredLimit)
	for idx := range matched {
		if gate.Readable(ctx, req.Actor, req.Access, matched[idx].Name) {
			offered = append(offered, matched[idx])
		}
	}
	if len(offered) == 0 {
		return Section{}
	}

	return Section{Text: write(offered, req.Reads, req.Draws), Metrics: names(offered)}
}

func names(metrics []boardcatalog.Metric) []string {
	out := make([]string, 0, len(metrics))
	for idx := range metrics {
		out = append(out, metrics[idx].Name)
	}

	return out
}

func write(metrics []boardcatalog.Metric, reads, draws bool) string {
	var b strings.Builder
	b.WriteString("Trenova keeps these figures over time as metrics, read live. Use them " +
		"rather than rebuilding them from records, reports or scripts:\n")
	for idx := range metrics {
		b.WriteString("- " + Line(&metrics[idx]) + "\n")
	}
	first := &metrics[0]
	if reads {
		fmt.Fprintf(&b, "To answer in a sentence, or to look before drawing, read one with "+
			"%s {\"metric\": %q}: it names every figure and point.\n",
			agenttoolports.MetricReadTool, first.Name)
	}
	if draws {
		fmt.Fprintf(&b, "When the person needs to see more than one figure, draw it in one "+
			"call: show_answer %s, with every figure written as {figure:metric}, never typed, "+
			"and at most %d figures in all.\n", Example(metrics), board.MaxReadings)
	}

	return strings.TrimRight(b.String(), "\n")
}

func Line(metric *boardcatalog.Metric) string {
	parts := []string{metric.Name + ": " + metric.Words}
	if len(metric.Periods) > 0 {
		periods := make([]string, 0, len(metric.Periods))
		for idx, period := range metric.Periods {
			if idx == 0 {
				periods = append(periods, string(period)+" (the default)")
				continue
			}
			periods = append(periods, string(period))
		}
		parts = append(parts, "periods "+strings.Join(periods, ", "))
	}
	figures := make([]string, 0, len(metric.Figures))
	for _, figure := range metric.Figures {
		figures = append(figures, figure.Name)
	}
	if len(figures) > 0 {
		parts = append(parts, "figures "+strings.Join(figures, ", "))
	}
	if metric.For != "" {
		parts = append(parts, "for one "+metric.For.Noun()+", by the id a tool returned")
	}

	return strings.Join(parts, "; ")
}

func Example(metrics []boardcatalog.Metric) string {
	groups := make([]string, 0, drawnGroups)
	for idx := range metrics {
		if len(groups) == drawnGroups {
			break
		}
		groups = append(groups, fmt.Sprintf(`{"title": "…", "metric": %q}`, metrics[idx].Name))
	}
	figure := ""
	if len(metrics[0].Figures) > 0 {
		figure = " {" + metrics[0].Figures[0].Name + ":" + metrics[0].Name + "}"
	}

	return fmt.Sprintf(`{"title": "…", "answer": "…%s …", "groups": [%s]}`, figure,
		strings.Join(groups, ", "))
}
