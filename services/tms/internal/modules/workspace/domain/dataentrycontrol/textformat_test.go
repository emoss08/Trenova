package dataentrycontrol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataEntryControl_TextFormat(t *testing.T) {
	t.Parallel()

	dec := &DataEntryControl{
		CodeCase:  CaseFormatUpper,
		NameCase:  CaseFormatTitleCase,
		EmailCase: CaseFormatLower,
		CityCase:  CaseFormatAsEntered,
	}

	assert.Equal(t, TextFormat{
		CodeCase:  CaseFormatUpper,
		NameCase:  CaseFormatTitleCase,
		EmailCase: CaseFormatLower,
		CityCase:  CaseFormatAsEntered,
	}, dec.TextFormat())
}

func TestTextFormat_Code(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format CaseFormat
		in     string
		want   string
	}{
		{
			name:   "upper strips every space",
			format: CaseFormatUpper,
			in:     "  ac me\t01 ",
			want:   "ACME01",
		},
		{name: "lower", format: CaseFormatLower, in: "AcMe 01", want: "acme01"},
		{
			name:   "title keeps the rest of the word",
			format: CaseFormatTitleCase,
			in:     "aCME",
			want:   "ACME",
		},
		{
			name:   "as entered keeps case",
			format: CaseFormatAsEntered,
			in:     " AcMe 01 ",
			want:   "AcMe01",
		},
		{name: "empty stays empty", format: CaseFormatUpper, in: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TextFormat{CodeCase: tt.format}.Code(tt.in))
		})
	}
}

func TestTextFormat_Name(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format CaseFormat
		in     string
		want   string
	}{
		{
			name:   "title case keeps acronyms",
			format: CaseFormatTitleCase,
			in:     "ACME FREIGHT LLC",
			want:   "ACME FREIGHT LLC",
		},
		{
			name:   "title case capitalizes each word",
			format: CaseFormatTitleCase,
			in:     "acme freight llc",
			want:   "Acme Freight Llc",
		},
		{
			name:   "title case keeps inner capitals",
			format: CaseFormatTitleCase,
			in:     "mcDonald freight",
			want:   "McDonald Freight",
		},
		{
			name:   "upper collapses whitespace",
			format: CaseFormatUpper,
			in:     "  acme \t freight ",
			want:   "ACME FREIGHT",
		},
		{name: "lower", format: CaseFormatLower, in: "ACME Freight", want: "acme freight"},
		{
			name:   "as entered only cleans whitespace",
			format: CaseFormatAsEntered,
			in:     " McDonald   Freight LLC ",
			want:   "McDonald Freight LLC",
		},
		{name: "empty stays empty", format: CaseFormatTitleCase, in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TextFormat{NameCase: tt.format}.Name(tt.in))
		})
	}
}

func TestTextFormat_City(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format CaseFormat
		in     string
		want   string
	}{
		{name: "title", format: CaseFormatTitleCase, in: " san  antonio ", want: "San Antonio"},
		{name: "title keeps capitals", format: CaseFormatTitleCase, in: "McAllen", want: "McAllen"},
		{name: "upper", format: CaseFormatUpper, in: "el paso", want: "EL PASO"},
		{name: "lower", format: CaseFormatLower, in: "El Paso", want: "el paso"},
		{name: "as entered", format: CaseFormatAsEntered, in: "eL pASO", want: "eL pASO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TextFormat{CityCase: tt.format}.City(tt.in))
		})
	}
}

func TestTextFormat_Email(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format CaseFormat
		in     string
		want   string
	}{
		{name: "lower", format: CaseFormatLower, in: " Ops@Acme.COM ", want: "ops@acme.com"},
		{name: "upper", format: CaseFormatUpper, in: "Ops@Acme.com", want: "OPS@ACME.COM"},
		{
			name:   "as entered",
			format: CaseFormatAsEntered,
			in:     " Ops@Acme.com",
			want:   "Ops@Acme.com",
		},
		{
			name:   "title case leaves the address as entered",
			format: CaseFormatTitleCase,
			in:     "ops@acme.com",
			want:   "ops@acme.com",
		},
		{
			name:   "a recipient list keeps its separators",
			format: CaseFormatLower,
			in:     "Ops@Acme.com, Track@ACME.com",
			want:   "ops@acme.com, track@acme.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TextFormat{EmailCase: tt.format}.Email(tt.in))
		})
	}
}

func TestCleanText(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", CleanText(""))
	assert.Equal(t, "hello world", CleanText("hello world"))
	assert.Equal(t, "hello", CleanText("  hello  "))
	assert.Equal(t, "hello world", CleanText("hello    world"))
	assert.Equal(t, "hello world", CleanText("hello\t\n\t world"))
}
