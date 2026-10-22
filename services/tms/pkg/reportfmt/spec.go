package reportfmt

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/reportcatalog"
)

type Style string

const (
	StyleAuto     = Style("")
	StyleNumber   = Style("number")
	StyleCurrency = Style("currency")
	StylePercent  = Style("percent")
	StyleDuration = Style("duration")
	StyleDate     = Style("date")
	StyleBool     = Style("bool")
	StyleText     = Style("text")
)

func (s Style) IsValid() bool {
	switch s {
	case StyleAuto, StyleNumber, StyleCurrency, StylePercent, StyleDuration,
		StyleDate, StyleBool, StyleText:
		return true
	default:
		return false
	}
}

type Negative string

const (
	NegativeAuto        = Negative("")
	NegativeMinus       = Negative("minus")
	NegativeParentheses = Negative("parentheses")
)

func (n Negative) IsValid() bool {
	switch n {
	case NegativeAuto, NegativeMinus, NegativeParentheses:
		return true
	default:
		return false
	}
}

type Notation string

const (
	NotationAuto     = Notation("")
	NotationStandard = Notation("standard")
	NotationCompact  = Notation("compact")
)

func (n Notation) IsValid() bool {
	switch n {
	case NotationAuto, NotationStandard, NotationCompact:
		return true
	default:
		return false
	}
}

type DateStyle string

const (
	DateStyleAuto      = DateStyle("")
	DateStyleDate      = DateStyle("date")
	DateStyleDateLong  = DateStyle("dateLong")
	DateStyleDateTime  = DateStyle("dateTime")
	DateStyleTime      = DateStyle("time")
	DateStyleMonthYear = DateStyle("monthYear")
	DateStyleQuarter   = DateStyle("quarter")
	DateStyleYear      = DateStyle("year")
	DateStyleWeekday   = DateStyle("weekday")
	DateStyleISO       = DateStyle("iso")
)

func (d DateStyle) IsValid() bool {
	switch d {
	case DateStyleAuto, DateStyleDate, DateStyleDateLong, DateStyleDateTime, DateStyleTime,
		DateStyleMonthYear, DateStyleQuarter, DateStyleYear, DateStyleWeekday, DateStyleISO:
		return true
	default:
		return false
	}
}

type BoolStyle string

const (
	BoolStyleAuto      BoolStyle = ""
	BoolStyleYesNo     BoolStyle = "yesNo"
	BoolStyleTrueFalse BoolStyle = "trueFalse"
	BoolStyleOnOff     BoolStyle = "onOff"
	BoolStyleCheck     BoolStyle = "check"
	BoolStyleOneZero   BoolStyle = "oneZero"
)

func (b BoolStyle) IsValid() bool {
	switch b {
	case BoolStyleAuto, BoolStyleYesNo, BoolStyleTrueFalse, BoolStyleOnOff,
		BoolStyleCheck, BoolStyleOneZero:
		return true
	default:
		return false
	}
}

type DurationUnit string

const (
	DurationUnitAuto    DurationUnit = ""
	DurationUnitSeconds DurationUnit = "seconds"
	DurationUnitMinutes DurationUnit = "minutes"
	DurationUnitHours   DurationUnit = "hours"
	DurationUnitDays    DurationUnit = "days"
)

func (u DurationUnit) IsValid() bool {
	switch u {
	case DurationUnitAuto, DurationUnitSeconds, DurationUnitMinutes,
		DurationUnitHours, DurationUnitDays:
		return true
	default:
		return false
	}
}

type DurationStyle string

const (
	DurationStyleAuto    DurationStyle = ""
	DurationStyleClock   DurationStyle = "clock"
	DurationStyleCompact DurationStyle = "compact"
	DurationStyleVerbose DurationStyle = "verbose"
	DurationStyleDecimal DurationStyle = "decimalHours"
)

func (d DurationStyle) IsValid() bool {
	switch d {
	case DurationStyleAuto, DurationStyleClock, DurationStyleCompact,
		DurationStyleVerbose, DurationStyleDecimal:
		return true
	default:
		return false
	}
}

const (
	MaxDecimals      = 10
	maxAffixLength   = 12
	maxNullTextChars = 24
)

// Spec is the author-supplied display override for a report column. Every
// field is optional; unset fields fall back to the defaults derived from the
// column's catalog type and format hint.
type Spec struct {
	Style         Style         `json:"style,omitempty"`
	Decimals      *int          `json:"decimals,omitempty"`
	Grouping      *bool         `json:"grouping,omitempty"`
	Currency      string        `json:"currency,omitempty"`
	Negative      Negative      `json:"negative,omitempty"`
	Notation      Notation      `json:"notation,omitempty"`
	Prefix        string        `json:"prefix,omitempty"`
	Suffix        string        `json:"suffix,omitempty"`
	DateStyle     DateStyle     `json:"dateStyle,omitempty"`
	BoolStyle     BoolStyle     `json:"boolStyle,omitempty"`
	DurationUnit  DurationUnit  `json:"durationUnit,omitempty"`
	DurationStyle DurationStyle `json:"durationStyle,omitempty"`
	NullText      string        `json:"nullText,omitempty"`
	// Rules band a numeric column so an exception is visible without reading
	// every number. First match wins.
	Rules []Rule `json:"rules,omitempty"`
}

// Resolved is the fully defaulted display contract shared by the renderers,
// the JSON envelope, and the client preview grid.
type Resolved struct {
	Style         Style         `json:"style"`
	Decimals      int           `json:"decimals"`
	Grouping      bool          `json:"grouping"`
	Currency      string        `json:"currency"`
	Negative      Negative      `json:"negative"`
	Notation      Notation      `json:"notation"`
	Prefix        string        `json:"prefix"`
	Suffix        string        `json:"suffix"`
	DateStyle     DateStyle     `json:"dateStyle"`
	BoolStyle     BoolStyle     `json:"boolStyle"`
	DurationUnit  DurationUnit  `json:"durationUnit"`
	DurationStyle DurationStyle `json:"durationStyle"`
	NullText      string        `json:"nullText"`
	Rules         []Rule        `json:"rules,omitempty"`
	// Band is set when the column groups a numeric dimension into ranges; the
	// scanned value is the range's lower bound.
	Band *Band `json:"band,omitempty"`
}

// Validate rejects display specs that cannot apply to the column's resolved
// value type, so authors get a compile error instead of a silently ignored
// option.
// IsEmpty reports whether the author overrode nothing, so a display carrying
// only defaults is not persisted. Spec holds a slice and is therefore not
// comparable, so the check lives with the type rather than at each call site.
func (s *Spec) IsEmpty() bool {
	if s == nil {
		return true
	}
	return s.Style == StyleAuto &&
		s.Decimals == nil &&
		s.Grouping == nil &&
		s.Currency == "" &&
		s.Negative == NegativeAuto &&
		s.Notation == NotationAuto &&
		s.Prefix == "" &&
		s.Suffix == "" &&
		s.DateStyle == DateStyleAuto &&
		s.BoolStyle == BoolStyleAuto &&
		s.DurationUnit == DurationUnitAuto &&
		s.DurationStyle == DurationStyleAuto &&
		s.NullText == "" &&
		len(s.Rules) == 0
}

func (s *Spec) Validate(fieldType reportcatalog.FieldType) error {
	if s == nil {
		return nil
	}
	if !s.Style.IsValid() {
		return fmt.Errorf("unknown display style %q", s.Style)
	}
	if !s.Negative.IsValid() {
		return fmt.Errorf("unknown negative style %q", s.Negative)
	}
	if !s.Notation.IsValid() {
		return fmt.Errorf("unknown notation %q", s.Notation)
	}
	if err := validateRules(s.Rules); err != nil {
		return err
	}
	if !s.DateStyle.IsValid() {
		return fmt.Errorf("unknown date style %q", s.DateStyle)
	}
	if !s.BoolStyle.IsValid() {
		return fmt.Errorf("unknown boolean style %q", s.BoolStyle)
	}
	if !s.DurationUnit.IsValid() {
		return fmt.Errorf("unknown duration unit %q", s.DurationUnit)
	}
	if !s.DurationStyle.IsValid() {
		return fmt.Errorf("unknown duration style %q", s.DurationStyle)
	}
	if s.Decimals != nil && (*s.Decimals < 0 || *s.Decimals > MaxDecimals) {
		return fmt.Errorf("decimals must be between 0 and %d", MaxDecimals)
	}
	if s.Currency != "" && len(s.Currency) != 3 {
		return fmt.Errorf("currency must be a 3-letter ISO code, got %q", s.Currency)
	}
	if len([]rune(s.Prefix)) > maxAffixLength || len([]rune(s.Suffix)) > maxAffixLength {
		return fmt.Errorf("prefix and suffix are limited to %d characters", maxAffixLength)
	}
	if len([]rune(s.NullText)) > maxNullTextChars {
		return fmt.Errorf("empty-value text is limited to %d characters", maxNullTextChars)
	}

	return s.validateStyleForType(fieldType)
}

func (s *Spec) validateStyleForType(fieldType reportcatalog.FieldType) error {
	if s.Style == StyleAuto {
		return nil
	}
	if !styleAllowed(s.Style, fieldType) {
		return fmt.Errorf("display style %q is not valid for %s columns", s.Style, fieldType)
	}
	return nil
}

func styleAllowed(style Style, fieldType reportcatalog.FieldType) bool {
	//nolint:exhaustive // string-like types only accept the text style
	switch fieldType {
	case reportcatalog.FieldInt, reportcatalog.FieldDecimal:
		// Date is legal here because aggregates over epoch columns (an average
		// arrival time, say) come back as numbers that still denote an instant.
		return style == StyleNumber || style == StyleCurrency ||
			style == StylePercent || style == StyleDuration ||
			style == StyleDate || style == StyleText
	case reportcatalog.FieldEpoch:
		return style == StyleDate || style == StyleNumber || style == StyleText
	case reportcatalog.FieldBool:
		return style == StyleBool || style == StyleText
	default:
		return style == StyleText
	}
}
