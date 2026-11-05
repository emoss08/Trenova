package chainsnap

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	TimePlaceholder     = "<time>"
	DatePlaceholder     = "<date>"
	VolatilePlaceholder = "<volatile>"

	unixSecondsFloor = 1_000_000_000
	unixSecondsCeil  = 10_000_000_000
	unixMillisFloor  = 1_000_000_000_000
	unixMillisCeil   = 10_000_000_000_000
	nearNow          = 400 * 24 * 60 * 60
)

var (
	idPattern      = regexp.MustCompile(`\b[a-z][a-z0-9]{0,11}_[0-9A-HJKMNP-TV-Z]{26}\b`)
	timeKeyPattern = regexp.MustCompile(
		`[a-z](?:At|Date|Time|Timestamp|Until|Since|Deadline)$|_(?:at|date|time)$|^(?:date|time|timestamp|deadline)$`,
	)
	rfc3339Prefixes = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}`)
	datePattern     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

type Normalizer struct {
	aliases  map[string]string
	counts   map[string]int
	volatile [][]string
	literals map[string]string
	replacer *strings.Replacer
	now      int64
}

func IsID(text string) bool {
	return idPattern.MatchString(text) && idPattern.FindString(text) == text
}

type Options struct {
	Named    map[string]string
	Volatile []string
	Literals map[string]string
	Now      int64
}

func NewNormalizer(opts Options) *Normalizer {
	named, volatile := opts.Named, opts.Volatile
	n := &Normalizer{
		aliases:  make(map[string]string, len(named)),
		counts:   map[string]int{},
		literals: map[string]string{},
		now:      opts.Now,
	}
	for literal, placeholder := range opts.Literals {
		n.literals[literal] = placeholder
	}
	n.rebuild()
	for _, id := range slices.Sorted(maps.Keys(named)) {
		n.aliases[id] = "$" + named[id]
	}
	for _, path := range volatile {
		n.volatile = append(n.volatile, strings.Split(path, "."))
	}

	return n
}

func (n *Normalizer) AddLiteral(literal, placeholder string) {
	if literal == "" {
		return
	}
	n.literals[literal] = placeholder
	n.rebuild()
}

func (n *Normalizer) rebuild() {
	ordered := slices.SortedFunc(maps.Keys(n.literals), func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	pairs := make([]string, 0, 2*len(ordered))
	for _, literal := range ordered {
		pairs = append(pairs, literal, n.literals[literal])
	}
	n.replacer = strings.NewReplacer(pairs...)
}

func (n *Normalizer) Alias(id string) string {
	if alias, ok := n.aliases[id]; ok {
		return alias
	}
	prefix, _, _ := strings.Cut(id, "_")
	n.counts[prefix]++
	alias := fmt.Sprintf("$%s_%d", prefix, n.counts[prefix])
	n.aliases[id] = alias

	return alias
}

func (n *Normalizer) Value(path []string, value any) any {
	for _, pattern := range n.volatile {
		if matches(pattern, path) {
			return VolatilePlaceholder
		}
	}

	key := ""
	if len(path) > 0 {
		key = path[len(path)-1]
	}

	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for _, k := range slices.Sorted(maps.Keys(typed)) {
			out[k] = n.Value(append(slices.Clip(path), k), typed[k])
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = n.Value(append(slices.Clip(path), fmt.Sprint(i)), item)
		}
		return out
	case string:
		if timeKeyPattern.MatchString(key) && rfc3339Prefixes.MatchString(typed) {
			if _, err := time.Parse(time.RFC3339, typed); err == nil {
				return TimePlaceholder
			}
		}
		if timeKeyPattern.MatchString(key) && datePattern.MatchString(typed) {
			return DatePlaceholder
		}
		return idPattern.ReplaceAllStringFunc(n.replacer.Replace(typed), n.Alias)
	case float64:
		if (timeKeyPattern.MatchString(key) && isUnixTime(typed)) || n.nearNow(typed) {
			return TimePlaceholder
		}
		return typed
	case int64:
		if n.nearNow(float64(typed)) {
			return TimePlaceholder
		}
		return typed
	default:
		return value
	}
}

func (n *Normalizer) nearNow(value float64) bool {
	if n.now == 0 || value != float64(int64(value)) {
		return false
	}
	seconds := int64(value)
	if seconds >= unixMillisFloor {
		seconds /= 1000
	}

	return seconds > n.now-nearNow && seconds < n.now+nearNow
}

func isUnixTime(value float64) bool {
	if value != float64(int64(value)) {
		return false
	}

	return (value >= unixSecondsFloor && value < unixSecondsCeil) ||
		(value >= unixMillisFloor && value < unixMillisCeil)
}

func matches(pattern, path []string) bool {
	if len(pattern) != len(path) {
		return false
	}
	for i := range pattern {
		if pattern[i] != "*" && pattern[i] != path[i] {
			return false
		}
	}

	return true
}
