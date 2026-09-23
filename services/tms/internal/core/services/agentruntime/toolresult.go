package agentruntime

import (
	"math"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/shared/timeutils"
)

// Epoch seconds a tool could plausibly be reporting as a date. The bound is what
// separates a timestamp from a large quantity that happens to share a column
// name: an invoice total in cents can land in this range, a due date cannot land
// outside it.
const (
	earliestPlausibleInstant int64 = 946684800  // 2000-01-01
	latestPlausibleInstant   int64 = 4102444800 // 2100-01-01
)

// dateKeySuffixes are the field-name endings that mean an instant. Matching on
// the name as well as the range is belt and braces: either test alone has a
// false positive the other rules out.
//
// "on" is deliberately absent — "reason" ends in it.
var dateKeySuffixes = []string{"at", "date", "expiry", "expires"}

// instantKeySuffixes are the field-name endings of an instant whose hour
// matters. A delivery window read as a date alone told the model the stop had
// no window at all, and it said so.
var instantKeySuffixes = []string{"time", "windowstart", "windowend", "arrival", "departure", "eta"}

// Keys that say nothing to the model about the record in front of it. The
// tenant is the one the conversation is in, and a nested record's version and
// audit stamps repeat on every commodity, stop and charge of a shipment; the
// get_shipment the Dispatch desk read carried twenty of each.
//
// A record's own version stays at the top level, where a write that checks it
// reads it from.
var (
	tenantKeys      = []string{"organizationId", "businessUnitId"}
	nestedNoiseKeys = []string{"version", "createdAt", "updatedAt"}
)

// humanizeDates rewrites every epoch integer a tool returned into a date a
// reader can act on, before the result reaches the model.
//
// This is the whole fix for a wrong compliance answer we shipped: tools emitted
// medicalCardExpiry as 1791591001, and the model — having no arithmetic —
// guessed its way to the wrong month and reported the opposite of the truth.
// Rendering the date costs nothing and removes the guess entirely.
//
// The epoch is replaced rather than supplemented. Nothing downstream wants the
// integer: the model cannot compute with it, and the date filters already
// accept the YYYY-MM-DD form this produces, so a value read out of one result
// can be passed straight back into the next call.
func humanizeDates(value any, now int64, timezone string) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if replacement, ok := describeInstant(key, nested, now, timezone); ok {
				typed[key] = replacement
				continue
			}
			typed[key] = humanizeDates(nested, now, timezone)
		}

		return typed
	case []any:
		for i, nested := range typed {
			typed[i] = humanizeDates(nested, now, timezone)
		}

		return typed
	default:
		return value
	}
}

// trimRecord takes out what a record carries for the database rather than for
// the reader: the tenant everywhere, and the version and audit stamps of every
// record nested inside another. depth is zero for the result itself and for
// each row of a top-level list.
func trimRecord(value any, depth int) any {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range tenantKeys {
			delete(typed, key)
		}
		if depth > 0 {
			for _, key := range nestedNoiseKeys {
				delete(typed, key)
			}
		}
		for key, nested := range typed {
			typed[key] = trimRecord(nested, depth+1)
		}

		return typed
	case []any:
		for i, nested := range typed {
			typed[i] = trimRecord(nested, depth)
		}

		return typed
	default:
		return value
	}
}

// describeInstant renders one instant as the date it falls on in the
// organization's zone. The zone matters at the edges, which are exactly where
// compliance questions live: a card expiring at 00:30 local time is tomorrow's
// problem to the person asking, whatever a UTC clock says.
func describeInstant(key string, value any, now int64, timezone string) (string, bool) {
	withClock := hasKeySuffix(key, instantKeySuffixes)
	if !withClock && !hasKeySuffix(key, dateKeySuffixes) {
		return "", false
	}

	seconds, ok := wholeNumber(value)
	if !ok {
		return "", false
	}

	// A count that shares a date-like name — daysUntilExpiry is the one we
	// already ship — is never in epoch range, so the range check is what keeps
	// this from turning 21 into 1970.
	if seconds < earliestPlausibleInstant || seconds > latestPlausibleInstant {
		return "", false
	}

	if withClock {
		return timeutils.DescribeUnixInstantIn(seconds, now, timezone), true
	}

	return timeutils.DescribeUnixDateIn(seconds, now, timezone), true
}

func hasKeySuffix(key string, suffixes []string) bool {
	lowered := strings.ToLower(key)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lowered, suffix) {
			return true
		}
	}

	return false
}

// wholeNumber reads the number shapes a decoded JSON document can hold. A
// fractional value is not an instant any tool here produces, and coercing one
// would invent a precision the source never had.
func wholeNumber(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != math.Trunc(typed) {
			return 0, false
		}

		return int64(typed), true
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	default:
		return 0, false
	}
}

// encodeToolResult renders a tool's return value for the model.
//
// The round trip through a generic document is what lets one rule reach every
// tool: the tools return their own typed rows, and whole domain entities in the
// case of the get_ family, so there is no single struct to annotate. Doing it
// here means a tool added later inherits readable dates without knowing this
// exists, which is the only version of this that stays true.
//
// The round trip has one cost that has to be paid back here. Going through a
// map loses the struct's field order, and sonic's default marshaller emits map
// keys in whatever order it finds them — so the same rows re-serialised on a
// later turn came out as a different byte string. That makes a tool result
// unreadable to a person comparing two turns, and it defeats prompt caching
// outright, because a cache is keyed on a prefix that is now different for no
// reason. ConfigStd sorts the keys, which is what the repeat guard and the
// transcript exporter already do for the same reason.
func encodeToolResult(data any, now int64, timezone string) (string, error) {
	encoded, err := sonic.Marshal(data)
	if err != nil {
		return "", err
	}

	var document any
	if err = sonic.Unmarshal(encoded, &document); err != nil {
		return "", err
	}

	humanized, err := sonic.ConfigStd.Marshal(humanizeDates(trimRecord(document, 0), now, timezone))
	if err != nil {
		return "", err
	}

	return string(humanized), nil
}
