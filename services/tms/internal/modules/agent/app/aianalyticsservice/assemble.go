package aianalyticsservice

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	maxTools            = 12
	deploymentSeparator = "|"
)

type assembly struct {
	locale  i18n.Locale
	request *aianalyticsports.AnalyticsRequest
	since   int64
	until   int64
	data    *readings
}

type providerLookup struct {
	locale i18n.Locale
	byID   map[pulid.ID]*aiprovider.Provider
}

func providerIndex(locale i18n.Locale, providers []*aiprovider.Provider) providerLookup {
	index := providerLookup{
		locale: locale,
		byID:   make(map[pulid.ID]*aiprovider.Provider, len(providers)),
	}
	for _, provider := range providers {
		index.byID[provider.ID] = provider
	}

	return index
}

func (l providerLookup) get(id pulid.ID) (*aiprovider.Provider, bool) {
	provider, ok := l.byID[id]
	return provider, ok
}

func (l providerLookup) ref(id string) aianalyticsports.ProviderRef {
	if id == "" {
		return aianalyticsports.ProviderRef{
			Name:    i18n.Translate(l.locale, "No provider"),
			Removed: true,
		}
	}
	provider, ok := l.byID[pulid.ID(id)]
	if !ok {
		return aianalyticsports.ProviderRef{
			ID:      pulid.ID(id),
			Name:    i18n.Translate(l.locale, "Removed provider"),
			Removed: true,
		}
	}

	return aianalyticsports.ProviderRef{
		ID:      provider.ID,
		Name:    provider.Name,
		Kind:    provider.Kind,
		BaseURL: provider.BaseURL,
		Enabled: provider.Enabled,
	}
}

func rate(part, whole int) *float64 {
	if whole <= 0 {
		return nil
	}
	value := float64(part) / float64(whole)

	return &value
}

func ptr[T any](value T) *T {
	return &value
}

func costPerCompleted(cost decimal.Decimal, priced, completed int) *decimal.Decimal {
	if priced == 0 || completed == 0 {
		return nil
	}
	value := cost.Div(decimal.NewFromInt(int64(completed))).Round(6)

	return &value
}

type keyFunc func(aianalyticsports.DeploymentKey) string

func groupKeyOf(compareBy aianalytics.CompareBy) keyFunc {
	if compareBy == aianalytics.CompareByProvider {
		return func(key aianalyticsports.DeploymentKey) string { return key.ProviderID }
	}

	return func(key aianalyticsports.DeploymentKey) string { return key.ModelKey }
}

func deploymentKeyOf(key aianalyticsports.DeploymentKey) string {
	return key.ModelKey + deploymentSeparator + key.ProviderID
}

func keyOfGrouping(grouping aianalytics.Grouping) keyFunc {
	switch grouping {
	case aianalytics.GroupingProvider:
		return groupKeyOf(aianalytics.CompareByProvider)
	case aianalytics.GroupingDeployment:
		return deploymentKeyOf
	default:
		return groupKeyOf(aianalytics.CompareByModel)
	}
}

func assemble(a *assembly) *aianalyticsports.Analytics {
	providers := providerIndex(a.locale, a.data.providers)
	compareBy := a.request.CompareBy
	grouping := compareBy.Grouping()
	groupOf := groupKeyOf(compareBy)

	palette := buildPalette(a.data)
	series := buildSeries(&seriesInput{
		grouping:  grouping,
		tallies:   tallyBy(a.data, groupOf),
		durations: indexDurations(a.data.durations, groupOf),
		providers: providers,
		palette:   palette,
	})
	deployments := buildSeries(&seriesInput{
		grouping:  aianalytics.GroupingDeployment,
		tallies:   tallyBy(a.data, deploymentKeyOf),
		durations: indexDurations(a.data.deploymentDurations, deploymentKeyOf),
		providers: providers,
		palette:   palette,
	})

	out := &aianalyticsports.Analytics{
		Window:        a.request.Window,
		CompareBy:     compareBy,
		Since:         a.since,
		Until:         a.until,
		BucketSeconds: a.request.Window.BucketSeconds(),
		RefreshedAt:   a.data.state.refreshedAt,
		Totals:        buildTotals(a.data.totals),
		Series:        series,
		Deployments:   deployments,
		Trend:         buildTrend(a.data.trend, groupOf),
		Tools:         buildTools(a.data.toolCells, groupOf),
		Tasks: buildTasks(&taskInput{
			rows:      a.data.taskCells,
			ordered:   a.data.providers,
			locale:    a.locale,
			compareBy: compareBy,
		}),
		Providers: buildProviderHealth(a.data, providers),
		Events:    buildEvents(a.data, providers, a.since),
	}
	out.Summary = summarize(&narration{
		locale:      a.locale,
		window:      a.request.Window,
		compareBy:   compareBy,
		series:      series,
		deployments: deployments,
		providers:   providers,
	})
	out.RunTimeNote = runTimeNote(deployments, providers)

	return out
}

func buildTotals(row *aianalyticsports.TotalsRow) aianalyticsports.Totals {
	if row == nil {
		return aianalyticsports.Totals{CostUSD: decimal.Zero}
	}
	failed := row.ToolFailuresModel + row.ToolFailuresTool
	totals := aianalyticsports.Totals{
		Runs:                     row.Runs,
		ModelCalls:               row.ModelCalls,
		Completed:                row.Completed,
		HandedOff:                row.HandedOff,
		Stopped:                  row.Stopped,
		Failed:                   row.Failed,
		CompletedRate:            rate(row.Completed, row.Runs),
		ToolCalls:                row.ToolCalls,
		ToolCallsFailed:          failed,
		ToolFailuresModel:        row.ToolFailuresModel,
		ToolFailureRate:          rate(failed, row.ToolCalls),
		ModelShareOfToolFailures: rate(row.ToolFailuresModel, failed),
		CostUSD:                  row.CostUSD,
		PricedRuns:               row.PricedRuns,
		CostPerCompletedRun:      costPerCompleted(row.CostUSD, row.PricedRuns, row.Completed),
	}
	if row.Completion.HasAny {
		totals.MedianRunMs = ptr(row.Completion.P50Ms)
		totals.P95RunMs = ptr(row.Completion.P95Ms)
	}

	return totals
}

type served struct {
	key  aianalyticsports.DeploymentKey
	runs int
}

type tally struct {
	key        string
	counts     aianalyticsports.Counts
	served     []served
	accepted   int
	modified   int
	rejected   int
	rated      int
	helpful    int
	causes     map[aianalytics.Cause]int
	firstToken aianalytics.Histogram
}

type tallies struct {
	order []string
	byKey map[string]*tally
}

func (t *tallies) entry(key string) *tally {
	item, ok := t.byKey[key]
	if !ok {
		item = &tally{key: key, counts: aianalyticsports.Counts{CostUSD: decimal.Zero}}
		t.byKey[key] = item
		t.order = append(t.order, key)
	}

	return item
}

func (t *tallies) lookup(key string) *tally {
	return t.byKey[key]
}

func addCounts(into *aianalyticsports.Counts, from *aianalyticsports.Counts) {
	into.Runs += from.Runs
	into.Completed += from.Completed
	into.HandedOff += from.HandedOff
	into.Stopped += from.Stopped
	into.Failed += from.Failed
	into.ModelCalls += from.ModelCalls
	into.FailedCalls += from.FailedCalls
	into.ToolCalls += from.ToolCalls
	into.ToolFailuresModel += from.ToolFailuresModel
	into.ToolFailuresTool += from.ToolFailuresTool
	into.CostUSD = into.CostUSD.Add(from.CostUSD)
	into.PricedRuns += from.PricedRuns
	into.LastRanAt = max(into.LastRanAt, from.LastRanAt)
}

func tallyBy(data *readings, keyOf keyFunc) *tallies {
	out := &tallies{byKey: make(map[string]*tally, len(data.deployments))}
	for idx := range data.deployments {
		row := &data.deployments[idx]
		item := out.entry(keyOf(row.DeploymentKey))
		addCounts(&item.counts, &row.Counts)
		item.served = append(item.served, served{key: row.DeploymentKey, runs: row.Runs})
	}
	for _, row := range data.decisions {
		if item := out.lookup(keyOf(row.DeploymentKey)); item != nil {
			item.accepted += row.Accepted
			item.modified += row.Modified
			item.rejected += row.Rejected
		}
	}
	for _, row := range data.ratings {
		if item := out.lookup(keyOf(row.DeploymentKey)); item != nil {
			item.rated += row.Rated
			item.helpful += row.Helpful
		}
	}
	for _, row := range data.histograms {
		if item := out.lookup(keyOf(row.DeploymentKey)); item != nil {
			item.firstToken.AddBucket(row.Bucket, row.Count)
		}
	}
	for _, row := range data.toolCells {
		if row.Cause == "" {
			continue
		}
		item := out.lookup(keyOf(row.DeploymentKey))
		if item == nil {
			continue
		}
		if item.causes == nil {
			item.causes = make(map[aianalytics.Cause]int, len(aianalytics.AllCauses()))
		}
		item.causes[row.Cause] += row.Calls
	}

	return out
}

func indexDurations(
	rows []aianalyticsports.OutcomeDurationRow,
	keyOf keyFunc,
) map[string][]aianalyticsports.OutcomeDurationRow {
	out := make(map[string][]aianalyticsports.OutcomeDurationRow, len(rows))
	for _, row := range rows {
		key := keyOf(row.DeploymentKey)
		out[key] = append(out[key], row)
	}

	return out
}

type seriesInput struct {
	grouping  aianalytics.Grouping
	tallies   *tallies
	durations map[string][]aianalyticsports.OutcomeDurationRow
	providers providerLookup
	palette   palette
}

type palette struct {
	models    map[string]int
	providers map[string]int
}

func buildPalette(data *readings) palette {
	models := make([]string, 0, len(data.palette)+len(data.deployments))
	providers := make([]string, 0, len(data.palette)+len(data.deployments))
	add := func(key aianalyticsports.DeploymentKey) {
		if key.ModelKey != "" {
			models = append(models, key.ModelKey)
		}
		if key.ProviderID != "" {
			providers = append(providers, key.ProviderID)
		}
	}
	for _, key := range data.palette {
		add(key)
	}
	for idx := range data.deployments {
		add(data.deployments[idx].DeploymentKey)
	}

	return palette{
		models:    aianalytics.AssignPaletteSlots(models),
		providers: aianalytics.AssignPaletteSlots(providers),
	}
}

func (p palette) slot(grouping aianalytics.Grouping, series *aianalyticsports.Series) int {
	if grouping == aianalytics.GroupingModel {
		return aianalytics.PaletteSlot(p.models, series.ModelKey)
	}

	return aianalytics.PaletteSlot(p.providers, series.ProviderID.String())
}

func buildSeries(in *seriesInput) []aianalyticsports.Series {
	out := make([]aianalyticsports.Series, 0, len(in.tallies.order))
	for _, key := range in.tallies.order {
		item := in.tallies.byKey[key]
		series := seriesFromTally(item, runTimes(in.durations[key]))
		identify(&series, in.grouping, item, in.providers)
		series.ColorSlot = in.palette.slot(in.grouping, &series)
		out = append(out, series)
	}
	slices.SortStableFunc(out, func(left, right aianalyticsports.Series) int {
		return cmp.Or(cmp.Compare(right.Runs, left.Runs), cmp.Compare(left.Label, right.Label))
	})

	return out
}

func seriesFromTally(item *tally, times []aianalyticsports.RunTime) aianalyticsports.Series {
	counts := &item.counts
	failed := counts.ToolFailuresModel + counts.ToolFailuresTool
	series := aianalyticsports.Series{
		Key:                 item.key,
		Few:                 !aianalytics.RanksByRuns(counts.Runs),
		Runs:                counts.Runs,
		Completed:           counts.Completed,
		HandedOff:           counts.HandedOff,
		Stopped:             counts.Stopped,
		Failed:              counts.Failed,
		CompletedRate:       rate(counts.Completed, counts.Runs),
		ModelCalls:          counts.ModelCalls,
		ToolCalls:           counts.ToolCalls,
		ToolCallsFailed:     failed,
		ToolFailuresModel:   counts.ToolFailuresModel,
		ToolFailureRate:     rate(failed, counts.ToolCalls),
		ModelMistakeRate:    rate(counts.ToolFailuresModel, failed),
		ToolFailuresByCause: causeCounts(item.causes),
		CostUSD:             counts.CostUSD,
		PricedRuns:          counts.PricedRuns,
		CostPerCompletedRun: costPerCompleted(counts.CostUSD, counts.PricedRuns, counts.Completed),
		RunTimes:            times,
		Providers:           []aianalyticsports.ProviderRef{},
		Models:              []string{},
		Accepted:            item.accepted,
		Modified:            item.modified,
		Rejected:            item.rejected,
		Decided:             item.accepted + item.modified + item.rejected,
		Rated:               item.rated,
		Helpful:             item.helpful,
	}
	series.ApprovalRate = rate(series.Accepted, series.Decided)
	series.HelpfulRate = rate(series.Helpful, series.Rated)
	if counts.LastRanAt > 0 {
		series.LastRanAt = ptr(counts.LastRanAt)
	}
	for _, runTime := range times {
		if runTime.Outcome == aianalytics.OutcomeCompleted {
			series.MedianRunMs = ptr(runTime.P50Ms)
			series.P95RunMs = ptr(runTime.P95Ms)
		}
	}
	if median, found := item.firstToken.Percentile(0.5); found {
		series.FirstTokenMedianMs = ptr(median)
	}

	return series
}

func causeCounts(causes map[aianalytics.Cause]int) []aianalyticsports.CauseCount {
	out := make([]aianalyticsports.CauseCount, 0, len(causes))
	for _, cause := range aianalytics.AllCauses() {
		if calls := causes[cause]; calls > 0 {
			out = append(
				out,
				aianalyticsports.CauseCount{Cause: cause, Blame: cause.Blame(), Calls: calls},
			)
		}
	}

	return out
}

func identify(
	series *aianalyticsports.Series,
	grouping aianalytics.Grouping,
	item *tally,
	providers providerLookup,
) {
	rows := slices.Clone(item.served)
	slices.SortStableFunc(rows, func(left, right served) int {
		return cmp.Compare(right.runs, left.runs)
	})
	seenProviders := make(map[string]struct{}, len(rows))
	seenModels := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seenProviders[row.key.ProviderID]; !ok {
			seenProviders[row.key.ProviderID] = struct{}{}
			series.Providers = append(series.Providers, providers.ref(row.key.ProviderID))
		}
		if grouping == aianalytics.GroupingModel || row.key.ModelKey == "" {
			continue
		}
		if _, ok := seenModels[row.key.ModelKey]; ok {
			continue
		}
		seenModels[row.key.ModelKey] = struct{}{}
		series.Models = append(series.Models, providers.modelName(row.key.ModelKey).Label)
	}

	switch grouping {
	case aianalytics.GroupingProvider:
		name := providers.ref(item.key).Name
		series.Label, series.ShortLabel = name, name
		series.ProviderID = pulid.ID(item.key)
	case aianalytics.GroupingDeployment:
		key := item.served[0].key
		name := providers.ref(key.ProviderID).Name
		series.Label, series.ShortLabel = name, name
		series.ModelKey = key.ModelKey
		series.ProviderID = pulid.ID(key.ProviderID)
	default:
		name := providers.modelName(item.key)
		series.Label, series.ShortLabel = name.Label, name.Short
		series.ModelKey = item.key
	}
}

func (l providerLookup) modelName(key string) aianalytics.ModelName {
	if key == "" {
		label := i18n.Translate(l.locale, "Unknown model")
		return aianalytics.ModelName{Label: label, Short: label}
	}

	return aianalytics.NameModel(key)
}

var outcomeOrder = map[aianalytics.Outcome]int{
	aianalytics.OutcomeCompleted: 0,
	aianalytics.OutcomeHandedOff: 1,
	aianalytics.OutcomeStopped:   2,
	aianalytics.OutcomeFailed:    3,
}

func runTimes(rows []aianalyticsports.OutcomeDurationRow) []aianalyticsports.RunTime {
	out := make([]aianalyticsports.RunTime, 0, len(rows))
	for _, row := range rows {
		if !row.HasAny {
			continue
		}
		out = append(out, aianalyticsports.RunTime{
			Outcome: row.Outcome,
			Runs:    row.Count,
			P25Ms:   row.P25Ms,
			P50Ms:   row.P50Ms,
			P75Ms:   row.P75Ms,
			P95Ms:   row.P95Ms,
		})
	}
	slices.SortFunc(out, func(left, right aianalyticsports.RunTime) int {
		return cmp.Compare(outcomeOrder[left.Outcome], outcomeOrder[right.Outcome])
	})

	return out
}

func buildTrend(rows []aianalyticsports.TrendRow, groupOf keyFunc) []aianalyticsports.TrendPoint {
	out := make([]aianalyticsports.TrendPoint, 0, len(rows))
	for _, row := range rows {
		point := aianalyticsports.TrendPoint{
			SeriesKey:       groupOf(row.DeploymentKey),
			BucketStart:     row.BucketStart,
			Runs:            row.Runs,
			CompletedRate:   rate(row.Completed, row.Runs),
			ToolFailureRate: rate(row.ToolFailures, row.ToolCalls),
		}
		if row.HasMedian {
			point.MedianRunMs = ptr(row.MedianRunMs)
		}
		out = append(out, point)
	}
	slices.SortFunc(out, func(left, right aianalyticsports.TrendPoint) int {
		return cmp.Or(
			cmp.Compare(left.BucketStart, right.BucketStart),
			cmp.Compare(left.SeriesKey, right.SeriesKey),
		)
	})

	return out
}

func buildTools(rows []aianalyticsports.ToolCellRow, groupOf keyFunc) []aianalyticsports.ToolRow {
	type cellKey struct{ tool, series string }
	cells := make(map[cellKey]*aianalyticsports.ToolCell, len(rows))
	tools := make(map[string]*aianalyticsports.ToolRow, len(rows))
	for _, row := range rows {
		tool, ok := tools[row.ToolName]
		if !ok {
			tool = &aianalyticsports.ToolRow{Name: row.ToolName}
			tools[row.ToolName] = tool
		}
		key := cellKey{tool: row.ToolName, series: groupOf(row.DeploymentKey)}
		cell, ok := cells[key]
		if !ok {
			cell = &aianalyticsports.ToolCell{SeriesKey: key.series}
			cells[key] = cell
		}
		cell.Calls += row.Calls
		tool.Calls += row.Calls
		if row.Cause != "" {
			cell.Failed += row.Calls
			tool.Failed += row.Calls
		}
	}

	ordered := make([]*aianalyticsports.ToolRow, 0, len(tools))
	for _, tool := range tools {
		tool.FailureRate = rate(tool.Failed, tool.Calls)
		ordered = append(ordered, tool)
	}
	slices.SortFunc(ordered, compareTools)
	ordered = ordered[:min(len(ordered), maxTools)]

	out := make([]aianalyticsports.ToolRow, 0, len(ordered))
	for _, tool := range ordered {
		row := *tool
		row.Cells = make([]aianalyticsports.ToolCell, 0, 8)
		for key, cell := range cells {
			if key.tool != tool.Name {
				continue
			}
			cell.FailureRate = rate(cell.Failed, cell.Calls)
			cell.Few = !aianalytics.RanksByCalls(cell.Calls)
			row.Cells = append(row.Cells, *cell)
		}
		slices.SortFunc(row.Cells, func(left, right aianalyticsports.ToolCell) int {
			return cmp.Compare(left.SeriesKey, right.SeriesKey)
		})
		out = append(out, row)
	}

	return out
}

func compareTools(left, right *aianalyticsports.ToolRow) int {
	leftRanks, rightRanks := aianalytics.RanksByCalls(
		left.Calls,
	), aianalytics.RanksByCalls(
		right.Calls,
	)
	if leftRanks != rightRanks {
		if leftRanks {
			return -1
		}
		return 1
	}

	return cmp.Or(
		cmp.Compare(valueOr(right.FailureRate), valueOr(left.FailureRate)),
		cmp.Compare(right.Calls, left.Calls),
		cmp.Compare(left.Name, right.Name),
	)
}

func valueOr(value *float64) float64 {
	if value == nil {
		return 0
	}

	return *value
}

func buildProviderHealth(
	data *readings,
	providers providerLookup,
) []aianalyticsports.ProviderHealth {
	health := make(map[string]*aianalyticsports.ProviderHealth, len(data.providerCalls))
	order := make([]string, 0, len(data.providerCalls))
	kinds := make(map[string]map[aianalytics.ProviderError]int, len(data.providerCalls))
	for _, row := range data.providerCalls {
		if row.ProviderID == "" {
			continue
		}
		key := deploymentKeyOf(row.DeploymentKey)
		item, ok := health[key]
		if !ok {
			item = &aianalyticsports.ProviderHealth{
				Key:          key,
				Provider:     providers.ref(row.ProviderID),
				ModelKey:     row.ModelKey,
				Model:        providers.modelName(row.ModelKey).Label,
				ErrorsByKind: []aianalyticsports.ErrorCount{},
			}
			health[key] = item
			order = append(order, key)
			kinds[key] = make(
				map[aianalytics.ProviderError]int,
				len(aianalytics.AllProviderErrors()),
			)
		}
		item.Calls += row.Calls
		item.FellToNext += row.FellToNext
		if row.LastHour > 0 && (item.LastCalledAt == nil || row.LastHour > *item.LastCalledAt) {
			item.LastCalledAt = ptr(row.LastHour)
		}
		if row.Error.BeforeAnswer() {
			item.Errors += row.Calls
			kinds[key][row.Error] += row.Calls
		}
	}

	firstTokens := make(map[string]*aianalytics.Histogram, len(order))
	for _, row := range data.histograms {
		key := deploymentKeyOf(row.DeploymentKey)
		histogram, ok := firstTokens[key]
		if !ok {
			histogram = new(aianalytics.Histogram)
			firstTokens[key] = histogram
		}
		histogram.AddBucket(row.Bucket, row.Count)
	}

	out := make([]aianalyticsports.ProviderHealth, 0, len(order))
	for _, key := range order {
		item := health[key]
		item.ErrorRate = rate(item.Errors, item.Calls)
		for _, kind := range aianalytics.AllProviderErrors() {
			if calls := kinds[key][kind]; calls > 0 {
				item.ErrorsByKind = append(
					item.ErrorsByKind,
					aianalyticsports.ErrorCount{Error: kind, Calls: calls},
				)
			}
		}
		if histogram, ok := firstTokens[key]; ok {
			if median, found := histogram.Percentile(0.5); found {
				item.FirstTokenMedianMs = ptr(median)
			}
		}
		out = append(out, *item)
	}
	slices.SortStableFunc(out, func(left, right aianalyticsports.ProviderHealth) int {
		return cmp.Or(cmp.Compare(right.Calls, left.Calls), cmp.Compare(left.Key, right.Key))
	})

	return out
}
