package aianalytics

const (
	MinRunsToRank       = 30
	MinCallsToRank      = 20
	MinRouteGainPoints  = 5.0
	MinMistakeRatio     = 2.0
	MinGroupsForOutlier = 3
	MinFailedRunsToNote = 10
	MinRunTimeRatio     = 2.0
	MinConnectionGap    = 3.0

	ToolFailureAlarmRate   = 0.03
	ProviderErrorAlarmRate = 0.05
	FellToNextAlarmCalls   = 50
	QuietSeconds           = 24 * 60 * 60

	SuggestionSettleSeconds = 30 * 60
)

func RanksByRuns(runs int) bool {
	return runs >= MinRunsToRank
}

func RanksByCalls(calls int) bool {
	return calls >= MinCallsToRank
}

func ToolFailureAlarm(rate *float64) bool {
	return rate != nil && *rate > ToolFailureAlarmRate
}

func ProviderErrorAlarm(rate *float64) bool {
	return rate != nil && *rate > ProviderErrorAlarmRate
}

func FellToNextAlarm(calls int) bool {
	return calls > FellToNextAlarmCalls
}

func IsQuiet(lastAt *int64, until int64) bool {
	return lastAt != nil && until-*lastAt > QuietSeconds
}

func SuggestionPendingSince(now int64) int64 {
	return now - SuggestionSettleSeconds
}
