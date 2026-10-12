package common

import (
	"errors"
)

// ErrApprovalNeedsAPerson is an approve_detention or waive_detention call that
// did not come from a proposal a person approved. The tool's tier ceiling already keeps the
// runtime from running it on its own; this is the same rule where the money
// moves.
var ErrApprovalNeedsAPerson = errors.New(
	"a detention charge is approved or waived only once a person approves the proposal",
)
