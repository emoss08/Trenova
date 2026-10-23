package ratematrix

import (
	"slices"
	"strings"
	"unicode/utf8"
)

type PostalRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func NormalizePostal(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func NewPostalRange(from, to string) PostalRange {
	return PostalRange{From: NormalizePostal(from), To: NormalizePostal(to)}
}

func (pr PostalRange) Length() int {
	return utf8.RuneCountInString(pr.From)
}

func (pr PostalRange) IsWellFormed() bool {
	length := pr.Length()
	return length > 0 && utf8.RuneCountInString(pr.To) == length && pr.From <= pr.To
}

func (pr PostalRange) Contains(key string) bool {
	normalized := NormalizePostal(key)
	length := pr.Length()
	if utf8.RuneCountInString(normalized) < length {
		return false
	}
	prefix := string([]rune(normalized)[:length])
	return pr.From <= prefix && prefix <= pr.To
}

func (pr PostalRange) Overlaps(other PostalRange) bool {
	return pr.Length() == other.Length() && pr.From <= other.To && other.From <= pr.To
}

func (pr PostalRange) String() string {
	return pr.From + "–" + pr.To
}

type PostalIndex struct {
	byLength map[int][]PostalRange
}

func NewPostalIndex(ranges []PostalRange) *PostalIndex {
	index := &PostalIndex{byLength: make(map[int][]PostalRange, 1)}
	for _, item := range ranges {
		normalized := NewPostalRange(item.From, item.To)
		if !normalized.IsWellFormed() {
			continue
		}
		length := normalized.Length()
		index.byLength[length] = append(index.byLength[length], normalized)
	}
	for length := range index.byLength {
		slices.SortFunc(index.byLength[length], func(a, b PostalRange) int {
			return strings.Compare(a.From, b.From)
		})
	}
	return index
}

func (pi *PostalIndex) Find(key string) []PostalRange {
	if pi == nil {
		return nil
	}
	normalized := NormalizePostal(key)
	keyLength := utf8.RuneCountInString(normalized)
	runes := []rune(normalized)
	found := make([]PostalRange, 0, 1)
	for length, ranges := range pi.byLength {
		if keyLength < length {
			continue
		}
		prefix := string(runes[:length])
		at, _ := slices.BinarySearchFunc(ranges, prefix, func(item PostalRange, target string) int {
			if item.From > target {
				return 1
			}
			return -1
		})
		for i := at - 1; i >= 0; i-- {
			if ranges[i].To >= prefix {
				found = append(found, ranges[i])
				continue
			}
			break
		}
	}
	slices.SortFunc(found, func(a, b PostalRange) int {
		return strings.Compare(a.From, b.From)
	})
	return found
}
