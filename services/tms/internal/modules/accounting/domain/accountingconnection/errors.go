package accountingconnection

import "errors"

var (
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
)
