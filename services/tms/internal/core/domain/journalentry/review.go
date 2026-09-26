package journalentry

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (e *JournalEntry) CanApprove() error {
	switch e.Status {
	case StatusPending:
		if e.IsApproved {
			return errortypes.NewBusinessError(
				"Journal entry " + e.EntryNumber + " is already approved",
			)
		}
		return nil
	case StatusApproved, StatusPosted:
		return errortypes.NewBusinessError(
			"Journal entry " + e.EntryNumber + " is already approved",
		)
	case StatusDraft, StatusReversed, StatusRejected, StatusVoid:
		return errortypes.NewBusinessError(
			"Journal entry " + e.EntryNumber + " is " + string(
				e.Status,
			) + " and cannot be approved",
		)
	default:
		return errortypes.NewBusinessError(
			"Journal entry {0} is in an unknown state",
			e.EntryNumber,
		)
	}
}

func (e *JournalEntry) CanPost() error {
	switch e.Status {
	case StatusApproved:
		if e.IsPosted {
			return errortypes.NewBusinessError(
				"Journal entry " + e.EntryNumber + " is already posted",
			)
		}
		return e.balanceError()
	case StatusPending:
		return errortypes.NewBusinessError(
			"Journal entry {0} needs approval before it can be posted",
			e.EntryNumber,
		)
	case StatusPosted:
		return errortypes.NewBusinessError("Journal entry {0} is already posted", e.EntryNumber)
	case StatusDraft, StatusReversed, StatusRejected, StatusVoid:
		return errortypes.NewBusinessError(
			"Journal entry {0} is {1} and cannot be posted",
			e.EntryNumber,
			string(e.Status),
		)
	default:
		return errortypes.NewBusinessError(
			"Journal entry {0} is in an unknown state",
			e.EntryNumber,
		)
	}
}

func (e *JournalEntry) balanceError() error {
	var debit, credit int64
	for _, line := range e.Lines {
		if line == nil {
			continue
		}
		debit += line.DebitAmount
		credit += line.CreditAmount
	}
	if len(e.Lines) < 2 || debit != credit || debit != e.TotalDebit || credit != e.TotalCredit {
		return errortypes.NewBusinessError(
			"Journal entry {0} does not balance and cannot be posted",
			e.EntryNumber,
		)
	}
	return nil
}
