package aianalyticsresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/shopspring/decimal"
)

func analyticsRequest(
	tenant pagination.TenantInfo,
	input gqlmodel.AIAnalyticsInput,
) (*aianalyticsports.AnalyticsRequest, error) {
	agentID, err := base.OptionalID(input.AgentDefinitionID)
	if err != nil {
		return nil, err
	}

	req := &aianalyticsports.AnalyticsRequest{
		TenantInfo:        tenant,
		Window:            input.Window,
		CompareBy:         input.CompareBy,
		AgentDefinitionID: agentID,
	}
	if input.Task != nil {
		req.Task = *input.Task
	}
	if input.Timezone != nil {
		req.Timezone = *input.Timezone
	}

	return req, nil
}

func optionalInt(value *int64) *int {
	if value == nil {
		return nil
	}
	out := int(*value)

	return &out
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func optionalDecimal(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	out := value.String()

	return &out
}

func providerModel(ref aianalyticsports.ProviderRef) *gqlmodel.AIAnalyticsProvider {
	out := &gqlmodel.AIAnalyticsProvider{
		ID:        optionalString(ref.ID.String()),
		Name:      ref.Name,
		ShortName: ref.ShortName,
		BaseURL:   ref.BaseURL,
		Enabled:   ref.Enabled,
		Removed:   ref.Removed,
	}
	if ref.Kind != "" {
		kind := ref.Kind
		out.Kind = &kind
	}

	return out
}

func providerModels(refs []aianalyticsports.ProviderRef) []*gqlmodel.AIAnalyticsProvider {
	out := make([]*gqlmodel.AIAnalyticsProvider, 0, len(refs))
	for _, ref := range refs {
		out = append(out, providerModel(ref))
	}

	return out
}

var segmentTargets = map[aianalyticsports.SegmentTarget]gqlmodel.AIAnalyticsTarget{
	aianalyticsports.SegmentTargetSeries:   gqlmodel.AIAnalyticsTargetSeries,
	aianalyticsports.SegmentTargetProvider: gqlmodel.AIAnalyticsTargetProvider,
}

func segmentModels(segments []aianalyticsports.Segment) []*gqlmodel.AIAnalyticsSegment {
	out := make([]*gqlmodel.AIAnalyticsSegment, 0, len(segments))
	for _, segment := range segments {
		item := &gqlmodel.AIAnalyticsSegment{
			Text:       segment.Text,
			Strong:     segment.Strong,
			Tone:       optionalString(segment.Tone),
			SeriesKey:  optionalString(segment.SeriesKey),
			ProviderID: optionalString(segment.ProviderID.String()),
		}
		if target, ok := segmentTargets[segment.Target]; ok {
			item.Target = &target
		}
		out = append(out, item)
	}

	return out
}

func totalsModel(totals *aianalyticsports.Totals) *gqlmodel.AIAnalyticsTotals {
	return &gqlmodel.AIAnalyticsTotals{
		Runs:                     totals.Runs,
		ModelCalls:               totals.ModelCalls,
		Completed:                totals.Completed,
		HandedOff:                totals.HandedOff,
		Stopped:                  totals.Stopped,
		Failed:                   totals.Failed,
		CompletedRate:            totals.CompletedRate,
		MedianRunMs:              optionalInt(totals.MedianRunMs),
		P95RunMs:                 optionalInt(totals.P95RunMs),
		ToolCalls:                totals.ToolCalls,
		ToolCallsFailed:          totals.ToolCallsFailed,
		ToolFailuresModel:        totals.ToolFailuresModel,
		ToolFailureRate:          totals.ToolFailureRate,
		ModelShareOfToolFailures: totals.ModelShareOfToolFailures,
		CostUsd:                  totals.CostUSD.String(),
		PricedRuns:               totals.PricedRuns,
		CostPerCompletedRun:      optionalDecimal(totals.CostPerCompletedRun),
		ToolFailureAlarm:         totals.ToolFailureAlarm,
	}
}

func seriesModel(series *aianalyticsports.Series) *gqlmodel.AIAnalyticsSeries {
	runTimes := make([]*gqlmodel.AIAnalyticsRunTime, 0, len(series.RunTimes))
	for _, runTime := range series.RunTimes {
		runTimes = append(runTimes, &gqlmodel.AIAnalyticsRunTime{
			Outcome:  runTime.Outcome,
			Runs:     runTime.Runs,
			P25Ms:    int(runTime.P25Ms),
			MedianMs: int(runTime.P50Ms),
			P75Ms:    int(runTime.P75Ms),
			P95Ms:    int(runTime.P95Ms),
		})
	}

	causes := make([]*gqlmodel.AIAnalyticsCauseCount, 0, len(series.ToolFailuresByCause))
	for _, cause := range series.ToolFailuresByCause {
		causes = append(causes, causeModel(cause))
	}

	return &gqlmodel.AIAnalyticsSeries{
		Key:                 series.Key,
		Label:               series.Label,
		ShortLabel:          series.ShortLabel,
		ModelKey:            optionalString(series.ModelKey),
		ProviderID:          optionalString(series.ProviderID.String()),
		Few:                 series.Few,
		ColorSlot:           series.ColorSlot,
		Models:              series.Models,
		Providers:           providerModels(series.Providers),
		Runs:                series.Runs,
		Completed:           series.Completed,
		HandedOff:           series.HandedOff,
		Stopped:             series.Stopped,
		Failed:              series.Failed,
		CompletedRate:       series.CompletedRate,
		MedianRunMs:         optionalInt(series.MedianRunMs),
		P95RunMs:            optionalInt(series.P95RunMs),
		ModelCalls:          series.ModelCalls,
		ToolCalls:           series.ToolCalls,
		ToolCallsFailed:     series.ToolCallsFailed,
		ToolFailuresModel:   series.ToolFailuresModel,
		ToolFailureRate:     series.ToolFailureRate,
		ModelMistakeRate:    series.ModelMistakeRate,
		ToolFailuresByCause: causes,
		CostUsd:             series.CostUSD.String(),
		PricedRuns:          series.PricedRuns,
		CostPerCompletedRun: optionalDecimal(series.CostPerCompletedRun),
		FirstTokenMedianMs:  optionalInt(series.FirstTokenMedianMs),
		LastRanAt:           optionalInt(series.LastRanAt),
		Quiet:               series.Quiet,
		RunTimes:            runTimes,
		Decided:             series.Decided,
		Accepted:            series.Accepted,
		Modified:            series.Modified,
		Rejected:            series.Rejected,
		ApprovalRate:        series.ApprovalRate,
		Rated:               series.Rated,
		Helpful:             series.Helpful,
		HelpfulRate:         series.HelpfulRate,
	}
}

func toolModels(tools []aianalyticsports.ToolRow) []*gqlmodel.AIAnalyticsTool {
	out := make([]*gqlmodel.AIAnalyticsTool, 0, len(tools))
	for idx := range tools {
		tool := &tools[idx]
		cells := make([]*gqlmodel.AIAnalyticsToolCell, 0, len(tool.Cells))
		for _, cell := range tool.Cells {
			cells = append(cells, &gqlmodel.AIAnalyticsToolCell{
				SeriesKey:   cell.SeriesKey,
				Few:         cell.Few,
				Calls:       cell.Calls,
				Failed:      cell.Failed,
				FailureRate: cell.FailureRate,
			})
		}
		out = append(out, &gqlmodel.AIAnalyticsTool{
			Name:        tool.Name,
			Calls:       tool.Calls,
			Failed:      tool.Failed,
			FailureRate: tool.FailureRate,
			Cells:       cells,
		})
	}

	return out
}

func taskModels(tasks []aianalyticsports.TaskRow) []*gqlmodel.AIAnalyticsTask {
	out := make([]*gqlmodel.AIAnalyticsTask, 0, len(tasks))
	for idx := range tasks {
		task := &tasks[idx]
		cells := make([]*gqlmodel.AIAnalyticsTaskCell, 0, len(task.Cells))
		for _, cell := range task.Cells {
			cells = append(cells, &gqlmodel.AIAnalyticsTaskCell{
				SeriesKey:     cell.SeriesKey,
				Few:           cell.Few,
				Runs:          cell.Runs,
				Completed:     cell.Completed,
				CompletedRate: cell.CompletedRate,
			})
		}
		item := &gqlmodel.AIAnalyticsTask{
			Task:             task.Task,
			CurrentSeriesKey: optionalString(task.CurrentSeriesKey),
			BestSeriesKey:    optionalString(task.BestSeriesKey),
			Cells:            cells,
		}
		if task.Current != nil {
			item.Current = providerModel(*task.Current)
		}
		if task.Suggestion != nil {
			item.Suggestion = &gqlmodel.AIAnalyticsRouteSuggestion{
				Provider:   providerModel(task.Suggestion.Provider),
				SeriesKey:  task.Suggestion.SeriesKey,
				GainPoints: task.Suggestion.GainPoints,
			}
		}
		out = append(out, item)
	}

	return out
}

func providerHealthModels(
	rows []aianalyticsports.ProviderHealth,
) []*gqlmodel.AIAnalyticsProviderHealth {
	out := make([]*gqlmodel.AIAnalyticsProviderHealth, 0, len(rows))
	for idx := range rows {
		row := &rows[idx]
		kinds := make([]*gqlmodel.AIAnalyticsErrorCount, 0, len(row.ErrorsByKind))
		for _, kind := range row.ErrorsByKind {
			kinds = append(
				kinds,
				&gqlmodel.AIAnalyticsErrorCount{Kind: kind.Error, Calls: kind.Calls},
			)
		}
		out = append(out, &gqlmodel.AIAnalyticsProviderHealth{
			Key:                row.Key,
			Provider:           providerModel(row.Provider),
			ModelKey:           row.ModelKey,
			Model:              row.Model,
			LastCalledAt:       optionalInt(row.LastCalledAt),
			Quiet:              row.Quiet,
			Calls:              row.Calls,
			Errors:             row.Errors,
			ErrorRate:          row.ErrorRate,
			ErrorAlarm:         row.ErrorAlarm,
			FellToNext:         row.FellToNext,
			FellToNextAlarm:    row.FellToNextAlarm,
			FirstTokenMedianMs: optionalInt(row.FirstTokenMedianMs),
			ErrorsByKind:       kinds,
		})
	}

	return out
}

func seriesModels(series []aianalyticsports.Series) []*gqlmodel.AIAnalyticsSeries {
	out := make([]*gqlmodel.AIAnalyticsSeries, 0, len(series))
	for idx := range series {
		out = append(out, seriesModel(&series[idx]))
	}

	return out
}

func causeModel(cause aianalyticsports.CauseCount) *gqlmodel.AIAnalyticsCauseCount {
	return &gqlmodel.AIAnalyticsCauseCount{
		Cause: cause.Cause,
		Blame: cause.Blame,
		Calls: cause.Calls,
	}
}

func analyticsModel(analytics *aianalyticsports.Analytics) *gqlmodel.AgentAnalytics {
	trend := make([]*gqlmodel.AIAnalyticsTrendPoint, 0, len(analytics.Trend))
	for _, point := range analytics.Trend {
		trend = append(trend, &gqlmodel.AIAnalyticsTrendPoint{
			SeriesKey:       point.SeriesKey,
			BucketStart:     int(point.BucketStart),
			Runs:            point.Runs,
			CompletedRate:   point.CompletedRate,
			ToolFailureRate: point.ToolFailureRate,
			MedianRunMs:     optionalInt(point.MedianRunMs),
		})
	}
	events := make([]*gqlmodel.AIAnalyticsEvent, 0, len(analytics.Events))
	for _, event := range analytics.Events {
		events = append(events, &gqlmodel.AIAnalyticsEvent{
			At:       int(event.At),
			Kind:     event.Kind,
			Label:    event.Label,
			Provider: providerModel(event.Provider),
		})
	}

	return &gqlmodel.AgentAnalytics{
		Window:        analytics.Window,
		CompareBy:     analytics.CompareBy,
		Since:         int(analytics.Since),
		Until:         int(analytics.Until),
		BucketSeconds: int(analytics.BucketSeconds),
		TrendBuckets:  intutils.Ints(analytics.TrendBuckets),
		RefreshedAt:   optionalInt(analytics.RefreshedAt),
		Summary:       segmentModels(analytics.Summary),
		RunTimeNote:   segmentModels(analytics.RunTimeNote),
		Totals:        totalsModel(&analytics.Totals),
		Series:        seriesModels(analytics.Series),
		Deployments:   seriesModels(analytics.Deployments),
		Trend:         trend,
		Events:        events,
		Tools:         toolModels(analytics.Tools),
		Tasks:         taskModels(analytics.Tasks),
		Providers:     providerHealthModels(analytics.Providers),
		SameModel:     sameModelModel(analytics.SameModel),
	}
}

func sameModelModel(pair *aianalyticsports.SameModel) *gqlmodel.AIAnalyticsSameModel {
	if pair == nil {
		return nil
	}

	return &gqlmodel.AIAnalyticsSameModel{
		ModelKey:  pair.ModelKey,
		BetterKey: pair.BetterKey,
		WorseKey:  pair.WorseKey,
	}
}

func toolFailureModel(
	detail *aianalyticsports.ToolFailureDetail,
) *gqlmodel.AgentAnalyticsToolFailures {
	causes := make([]*gqlmodel.AIAnalyticsCauseCount, 0, len(detail.Causes))
	for _, cause := range detail.Causes {
		causes = append(causes, causeModel(cause))
	}
	recent := make([]*gqlmodel.AIAnalyticsFailureExample, 0, len(detail.Recent))
	for _, example := range detail.Recent {
		item := &gqlmodel.AIAnalyticsFailureExample{
			UnitID:   example.UnitID,
			Source:   example.Source,
			ToolName: example.ToolName,
			Message:  example.Message,
			At:       int(example.At),
		}
		if example.Agent != nil {
			item.AgentID = optionalString(example.Agent.ID.String())
			item.AgentName = optionalString(example.Agent.Name)
		}
		recent = append(recent, item)
	}

	return &gqlmodel.AgentAnalyticsToolFailures{
		SeriesKey: detail.SeriesKey,
		ToolName:  detail.ToolName,
		Calls:     detail.Calls,
		Failed:    detail.Failed,
		Causes:    causes,
		Recent:    recent,
		Providers: providerModels(detail.Providers),
	}
}
