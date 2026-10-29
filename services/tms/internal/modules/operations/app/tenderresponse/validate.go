package tenderresponse

import (
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// validateCounter checks a counter-offer's own shape. Whether the offer can
// still take one is planResponse's call.
func validateCounter(req *tenderports.TenderResponseRequest) error {
	multiErr := errortypes.NewMultiError()
	switch {
	case !req.CounterRate.IsPositive():
		multiErr.Add(
			"counterRate",
			errortypes.ErrRequired,
			"A counter-offer needs a rate above zero",
		)
	case req.CounterRate.GreaterThanOrEqual(tender.MaxCounterRate):
		multiErr.Add("counterRate", errortypes.ErrInvalid, "Counter-offer rate is too large")
	}
	if utf8.RuneCountInString(req.CounterNote) > tender.MaxCounterNoteLength {
		multiErr.Add("counterNote", errortypes.ErrInvalid, "Note must be at most 500 characters")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
