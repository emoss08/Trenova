// Package webwatchjobs follows a web watch an agent set until it ends: it
// checks the vendor's runs on a schedule, or at once when the vendor reports
// one, hands what the watch found to the work waiting on it, and removes the
// vendor's monitor when the watch's time runs out.
package webwatchjobs

import "time"

const (
	checkInterval = 30 * time.Minute
	checkTimeout  = 2 * time.Minute

	// continueAfter bounds one execution's history: a watch can last ninety
	// days, so the workflow starts afresh with the same watch now and then.
	continueAfter = 200
)
