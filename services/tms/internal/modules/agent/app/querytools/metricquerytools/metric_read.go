package metricquerytools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

var metricPeriods = agenttoolschema.Source("board.period", boardcatalog.AllPeriods())

const (
	paramMetric = "metric"
	paramFor    = "for"
	paramPeriod = "period"
)

type boardReader interface {
	Check(ctx context.Context, req *assistantports.BoardFillRequest) error
	Fill(ctx context.Context, req *assistantports.BoardFillRequest) (*board.Filled, error)
}

type ReadMetricTool struct {
	boards boardReader
}

func NewReadMetricTool(boards assistantports.BoardFiller) agenttoolports.AgentQueryTool {
	return &ReadMetricTool{boards: boards}
}

func (t *ReadMetricTool) Name() string { return agenttoolports.MetricReadTool }

func (t *ReadMetricTool) Description() string {
	return "Read the figures and points of a metric Trenova keeps over time, to answer in a " +
		"sentence or to look before drawing it. search_tools names the metric for what a " +
		"question measures: a rate week by week, a ranking, today's figures. Give metric, " +
		"and for or period when the metric takes them. A figure the person may not see " +
		"comes back withheld, never as zero, and the result says how to draw the metric " +
		"with show_answer."
}

func (t *ReadMetricTool) SearchTerms() []string {
	return []string{"metric", "kpi", "trend", "figures over time", "rate by week", "ranking"}
}

func (t *ReadMetricTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramMetric: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: metricWords(),
			},
			paramFor: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "The id a tool returned for the record the metric " +
					"reads, when it reads one.",
			},
			paramPeriod: agenttoolschema.Enum(
				"The time the metric covers, when it takes one; left out, the metric's own.",
				metricPeriods,
			),
		},
		toolschema.KeyRequired:             []string{paramMetric},
		toolschema.KeyAdditionalProperties: false,
	}
}

func metricWords() string {
	metrics := boardregistry.Catalog().Metrics()
	lines := make([]string, 0, len(metrics))
	for idx := range metrics {
		lines = append(lines, metrics[idx].Name+": "+metrics[idx].Words)
	}

	return "The metric, by name. " + strings.Join(lines, ". ") + "."
}

func (t *ReadMetricTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceAssistant,
		Rationale: "Reads one metric Trenova keeps, through the board's own reader and with " +
			"the board's checks of the caller's access to that metric, its records and its " +
			"plan; it changes nothing and sends nothing.",
	})
}

func (t *ReadMetricTool) Readable(
	ctx context.Context,
	actor *permissionports.RequestActor,
	access permission.FieldSensitivity,
	metric string,
) bool {
	if actor == nil || t.boards == nil {
		return false
	}

	return t.boards.Check(ctx, &assistantports.BoardFillRequest{
		Actor: actor,
		Board: layout(&ask{metric: metric}),
		Agent: &assistantports.BoardAgentRead{DataAccess: access},
	}) == nil
}

type ask struct {
	metric string
	forID  pulid.ID
	period string
}

func layout(a *ask) *board.Board {
	return &board.Board{
		Title: a.metric,
		Place: board.PlaceConversation,
		Blocks: []board.Block{{
			Type:   board.BlockMetric,
			Metric: a.metric,
			For:    a.forID,
			Period: a.period,
		}},
	}
}

func (t *ReadMetricTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	metric, named, err := namedMetric(params.Params)
	if err != nil {
		return nil, err
	}
	asked, notes, err := askOf(&metric, params.Params)
	if err != nil {
		return nil, err
	}
	if named != "" {
		notes = append([]string{named}, notes...)
	}
	req := &assistantports.BoardFillRequest{
		Actor:    params.Actor,
		ThreadID: params.ThreadID,
		Board:    layout(asked),
		Agent:    &assistantports.BoardAgentRead{DataAccess: params.DataAccessCeiling},
	}
	if err = t.boards.Check(ctx, req); err != nil {
		if mistake, ok := errors.AsType[*agentruntimeports.PublishMistake](err); ok {
			return nil, refusal("The %s metric was not read: %s.", metric.Name, mistake.Reason)
		}

		return nil, err
	}
	filled, err := t.boards.Fill(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(filled.Blocks) == 0 || filled.Blocks[0].Metric == nil ||
		filled.Blocks[0].State == board.StateUnreadable {
		return nil, refusal("The %s metric could not be read now.", metric.Name)
	}

	return resultOf(&resultInput{
		metric: &metric,
		ask:    asked,
		block:  &filled.Blocks[0],
		notes:  notes,
		loc:    querytool.ClockFor(params).Location(),
	}), nil
}

func namedMetric(params map[string]any) (boardcatalog.Metric, string, error) {
	name, err := querytool.RequireString(params, paramMetric)
	if err != nil {
		return boardcatalog.Metric{}, "", invalid(
			paramMetric,
			errortypes.ErrRequired,
			"Give the metric to read, by name.",
		)
	}
	if metric, known := boardregistry.Catalog().MetricNamed(name); known {
		return metric, "", nil
	}
	if metric, figure, known := boardregistry.Catalog().FigureNamed(name); known {
		return metric, fmt.Sprintf("%s is a figure of the %s metric, so %s was read; write "+
			"it as {%s:%s}.", name, metric.Name, metric.Name, figure.Name, metric.Name), nil
	}

	return boardcatalog.Metric{}, "", invalid(paramMetric,
		errortypes.ErrInvalid, "%q is not a metric Trenova keeps. Ask search_tools with "+
			"what it measures, or give one of %s.", name,
		stringutils.JoinAlternatives(boardregistry.Catalog().MetricNames()))
}

func askOf(metric *boardcatalog.Metric, params map[string]any) (*ask, []string, error) {
	out := &ask{metric: metric.Name, period: string(metric.DefaultPeriod())}
	notes := make([]string, 0, 2)
	if period := querytool.OptionalString(params, paramPeriod); period != "" {
		switch {
		case metric.Takes(boardcatalog.Period(period)):
			out.period = period
		case out.period != "":
			notes = append(notes, fmt.Sprintf("%s does not take the period %s, so it was "+
				"read for %s.", metric.Name, period, out.period))
		default:
			notes = append(notes, fmt.Sprintf("%s takes no period, so %s was left out.",
				metric.Name, period))
		}
	}
	text := querytool.OptionalString(params, paramFor)
	if metric.For == "" {
		if text != "" {
			notes = append(notes, fmt.Sprintf("%s reads no single record, so for was left "+
				"out.", metric.Name))
		}

		return out, notes, nil
	}
	if text == "" {
		if metric.ForNeeded {
			return nil, nil, invalid(paramFor, errortypes.ErrRequired,
				"%s reads one %s: give for, the id a tool returned for it.", metric.Name,
				metric.For.Noun())
		}

		return out, notes, nil
	}
	id, err := pulid.Parse(text)
	kind, known := recordkind.OfIDPrefix(id.Prefix())
	if err != nil || id.IsNil() || !known || kind != metric.For {
		return nil, nil, invalid(paramFor, errortypes.ErrInvalid,
			"%q is not the id of a %s; %s reads one %s, by the id a tool returned for it.",
			text, metric.For.Noun(), metric.Name, metric.For.Noun())
	}
	out.forID = id

	return out, notes, nil
}

func refusal(format string, args ...any) error {
	return errortypes.NewBusinessError(fmt.Sprintf(format, args...))
}

func invalid(field string, code errortypes.ErrorCode, format string, args ...any) error {
	return errortypes.NewValidationError(field, code, fmt.Sprintf(format, args...))
}
