package stringutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
)

type testMethod string

func (m testMethod) String() string {
	return string(m)
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"equal to max", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello..."},
		{"empty string", "", 10, ""},
		{"max length zero", "hello", 0, "..."},
		{"single character truncation", "ab", 1, "a..."},
		{"unicode string truncates by bytes", "héllo wörld", 6, "héllo..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := stringutils.Truncate(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match", "hello", "hello", true},
		{"substring at start", "hello world", "hello", true},
		{"substring at end", "hello world", "world", true},
		{"substring in middle", "hello world", "lo wo", true},
		{"not found", "hello world", "xyz", false},
		{"empty substring", "hello", "", true},
		{"empty string with empty substr", "", "", true},
		{"empty string with non-empty substr", "", "a", false},
		{"single character found", "hello", "e", true},
		{"single character not found", "hello", "x", false},
		{"case sensitive", "Hello", "hello", false},
		{"substring longer than string", "hi", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := stringutils.Contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"filters out empty strings", []string{"a", "", "b"}, []string{"a", "b"}},
		{"filters out whitespace only strings", []string{"a", "   ", "b"}, []string{"a", "b"}},
		{"filters tabs and newlines", []string{"a", "\t", "\n", "b"}, []string{"a", "b"}},
		{"returns empty slice for all empty input", []string{"", "   ", "\t"}, nil},
		{
			"preserves and trims strings",
			[]string{"  hello  ", "world  "},
			[]string{"hello", "world"},
		},
		{"handles nil input", nil, nil},
		{"handles empty input slice", []string{}, nil},
		{"maintains order", []string{"c", "b", "a"}, []string{"c", "b", "a"}},
		{
			"mixed content",
			[]string{"", "  first  ", "   ", "second", "\t", "third  "},
			[]string{"first", "second", "third"},
		},
		{"single valid element", []string{"only"}, []string{"only"}},
		{"single empty element", []string{""}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := stringutils.FilterEmpty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJoinMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		methods  []testMethod
		expected string
	}{
		{
			name:     "joins multiple methods with comma and space",
			methods:  []testMethod{"air", "ground", "ocean"},
			expected: "air, ground, ocean",
		},
		{
			name:     "returns empty string for empty slice",
			methods:  []testMethod{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := stringutils.JoinMethods(tt.methods)

			assert.Equal(t, tt.expected, result)
		})
	}
}
