package accountingsync

import "errors"

var (
	ErrDriftClosed         = errors.New("the finding was already resolved or dismissed")
	ErrDriftNoteRequired   = errors.New("a note is required to dismiss a finding")
	ErrDriftFixUnavailable = errors.New("that fix is not offered for this finding")
	ErrDriftExplainOnly    = errors.New(
		"a trial balance difference is explained, not fixed: find the entry made in the " +
			"accounting system or the record Trenova could not send, then dismiss it",
	)
	ErrInboundChangeClosed = errors.New(
		"the change was already applied, ignored or superseded",
	)
	ErrInboundChangeNotProposed = errors.New("only a proposed change can be applied")
	ErrInboundNoteRequired      = errors.New("a note is required to ignore a change")
	ErrInboundNotApplicable     = errors.New(
		"this change cannot be applied until what it pays matches Trenova",
	)
	ErrModeFixed = errors.New(
		"what is sent is fixed for this connection once sending starts, because the books already hold what was sent",
	)
	ErrStartDateFixed = errors.New(
		"the start date is fixed once opening balances are sent, because they hold every balance up to it",
	)
	ErrModeNotRecognized   = errors.New("the mode is not recognized")
	ErrModeUnavailable     = errors.New("this accounting system cannot receive journal entries")
	ErrGranularityRequired = errors.New(
		"journal entries are sent detailed or as a daily summary",
	)
	ErrOpeningBalancesNeedLedger = errors.New(
		"opening balances are sent only when journal entries are",
	)
	ErrMappingNotProposed = errors.New("only a proposed mapping can be rejected")
	ErrMappingNotSet      = errors.New("the mapping has no accounting record to clear")
)
