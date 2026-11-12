// Package webwatchjobs follows the web watches agents set until they end: a
// sweep each minute claims the watches whose check is due, checks the
// vendor's runs, hands what each watch found to the work waiting on it, and
// removes the vendor's monitor when the watch's time runs out. A watch holds
// no workflow while it waits; its next check is a column.
package webwatchjobs

import "time"

const (
	// checkEvery is how often a watch is checked when nothing asks sooner;
	// the vendor's report of a run, or a stop, makes it due at once.
	checkEvery = 30 * time.Minute
	// retryAfter is how soon a check that failed is tried again.
	retryAfter = 5 * time.Minute
	// sweepEvery is how late a due check may run.
	sweepEvery = time.Minute
	// claimLimit and checkConcurrency bound one sweep: each check is a few
	// calls to the vendor, so a handful run at once and the rest wait their
	// turn at the next minute.
	claimLimit       = 100
	checkConcurrency = 8
	checkTimeout     = 2 * time.Minute

	SweepWorkflowName = "CheckDueWebWatchesWorkflow"
	SweepScheduleID   = "web-watch-checks"
)
