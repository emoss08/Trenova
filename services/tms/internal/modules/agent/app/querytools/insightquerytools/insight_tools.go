package insightquerytools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

const (
	maxInsightRows     = 50
	defaultInsightRows = 20
)

// insightReader is the slice of the insight service these tools use: the
// browse and the detail, both filtered to what the actor may see.
type insightReader interface {
	List(
		ctx context.Context,
		req insightports.BrowseInsightsRequest,
	) (*pagination.ListResult[*insight.Insight], error)
	GetDetail(
		ctx context.Context,
		req insightports.GetInsightDetailRequest,
	) (*insightports.InsightDetail, error)
}

// insightRow is a finding in the words the model needs: what was found,
// about whom, how bad, what the numbers are and what the detector suggests.
type insightRow struct {
	ID             string                 `json:"id"`
	Category       string                 `json:"category"`
	Severity       string                 `json:"severity"`
	Status         string                 `json:"status"`
	Subject        string                 `json:"subject,omitempty"`
	Headline       string                 `json:"headline"`
	Narrative      string                 `json:"narrative,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
	Metrics        []insightMetricRow     `json:"metrics,omitempty"`
	Links          []insightLinkRow       `json:"links,omitempty"`
	WindowStart    querytool.OptionalDate `json:"windowStart"`
	WindowEnd      querytool.OptionalDate `json:"windowEnd"`
	DetectedOn     querytool.OptionalDate `json:"detectedOn"`
	Stale          bool                   `json:"stale"`
	DismissReason  string                 `json:"dismissReason,omitempty"`
}

type insightMetricRow struct {
	Label     string           `json:"label"`
	Value     decimal.Decimal  `json:"value"`
	Unit      string           `json:"unit,omitempty"`
	Direction string           `json:"direction,omitempty"`
	Baseline  *decimal.Decimal `json:"baseline,omitempty"`
}

type insightLinkRow struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Count int    `json:"count,omitempty"`
}

func insightRowFrom(entity *insight.Insight, now int64) insightRow {
	row := insightRow{
		ID:             entity.ID.String(),
		Category:       string(entity.Category),
		Severity:       string(entity.Severity),
		Status:         string(entity.Status),
		Subject:        entity.Subject,
		Headline:       entity.Headline,
		Narrative:      entity.Narrative,
		Recommendation: entity.Recommendation,
		WindowStart:    querytool.RecordedDate(entity.WindowStart),
		WindowEnd:      querytool.RecordedDate(entity.WindowEnd),
		DetectedOn:     querytool.RecordedDate(entity.DetectedAt),
		Stale:          entity.IsStale(now),
		DismissReason:  entity.DismissReason,
	}
	if len(entity.Metrics) > 0 {
		row.Metrics = make([]insightMetricRow, 0, len(entity.Metrics))
		for _, metric := range entity.Metrics {
			row.Metrics = append(row.Metrics, insightMetricRow{
				Label:     metric.Label,
				Value:     metric.Value,
				Unit:      string(metric.Unit),
				Direction: string(metric.Direction),
				Baseline:  metric.Baseline,
			})
		}
	}
	if len(entity.Links) > 0 {
		row.Links = make([]insightLinkRow, 0, len(entity.Links))
		for _, link := range entity.Links {
			row.Links = append(
				row.Links,
				insightLinkRow{Label: link.Label, Path: link.Path, Count: link.Count},
			)
		}
	}

	return row
}

type listInsightsTool struct {
	insights insightReader
}

func NewListInsightsTool(insights insightReader) agenttoolports.AgentQueryTool {
	return &listInsightsTool{insights: insights}
}

func (t *listInsightsTool) Name() string { return "insight_list" }

func (t *listInsightsTool) Description() string {
	return "List insight detector findings: service slipping for a customer, cash tied " +
		"up in unbilled work, cost leaking at a location, compliance about to lapse. Each finding carries its measured numbers, " +
		"the records behind it and a recommendation. Active findings are returned unless " +
		"status says otherwise; narrow by category or severity. Only findings the person " +
		"you are working for may see are returned. Use insight_get for one finding's " +
		"history and the rule behind it, and insight_dismiss when a finding is not worth " +
		"acting on."
}

func (t *listInsightsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			querytool.ParamCategory: agenttoolschema.Enum(
				"Only findings in one area.",
				insightCategories,
			),
			"severity": agenttoolschema.Enum(
				"Only findings at one severity.", insightSeverities,
			),
			querytool.ParamStatus: agenttoolschema.Enum(
				"Which findings to list. Defaults to Active.", insightStatuses,
			),
			querytool.ParamLimit: querytool.LimitSchema(defaultInsightRows, maxInsightRows),
		},
		"additionalProperties": false,
	}
}

func (t *listInsightsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceInsight,
	})
}

func (t *listInsightsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	limit := querytool.OptionalInt(params.Params, "limit", defaultInsightRows)
	if limit <= 0 || limit > maxInsightRows {
		limit = maxInsightRows
	}

	req := insightports.BrowseInsightsRequest{
		TenantInfo: querytool.TenantOf(params),
		UserID:     params.Actor.UserID,
		Actor:      params.Actor,
		Limit:      limit,
	}

	clk := querytool.ClockFor(params)
	criteria := filtercatalog.NewCriteria("insights").At(clk)

	if raw := querytool.OptionalString(params.Params, "category"); raw != "" {
		category := insight.Category(raw)
		if !category.IsValid() {
			return nil, fmt.Errorf(
				"category %q is not one of %s",
				raw,
				strings.Join(insightCategoryNames(), ", "),
			)
		}
		req.Categories = []insight.Category{category}
		criteria.Field("category", raw)
	}

	if raw := querytool.OptionalString(params.Params, "severity"); raw != "" {
		severity := insight.Severity(raw)
		if !severity.IsValid() {
			return nil, fmt.Errorf(
				"severity %q is not one of %s",
				raw,
				strings.Join(insightSeverityNames(), ", "),
			)
		}
		req.Severities = []insight.Severity{severity}
		criteria.Field("severity", raw)
	}

	status := insight.StatusActive
	if raw := querytool.OptionalString(params.Params, "status"); raw != "" {
		status = insight.Status(raw)
		if !status.IsValid() {
			return nil, fmt.Errorf(
				"status %q is not one of %s",
				raw,
				strings.Join(insightStatusNames(), ", "),
			)
		}
	}
	req.Statuses = []insight.Status{status}
	criteria.Field("status", string(status))

	result, err := t.insights.List(ctx, req)
	if err != nil {
		return nil, err
	}

	now := clk.Instant()
	rows := make([]insightRow, 0, len(result.Items))
	for _, entity := range result.Items {
		rows = append(rows, insightRowFrom(entity, now))
	}

	outcome := querytool.SearchResult(criteria, rows, result.Total)
	outcome.TotalEstimated = !result.TotalExact

	return outcome, nil
}

// insightDetailRow is one finding with its trend and the rule behind it.
type insightDetailRow struct {
	insightRow

	Rule    insightRuleRow    `json:"rule"`
	History []insightTrendRow `json:"history"`
}

type insightRuleRow struct {
	Measures  string `json:"measures"`
	Threshold string `json:"threshold"`
	Excludes  string `json:"excludes"`
}

type insightTrendRow struct {
	DetectedOn querytool.OptionalDate `json:"detectedOn"`
	Severity   string                 `json:"severity"`
	Status     string                 `json:"status"`
	Metrics    []insightMetricRow     `json:"metrics,omitempty"`
}

type getInsightTool struct {
	insights insightReader
}

func NewGetInsightTool(insights insightReader) agenttoolports.AgentQueryTool {
	return &getInsightTool{insights: insights}
}

func (t *getInsightTool) Name() string { return "insight_get" }

func (t *getInsightTool) Description() string {
	return "Read one insight in full: the finding with its numbers and records, its " +
		"earlier runs, and the rule behind it. The earlier runs show whether it is " +
		"getting worse; the rule says what the detector measures, when it speaks and " +
		"what it passes over. Use the id from insight_list or from the run's subject."
}

func (t *getInsightTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"insightId": agenttoolschema.RecordIDText(permission.ResourceInsight,
				"The insight's id, from insight_list or this run's subject."),
		},
		"required":             []string{"insightId"},
		"additionalProperties": false,
	}
}

func (t *getInsightTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceInsight,
	})
}

func (t *getInsightTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	id, err := querytool.RequirePulid(params.Params, "insightId")
	if err != nil {
		return nil, err
	}

	detail, err := t.insights.GetDetail(ctx, insightports.GetInsightDetailRequest{
		ID:         id,
		UserID:     params.Actor.UserID,
		Actor:      params.Actor,
		TenantInfo: querytool.TenantOf(params),
	})
	if err != nil {
		return nil, err
	}

	now := querytool.ClockFor(params).Instant()
	row := insightDetailRow{
		insightRow: insightRowFrom(detail.Insight, now),
		Rule: insightRuleRow{
			Measures:  detail.Explanation.Measures,
			Threshold: detail.Explanation.Threshold,
			Excludes:  detail.Explanation.Excludes,
		},
		History: make([]insightTrendRow, 0, len(detail.History)),
	}
	for _, earlier := range detail.History {
		if earlier == nil {
			continue
		}
		row.History = append(row.History, insightTrendRow{
			DetectedOn: querytool.RecordedDate(earlier.DetectedAt),
			Severity:   string(earlier.Severity),
			Status:     string(earlier.Status),
			Metrics:    insightRowFrom(earlier, now).Metrics,
		})
	}

	return row, nil
}

func insightCategoryNames() []string {
	return []string{
		string(insight.CategoryServiceQuality),
		string(insight.CategoryCashFlow),
		string(insight.CategoryCostLeakage),
		string(insight.CategoryCompliance),
	}
}

func insightSeverityNames() []string {
	return []string{
		string(insight.SeverityInfo),
		string(insight.SeverityWarning),
		string(insight.SeverityCritical),
	}
}

func insightStatusNames() []string {
	return []string{
		string(insight.StatusActive),
		string(insight.StatusDismissed),
		string(insight.StatusResolved),
		string(insight.StatusSuperseded),
	}
}
