package fiscalclose

import "github.com/emoss08/trenova/pkg/errortypes"

type Check string

const (
	CheckPeriodStatus     = Check("periodStatus")
	CheckEarlierPeriods   = Check("earlierPeriods")
	CheckCloseControl     = Check("closeControl")
	CheckManualJournals   = Check("manualJournals")
	CheckJournalReversals = Check("journalReversals")
	CheckUnpostedSources  = Check("unpostedSources")
	CheckReconciliation   = Check("reconciliation")
)

func (c Check) Counted() bool {
	switch c {
	case CheckManualJournals, CheckJournalReversals, CheckUnpostedSources, CheckReconciliation:
		return true
	case CheckPeriodStatus, CheckEarlierPeriods, CheckCloseControl:
		return false
	default:
		return false
	}
}

type Blocker struct {
	Field    string               `json:"field"`
	Code     errortypes.ErrorCode `json:"code"`
	Message  string               `json:"message"`
	Category string               `json:"category"`
	Check    Check                `json:"check,omitempty"`
	Count    *int                 `json:"count,omitempty"`
}

type Result struct {
	CanClose bool       `json:"canClose"`
	Blockers []*Blocker `json:"blockers"`
	Cleared  []Check    `json:"cleared,omitempty"`
}
