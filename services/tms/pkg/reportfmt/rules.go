package reportfmt

import (
	"errors"
	"fmt"
	"math"
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
