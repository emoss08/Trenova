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
)

func RanksByRuns(runs int) bool {
	return runs >= MinRunsToRank
}

func RanksByCalls(calls int) bool {
	return calls >= MinCallsToRank
}
