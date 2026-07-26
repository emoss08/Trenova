package reportfmt

import (
	"errors"
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

// MaxRules bounds a column's rule list. Conditional formatting is meant to make
// exceptions obvious; past a handful of bands it stops communicating anything.
const MaxRules = 8

// Tone is the meaning a matched rule assigns to a cell, not a colour. The
// surface decides how to paint it, so light and dark themes and the XLSX
// palette stay consistent without the definition knowing about any of them.
type Tone string

const (
	ToneNone     = Tone("")
	TonePositive = Tone("positive")
	ToneWarning  = Tone("warning")
	ToneNegative = Tone("negative")
	ToneNeutral  = Tone("neutral")
)

func (t Tone) IsValid() bool {
	switch t {
	case ToneNone, TonePositive, ToneWarning, ToneNegative, ToneNeutral:
		return true
	default:
		return false
	}
}

type RuleOp string

const (
	RuleGreaterThan  = RuleOp("gt")
	RuleGreaterEqual = RuleOp("gte")
	RuleLessThan     = RuleOp("lt")
	RuleLessEqual    = RuleOp("lte")
	RuleEqual        = RuleOp("eq")
	RuleBetween      = RuleOp("between")
)

func (o RuleOp) IsValid() bool {
	switch o {
	case RuleGreaterThan, RuleGreaterEqual, RuleLessThan, RuleLessEqual,
		RuleEqual, RuleBetween:
		return true
	default:
		return false
	}
}

func (o RuleOp) NeedsUpper() bool { return o == RuleBetween }

// Rule paints a cell when its value falls in a band. Rules are evaluated in
// order and the first match wins, so an author can read them top to bottom.
type Rule struct {
	Op    RuleOp  `json:"op"`
	Value float64 `json:"value"`
	Upper float64 `json:"upper,omitempty"`
	Tone  Tone    `json:"tone"`
}

func (r *Rule) Validate() error {
	if !r.Op.IsValid() {
		return fmt.Errorf("unknown comparison %q", r.Op)
	}
	if !r.Tone.IsValid() || r.Tone == ToneNone {
		return fmt.Errorf("unknown emphasis %q", r.Tone)
	}
	if math.IsNaN(r.Value) || math.IsInf(r.Value, 0) {
		return errors.New("the threshold must be a finite number")
	}
	if !r.Op.NeedsUpper() {
		return nil
	}
	if math.IsNaN(r.Upper) || math.IsInf(r.Upper, 0) {
		return errors.New("the upper bound must be a finite number")
	}
	if r.Upper < r.Value {
		return errors.New("the upper bound must not be below the lower bound")
	}
	return nil
}

func (r *Rule) matches(value float64) bool {
	switch r.Op {
	case RuleGreaterThan:
		return value > r.Value
	case RuleGreaterEqual:
		return value >= r.Value
	case RuleLessThan:
		return value < r.Value
	case RuleLessEqual:
		return value <= r.Value
	case RuleEqual:
		return value == r.Value
	case RuleBetween:
		return value >= r.Value && value <= r.Upper
	default:
		return false
	}
}

// ToneFor returns the emphasis a value earns, or ToneNone when no rule matches.
// Only numeric cells are banded — a rule compares against a threshold, and a
// string or a boolean has no position on that scale.
func (r *Resolved) ToneFor(value any) Tone {
	if len(r.Rules) == 0 || value == nil {
		return ToneNone
	}

	number, ok := ruleValue(value)
	if !ok {
		return ToneNone
	}
	// Percent columns are stored as a fraction and displayed times 100, so the
	// threshold an author typed ("over 12%") has to be compared on that scale.
	if r.Style == StylePercent {
		number *= 100
	}

	for i := range r.Rules {
		if r.Rules[i].matches(number) {
			return r.Rules[i].Tone
		}
	}
	return ToneNone
}

func ruleValue(value any) (float64, bool) {
	switch v := value.(type) {
	case decimal.Decimal:
		result, _ := v.Float64()
		return result, true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func validateRules(rules []Rule) error {
	if len(rules) > MaxRules {
		return fmt.Errorf("a column can carry at most %d formatting rules", MaxRules)
	}
	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
