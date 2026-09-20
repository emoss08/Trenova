package agentquerytoolservice

import "strconv"

/*
An absent date has to say what its absence means.

The compliance expiry fields were `int64` with `omitempty`, so a driver with no
medical certificate on file produced no key at all. Nothing in the result said
the certificate was missing, and a reader with no key to look at concludes
there is nothing to worry about. That is the wrong conclusion and the dangerous
one: a driver with no medical card recorded cannot be dispatched, while a
driver whose card expires in three weeks still can.

In the incident this comes from, three of eight drivers had no medical card
date. The agent listed them under "drivers with no medical card expiry" and
moved on to ask about hazmat.

So the field is always present, and when it is unset it carries a phrase saying
what unset means here — which differs by field. No date on a credential means
nobody recorded one. No date on a delivery means it has not happened yet. Those
are not the same fact and must not render the same way.
*/
type optionalDate struct {
	seconds int64
	// absent is the phrase for an unset value, in the words of this field.
	// Empty means the zero value reached the encoder without going through a
	// constructor, which happens wherever a projection skips a row's optional
	// relation — applyProfile returns early for a worker with no profile — so
	// it falls back to the safe phrase rather than emitting an empty string.
	absent string
}

// defaultAbsent is what an unset date says when nothing more specific was
// chosen. It has to be the cautious reading: "no value here" must never render
// as something a reader can mistake for "fine".
const defaultAbsent = "none on file"

// recordedDate is for a value somebody was supposed to enter. Its absence is a
// gap in the record, not a state of the world.
func recordedDate(seconds int64) optionalDate {
	return optionalDate{seconds: seconds, absent: defaultAbsent}
}

// pointerDate is recordedDate for the columns the domain models as nullable.
func pointerDate(seconds *int64) optionalDate {
	if seconds == nil {
		return recordedDate(0)
	}

	return recordedDate(*seconds)
}

// expectedDate is for something that has not happened yet rather than something
// nobody wrote down.
func expectedDate(seconds int64, absent string) optionalDate {
	return optionalDate{seconds: seconds, absent: absent}
}

// MarshalJSON emits the instant as a bare number so the runtime's date pass
// renders it, or the phrase when there is nothing to render. A number reaching
// the model unrendered would be the bug this whole path exists to stop, so the
// two cases are deliberately the only two.
func (d optionalDate) MarshalJSON() ([]byte, error) {
	if d.seconds <= 0 {
		absent := d.absent
		if absent == "" {
			absent = defaultAbsent
		}

		return strconv.AppendQuote(nil, absent), nil
	}

	return strconv.AppendInt(nil, d.seconds, 10), nil
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}

	return *value
}
