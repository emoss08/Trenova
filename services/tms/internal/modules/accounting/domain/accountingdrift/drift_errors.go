package accountingdrift

import (
	"errors"
)

var (
	ErrDriftClosed         = errors.New("the finding was already resolved or dismissed")
	ErrDriftNoteRequired   = errors.New("a note is required to dismiss a finding")
	ErrDriftFixUnavailable = errors.New("that fix is not offered for this finding")
	ErrDriftExplainOnly    = errors.New(
		"a trial balance difference is explained, not fixed: find the entry made in the " +
			"accounting system or the record Trenova could not send, then dismiss it",
	)
)
