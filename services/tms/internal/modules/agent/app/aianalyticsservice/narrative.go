package aianalyticsservice

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/shared/i18n"
)

const (
	TimeoutShareOfFailedRun  = 0.8
	toneDanger               = "danger"
	toneWarn                 = "warn"
	unhealthyFailureRate     = 0.5
	recoveredFailureRate     = 0.2
	minCallsForHealthChange  = 3
	maxEvents                = 8
	eventDedupeWindowSeconds = 6 * 60 * 60
	percentScale             = 100
)

const (
	tagSeries   = "s"
	tagProvider = "p"
	tagOther    = "q"
	tagStrong   = "b"
)

var narrativeTags = []string{tagSeries, tagProvider, tagOther, tagStrong}

type narration struct {
	locale      i18n.Locale
	window      aianalytics.Window
	compareBy   aianalytics.CompareBy
	series      []aianalyticsports.Series
	deployments []aianalyticsports.Series
	providers   providerLookup
}

type sentenceParts map[string]aianalyticsports.Segment

func compose(pieces []i18n.RichPiece, parts sentenceParts) []aianalyticsports.Segment {
	out := make([]aianalyticsports.Segment, 0, len(pieces))
	for _, piece := range pieces {
		segment, ok := parts[piece.Tag]
		if piece.Tag == "" || !ok {
			segment = aianalyticsports.Segment{}
		}
		segment.Text = piece.Text
		if segment.Text != "" {
			out = append(out, segment)
		}
	}

	return out
}

func joinSentences(sentences ...[]aianalyticsports.Segment) []aianalyticsports.Segment {
	out := make([]aianalyticsports.Segment, 0, 16)
	for _, sentence := range sentences {
		if len(sentence) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, aianalyticsports.Segment{Text: " "})
		}
		out = append(out, sentence...)
	}

	return out
}

func strongPart() aianalyticsports.Segment {
	return aianalyticsports.Segment{Strong: true}
}

func seriesPart(series *aianalyticsports.Series, tone string) aianalyticsports.Segment {
	return aianalyticsports.Segment{
		Target:    aianalyticsports.SegmentTargetSeries,
		SeriesKey: series.Key,
		Tone:      tone,
	}
}

func providerPart(ref aianalyticsports.ProviderRef, tone string) aianalyticsports.Segment {
	if ref.ID.IsNil() || ref.Removed {
		return strongPart()
	}

	return aianalyticsports.Segment{
		Target:     aianalyticsports.SegmentTargetProvider,
		ProviderID: ref.ID,
		Tone:       tone,
	}
}

func percent(value float64) string {
	return strconv.Itoa(int(math.Round(value*percentScale))) + "%"
}

func times(ratio float64) string {
	return fmt.Sprintf("%.1f×", ratio)
}

func windowPhrase(locale i18n.Locale, window aianalytics.Window) string {
	switch window {
	case aianalytics.WindowDay:
		return i18n.Translate(locale, "Over the last 24 hours")
	case aianalytics.WindowWeek:
		return i18n.Translate(locale, "Over the last 7 days")
	default:
		return i18n.Translate(locale, "Over the last 30 days")
	}
}

func summarize(n *narration) []aianalyticsports.Segment {
	period := windowPhrase(n.locale, n.window)
	if len(n.series) == 0 {
		return []aianalyticsports.Segment{{Text: i18n.Translate(
			n.locale,
			"{0}, no agent run, conversation or AI call finished, so there is nothing to compare yet.",
			period,
		)}}
	}

	return joinSentences(leaderSentence(n, period), connectionGap(n), mistakeOutlier(n))
}

func leaderSentence(n *narration, period string) []aianalyticsports.Segment {
	leader := leadingSeries(n.series)
	if leader == nil {
		top := &n.series[0]
		parts := sentenceParts{tagSeries: seriesPart(top, "")}
		if n.compareBy == aianalytics.CompareByProvider {
			return compose(i18n.TranslateRich(
				n.locale,
				"{0}, <s>{1}</s> did the most work, but nothing has run often enough yet to say which provider finishes more of its runs.",
				narrativeTags,
				period,
				top.Label,
			), parts)
		}

		return compose(i18n.TranslateRich(
			n.locale,
			"{0}, <s>{1}</s> did the most work, but nothing has run often enough yet to say which model finishes more of its runs.",
			narrativeTags,
			period,
			top.Label,
		), parts)
	}

	parts := sentenceParts{tagSeries: seriesPart(leader, ""), tagStrong: strongPart()}
	share := percent(*leader.CompletedRate)
	if countComparable(n.series) == 1 {
		return compose(i18n.TranslateRich(
			n.locale,
			"{0}, <s>{1}</s> completed <b>{2}</b> of its runs.",
			narrativeTags,
			period,
			leader.Label,
			share,
		), parts)
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"{0}, <s>{1}</s> completed the most of its runs — <b>{2}</b>.",
		narrativeTags,
		period,
		leader.Label,
		share,
	), parts)
}

func countComparable(series []aianalyticsports.Series) int {
	count := 0
	for idx := range series {
		if !series[idx].Few {
			count++
		}
	}

	return count
}

func leadingSeries(series []aianalyticsports.Series) *aianalyticsports.Series {
	var leader *aianalyticsports.Series
	for idx := range series {
		item := &series[idx]
		if item.Few || item.CompletedRate == nil {
			continue
		}
		if leader == nil || *item.CompletedRate > *leader.CompletedRate {
			leader = item
		}
	}

	return leader
}

func completion(series *aianalyticsports.Series) float64 {
	if series.CompletedRate == nil {
		return 0
	}

	return *series.CompletedRate
}

type sharedModel struct {
	model  string
	better *aianalyticsports.Series
	worse  *aianalyticsports.Series
	points float64
}

func sharedModelPair(deployments []aianalyticsports.Series) *sharedModel {
	byModel := make(map[string][]*aianalyticsports.Series, len(deployments))
	for idx := range deployments {
		item := &deployments[idx]
		if item.ModelKey == "" || item.ProviderID.IsNil() || item.Few {
			continue
		}
		byModel[item.ModelKey] = append(byModel[item.ModelKey], item)
	}

	var widest *sharedModel
	for model, rows := range byModel {
		if len(rows) < 2 {
			continue
		}
		slices.SortStableFunc(rows, func(left, right *aianalyticsports.Series) int {
			return cmp.Or(
				cmp.Compare(completion(right), completion(left)),
				cmp.Compare(right.Runs, left.Runs),
				cmp.Compare(left.Key, right.Key),
			)
		})
		points := (completion(rows[0]) - completion(rows[len(rows)-1])) * percentScale
		if widest == nil || points > widest.points ||
			(points == widest.points && model < widest.model) {
			widest = &sharedModel{
				model:  model,
				better: rows[0],
				worse:  rows[len(rows)-1],
				points: points,
			}
		}
	}

	return widest
}

func connectionGap(n *narration) []aianalyticsports.Segment {
	widest := sharedModelPair(n.deployments)
	if widest == nil || widest.points < aianalytics.MinConnectionGap {
		return nil
	}
	better := n.providers.ref(widest.better.ProviderID.String())
	worse := n.providers.ref(widest.worse.ProviderID.String())

	return compose(i18n.TranslateRich(
		n.locale,
		"<b>{0}</b> completed {1} on <p>{2}</p> but {3} on <q>{4}</q>. It's the same model, so the gap comes from the connection.",
		narrativeTags,
		n.providers.modelName(widest.model).Label,
		percent(completion(widest.better)),
		better.Name,
		percent(completion(widest.worse)),
		worse.Name,
	), sentenceParts{
		tagStrong:   strongPart(),
		tagProvider: providerPart(better, ""),
		tagOther:    providerPart(worse, toneDanger),
	})
}

func toolFailureShare(series *aianalyticsports.Series) float64 {
	if series.ToolCalls == 0 {
		return 0
	}

	return float64(series.ToolCallsFailed) / float64(series.ToolCalls)
}

func mistakeOutlier(n *narration) []aianalyticsports.Segment {
	ranked := make([]*aianalyticsports.Series, 0, len(n.series))
	for idx := range n.series {
		if !n.series[idx].Few {
			ranked = append(ranked, &n.series[idx])
		}
	}
	if len(ranked) < aianalytics.MinGroupsForOutlier {
		return nil
	}

	var worst *aianalyticsports.Series
	for _, item := range ranked {
		if item.ToolCalls == 0 {
			continue
		}
		if worst == nil || toolFailureShare(item) > toolFailureShare(worst) {
			worst = item
		}
	}
	if worst == nil || worst.ToolCallsFailed == 0 {
		return nil
	}

	restCalls, restFailed := 0, 0
	for _, item := range ranked {
		if item == worst {
			continue
		}
		restCalls += item.ToolCalls
		restFailed += item.ToolCallsFailed
	}
	if restCalls == 0 || restFailed == 0 {
		return nil
	}
	ratio := toolFailureShare(worst) / (float64(restFailed) / float64(restCalls))
	if ratio < aianalytics.MinMistakeRatio {
		return nil
	}

	parts := sentenceParts{tagSeries: seriesPart(worst, ""), tagStrong: strongPart()}
	if cause := causePhrase(n.locale, topCause(worst.ToolFailuresByCause)); cause != "" {
		return compose(i18n.TranslateRich(
			n.locale,
			"<s>{0}</s> gets tool calls wrong <b>{1}</b> as often as the rest, mostly {2}.",
			narrativeTags,
			worst.Label,
			times(ratio),
			cause,
		), parts)
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"<s>{0}</s> gets tool calls wrong <b>{1}</b> as often as the rest.",
		narrativeTags,
		worst.Label,
		times(ratio),
	), parts)
}

func causePhrase(locale i18n.Locale, cause aianalytics.Cause) string {
	switch cause {
	case aianalytics.CauseWrongArguments:
		return i18n.Translate(locale, "wrong arguments")
	case aianalytics.CauseToolTimedOut:
		return i18n.Translate(locale, "tools timing out")
	case aianalytics.CauseRepeatedCall:
		return i18n.Translate(locale, "repeating the same call")
	case aianalytics.CauseBlockedByRule:
		return i18n.Translate(locale, "calls a rule blocked")
	case aianalytics.CauseToolError:
		return i18n.Translate(locale, "tools returning errors")
	case aianalytics.CauseToolNotHeld:
		return i18n.Translate(locale, "calling tools it doesn't hold")
	default:
		return ""
	}
}

func topCause(causes []aianalyticsports.CauseCount) aianalytics.Cause {
	var top aianalyticsports.CauseCount
	for _, item := range causes {
		if item.Calls > top.Calls {
			top = item
		}
	}

	return top.Cause
}

func runTimeOf(
	series *aianalyticsports.Series,
	outcome aianalytics.Outcome,
) *aianalyticsports.RunTime {
	for idx := range series.RunTimes {
		if series.RunTimes[idx].Outcome == outcome {
			return &series.RunTimes[idx]
		}
	}

	return nil
}

func runTimeNote(
	deployments []aianalyticsports.Series,
	providers providerLookup,
) []aianalyticsports.Segment {
	var slowest *aianalyticsports.Series
	var slowestFailed *aianalyticsports.RunTime
	var ratio float64
	for idx := range deployments {
		item := &deployments[idx]
		completed := runTimeOf(item, aianalytics.OutcomeCompleted)
		failed := runTimeOf(item, aianalytics.OutcomeFailed)
		if completed == nil || failed == nil || item.ProviderID.IsNil() ||
			!aianalytics.RanksByRuns(
				completed.Runs,
			) || failed.Runs < aianalytics.MinFailedRunsToNote ||
			completed.P50Ms <= 0 {
			continue
		}
		candidate := float64(failed.P50Ms) / float64(completed.P50Ms)
		if candidate >= aianalytics.MinRunTimeRatio &&
			(candidate > ratio || (candidate == ratio && item.Key < slowest.Key)) {
			slowest, slowestFailed, ratio = item, failed, candidate
		}
	}
	if slowest == nil {
		return nil
	}

	ref := providers.ref(slowest.ProviderID.String())
	note := compose(i18n.TranslateRich(
		providers.locale,
		"Failed runs on <p>{0}</p> take <b>{1}</b> as long as its completed ones.",
		narrativeTags,
		ref.Name,
		times(ratio),
	), sentenceParts{tagProvider: providerPart(ref, toneWarn), tagStrong: strongPart()})

	provider, ok := providers.get(slowest.ProviderID)
	if !ok || provider.TimeoutSeconds <= 0 {
		return note
	}
	timeoutMs := int64(provider.TimeoutSeconds) * 1000
	if float64(slowestFailed.P50Ms) < TimeoutShareOfFailedRun*float64(timeoutMs) {
		return note
	}

	return joinSentences(note, []aianalyticsports.Segment{{Text: i18n.Translate(
		providers.locale,
		"Most of them wait out the {0}-second timeout first.",
		provider.TimeoutSeconds,
	)}})
}

type providerVersionState struct {
	enabled bool
	model   string
}

func buildEvents(
	data *readings,
	providers providerLookup,
	since int64,
) []aianalyticsports.Event {
	events := make([]aianalyticsports.Event, 0, maxEvents*2)
	events = append(events, settingEvents(data.versions, providers, since)...)
	events = append(events, healthEvents(data.providerHours, providers)...)
	slices.SortStableFunc(events, func(left, right aianalyticsports.Event) int {
		return cmp.Compare(left.At, right.At)
	})

	kept := make([]aianalyticsports.Event, 0, len(events))
	lastAt := make(map[string]int64, len(events))
	for _, event := range events {
		key := event.Provider.ID.String() + "|" + string(event.Kind)
		if previous, ok := lastAt[key]; ok && event.At-previous < eventDedupeWindowSeconds {
			continue
		}
		lastAt[key] = event.At
		kept = append(kept, event)
	}
	if len(kept) > maxEvents {
		kept = kept[len(kept)-maxEvents:]
	}

	return kept
}

func settingEvents(
	rows []aianalyticsports.ProviderVersionRow,
	providers providerLookup,
	since int64,
) []aianalyticsports.Event {
	out := make([]aianalyticsports.Event, 0, len(rows))
	previous := make(map[string]providerVersionState, len(rows))
	for _, row := range rows {
		before, seen := previous[row.ProviderID]
		previous[row.ProviderID] = providerVersionState{enabled: row.Enabled, model: row.Model}
		if !seen || row.CreatedAt < since {
			continue
		}
		ref := providers.ref(row.ProviderID)
		if ref.Removed && row.Name != "" {
			ref.Name, ref.ShortName = row.Name, row.Name
		}
		switch {
		case before.enabled && !row.Enabled:
			out = append(out, aianalyticsports.Event{
				At: row.CreatedAt, Provider: ref, Kind: aianalytics.EventTurnedOff,
				Label: i18n.Translate(providers.locale, "{0} turned off", ref.ShortName),
			})
		case !before.enabled && row.Enabled:
			out = append(out, aianalyticsports.Event{
				At: row.CreatedAt, Provider: ref, Kind: aianalytics.EventTurnedOn,
				Label: i18n.Translate(providers.locale, "{0} turned on", ref.ShortName),
			})
		}
		if before.model != row.Model && row.Model != "" {
			out = append(out, aianalyticsports.Event{
				At: row.CreatedAt, Provider: ref, Kind: aianalytics.EventModelChanged,
				Label: i18n.Translate(
					providers.locale,
					"{0} switched to {1}",
					ref.ShortName,
					aianalytics.NameModel(aianalytics.CanonicalModel(row.Model)).Label,
				),
			})
		}
	}

	return out
}

func healthEvents(
	rows []aianalyticsports.ProviderHourRow,
	providers providerLookup,
) []aianalyticsports.Event {
	out := make([]aianalyticsports.Event, 0, 8)
	unhealthy := make(map[string]bool, 8)
	for _, row := range rows {
		if row.Calls < minCallsForHealthChange && row.Failed < row.Calls {
			continue
		}
		failureRate := float64(row.Failed) / float64(row.Calls)
		ref := providers.ref(row.ProviderID)
		switch {
		case !unhealthy[row.ProviderID] && row.Calls >= minCallsForHealthChange &&
			failureRate >= unhealthyFailureRate:
			unhealthy[row.ProviderID] = true
			event := aianalyticsports.Event{
				At: row.Hour, Provider: ref, Kind: aianalytics.EventStoppedAnswering,
				Label: i18n.Translate(providers.locale, "{0} stopped answering", ref.ShortName),
			}
			if row.AuthFailed*2 >= row.Failed {
				event.Kind = aianalytics.EventLostAccess
				event.Label = i18n.Translate(providers.locale, "{0} lost access", ref.ShortName)
			}
			out = append(out, event)
		case unhealthy[row.ProviderID] && failureRate < recoveredFailureRate:
			unhealthy[row.ProviderID] = false
			out = append(out, aianalyticsports.Event{
				At: row.Hour, Provider: ref, Kind: aianalytics.EventRecovered,
				Label: i18n.Translate(providers.locale, "{0} recovered", ref.ShortName),
			})
		}
	}

	return out
}
