package fixtures

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rotisserie/eris"
)

// FixtureHelpers provides a set of utility functions to facilitate generating
// template-based fixture data, particularly for testing and development. It
// allows manipulation of times, generation of unique identifiers, and various
// array and collection operations.
type FixtureHelpers struct {
	// baseTime holds a reference time that will be used as the basis for all
	// time-related helper functions. This makes it easier to consistently
	// generate timestamps that are relative to a fixed point in time.
	baseTime time.Time

	// mu protects concurrent access to shared resources
	mu sync.Mutex
}

// NewFixtureHelpers creates and returns a new instance of FixtureHelpers. It
// initializes the base time to the current time and sets up a random number
// generator seeded with the current Unix time.
func NewFixtureHelpers() *FixtureHelpers {
	return &FixtureHelpers{
		baseTime: time.Now(),
	}
}

// GetTemplateFuncs returns a map of template functions that can be used within
// templates. The returned map includes time-based helpers, unique ID generation,
// reference building, enumeration, and various collection utilities.
func (h *FixtureHelpers) GetTemplateFuncs() map[string]any {
	return map[string]any{
		// Time-related helpers
		"now":           h.Now,
		"yearsAgo":      h.YearsAgo,
		"yearsFromNow":  h.YearsFromNow,
		"monthsAgo":     h.MonthsAgo,
		"monthsFromNow": h.MonthsFromNow,
		"daysAgo":       h.DaysAgo,
		"daysFromNow":   h.DaysFromNow,
		"dateString":    h.DateString,
		"timestamp":     h.Timestamp,

		// ID and reference helpers
		"ulid":        h.ULID,
		"sequence":    h.Sequence,
		"reference":   h.Reference,
		"randomEnum":  h.RandomEnum,
		"randomPhone": h.RandomPhone,

		// Array and collection helpers
		"join":    strings.Join,
		"split":   strings.Split,
		"concat":  h.Concat,
		"slice":   h.Slice,
		"repeat":  h.Repeat,
		"random":  h.Random,
		"sample":  h.Sample,
		"shuffle": h.Shuffle,
	}
}

// secureRandomInt generates a cryptographically secure random number in the range [min, max]
func secureRandomInt(minX, maxX int64) (int64, error) {
	if minX > maxX {
		return 0, eris.New("min cannot be greater than max")
	}

	n := maxX - minX + 1
	bigN := big.NewInt(n)

	randNum, err := rand.Int(rand.Reader, bigN)
	if err != nil {
		return 0, eris.Wrap(err, "failed to generate random number")
	}

	return randNum.Int64() + minX, nil
}

// Sequences is a global map that keeps track of named counters used by the
// Sequence function. Each named sequence will increment independently.
var (
	sequences   = make(map[string]int64)
	sequencesMu sync.Mutex
)

// Now returns the base time stored in the FixtureHelpers instance. This is
// useful for generating time-dependent fixtures that all share the same
// reference time.
func (h *FixtureHelpers) Now() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.baseTime
}

// YearsAgo returns a Unix timestamp representing the time N years before the
// base time. This can be useful when generating historical dates.

func (h *FixtureHelpers) YearsAgo(years int64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.baseTime.AddDate(-int(years), 0, 0).Unix()
}

// YearsFromNow returns a Unix timestamp representing the time N years after the
// base time. This can be useful when generating future dates.
func (h *FixtureHelpers) YearsFromNow(years int64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.baseTime.AddDate(int(years), 0, 0).Unix()
}

// MonthsAgo returns a Unix timestamp for N months prior to the base time.
func (h *FixtureHelpers) MonthsAgo(months int64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.baseTime.AddDate(0, -int(months), 0).Unix()
}

// MonthsFromNow returns a Unix timestamp for N months after the base time.
func (h *FixtureHelpers) MonthsFromNow(months int64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.baseTime.AddDate(0, int(months), 0).Unix()
}

// DaysAgo returns a Unix timestamp for N days before the base time.
func (h *FixtureHelpers) DaysAgo(days int64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.baseTime.AddDate(0, 0, -int(days)).Unix()
}

// DaysFromNow returns a Unix timestamp for N days after the base time.
func (h *FixtureHelpers) DaysFromNow(days int64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.baseTime.AddDate(0, 0, int(days)).Unix()
}

// DateString formats a given Unix timestamp into a string according to the
// specified layout. The layout should be a standard Go time layout string (e.g.,
// "2006-01-02").
func (h *FixtureHelpers) DateString(format string, timestamp int64) string {
	return time.Unix(timestamp, 0).Format(format)
}

// Timestamp returns the Unix timestamp corresponding to the base time.
func (h *FixtureHelpers) Timestamp() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.baseTime.Unix()
}

// PULID generates and returns a new PULID as a string. This can be used
// for generating unique identifiers in fixtures.
func (h *FixtureHelpers) ULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0)).String()
}

// Sequence returns an incrementing integer each time it is called with the same
// name. This can be used to generate sequential values in templates, such as
// incrementing primary keys or version numbers.
func (h *FixtureHelpers) Sequence(name string) int64 {
	sequencesMu.Lock()
	defer sequencesMu.Unlock()
	sequences[name]++
	return sequences[name]
}

// Reference returns a Go template string that references a field within a given
// model in the template's data. This can be used within fixtures to dynamically
// link related data.
func (h *FixtureHelpers) Reference(model, field string) string {
	return fmt.Sprintf("{{ $.%s.%s }}", model, field)
}

// RandomEnum selects and returns a random string from the provided values.
// Useful for picking a random value from a set of predefined options.
func (h *FixtureHelpers) RandomEnum(values ...string) string {
	if len(values) == 0 {
		return ""
	}

	idx, err := secureRandomInt(0, int64(len(values)-1))
	if err != nil {
		return ""
	}

	return values[idx]
}

// RandomPhone generates and returns a random 10-digit phone number formatted as
// XXX-XXX-XXXX, ensuring that the first digit is never zero.
func (h *FixtureHelpers) RandomPhone() string {
	var digits [10]int64

	// First digit (1-9)
	first, err := secureRandomInt(1, 9)
	if err != nil {
		return ""
	}
	digits[0] = first

	// Generate remaining 9 digits
	for i := 1; i < 10; i++ {
		digit, err := secureRandomInt(0, 9)
		if err != nil {
			return ""
		}
		digits[i] = digit
	}

	return fmt.Sprintf("%d%d%d-%d%d%d-%d%d%d%d",
		digits[0], digits[1], digits[2],
		digits[3], digits[4], digits[5],
		digits[6], digits[7], digits[8], digits[9])
}

// Concat combines multiple slices of interfaces into a single slice. This is
// useful for building larger collections from smaller ones.
func (h *FixtureHelpers) Concat(arrays ...[]any) []any {
	var result []any
	for _, arr := range arrays {
		result = append(result, arr...)
	}
	return result
}

// Slice returns a portion of the given slice between the specified start and end
// indices. If indices are out of range, they are adjusted to valid boundaries.
func (h *FixtureHelpers) Slice(start, end int64, items []any) []any {
	if start < 0 {
		start = 0
	}
	if end > int64(len(items)) {
		end = int64(len(items))
	}
	return items[start:end]
}

// Repeat returns a slice containing the specified value repeated count times.
// Useful for generating repeated data in fixtures.
func (h *FixtureHelpers) Repeat(count int64, value any) []any {
	result := make([]any, count)
	for i := int64(0); i < count; i++ {
		result[i] = value
	}
	return result
}

// Random returns a random integer between min and max (inclusive).
func (h *FixtureHelpers) Random(minX, maxX int64) int64 {
	idx, err := secureRandomInt(minX, maxX)
	if err != nil {
		return 0
	}
	return idx
}

// Sample returns a cryptographically secure random sample of elements
func (h *FixtureHelpers) Sample(count int64, items []any) ([]any, error) {
	if len(items) == 0 {
		return nil, eris.New("empty items slice")
	}

	if count > int64(len(items)) {
		count = int64(len(items))
	}

	result := make([]any, count)
	tempItems := make([]any, len(items))
	copy(tempItems, items)

	for i := int64(0); i < count; i++ {
		idx, err := secureRandomInt(0, int64(len(tempItems)-1))
		if err != nil {
			return nil, fmt.Errorf("failed to generate random index: %w", err)
		}

		result[i] = tempItems[idx]
		// Remove selected item to avoid duplicates
		tempItems[idx] = tempItems[len(tempItems)-1]
		tempItems = tempItems[:len(tempItems)-1]
	}

	return result, nil
}

// Shuffle returns a new slice with all elements randomly shuffled using crypto/rand
func (h *FixtureHelpers) Shuffle(items []any) ([]any, error) {
	result := make([]any, len(items))
	copy(result, items)

	for i := len(result) - 1; i > 0; i-- {
		j, err := secureRandomInt(0, int64(i))
		if err != nil {
			return nil, fmt.Errorf("failed to generate random index for shuffle: %w", err)
		}
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}
