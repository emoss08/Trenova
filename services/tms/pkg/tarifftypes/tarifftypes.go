package tarifftypes

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Inputs map[string]rules.RawJSON

func (i Inputs) Clone() Inputs {
	if i == nil {
		return nil
	}
	out := make(Inputs, len(i))
	for name, value := range i {
		out[name] = slices.Clone(value)
	}
	return out
}

func (i Inputs) OrEmpty() Inputs {
	if i == nil {
		return Inputs{}
	}
	return i
}

func (i Inputs) Equal(other Inputs) bool {
	if len(i) != len(other) {
		return false
	}
	for name, value := range i {
		theirs, ok := other[name]
		if !ok || !EqualValues(value, theirs) {
			return false
		}
	}
	return true
}

func EqualValues(a, b rules.RawJSON) bool {
	var left, right any
	if exactJSON.Unmarshal(a, &left) != nil || exactJSON.Unmarshal(b, &right) != nil {
		return bytes.Equal(a, b)
	}
	return sameValue(left, right)
}

func sameValue(left, right any) bool {
	switch l := left.(type) {
	case map[string]any:
		r, ok := right.(map[string]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for key, value := range l {
			theirs, present := r[key]
			if !present {
				return false
			}
			if key == wireQuantityKey || key == wireNumberKey {
				if !sameNumber(value, theirs) {
					return false
				}
				continue
			}
			if !sameValue(value, theirs) {
				return false
			}
		}
		return true
	case []any:
		r, ok := right.([]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for index := range l {
			if !sameValue(l[index], r[index]) {
				return false
			}
		}
		return true
	default:
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
}

const (
	wireQuantityKey = "$q"
	wireNumberKey   = "$n"
)

func compact(raw rules.RawJSON) []byte {
	var decoded any
	if err := sonic.Unmarshal(raw, &decoded); err != nil {
		return raw
	}
	out, err := sonic.ConfigStd.Marshal(decoded)
	if err != nil {
		return raw
	}
	return out
}

func sameNumber(left, right any) bool {
	l, lErr := decimal.NewFromString(fmt.Sprint(left))
	r, rErr := decimal.NewFromString(fmt.Sprint(right))
	if lErr != nil || rErr != nil {
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
	return l.Equal(r)
}

type InputDecl struct {
	Name       string                  `json:"name"`
	Doc        string                  `json:"doc,omitempty"`
	Type       string                  `json:"type"`
	Descriptor rules.RawJSON           `json:"descriptor"`
	Required   bool                    `json:"required"`
	Default    rules.RawJSON           `json:"default,omitempty"`
	Constraint *rules.TariffConstraint `json:"constraint,omitempty"`
}

func InputDeclsFrom(inputs []rules.TariffInput) []InputDecl {
	out := make([]InputDecl, 0, len(inputs))
	for i := range inputs {
		in := &inputs[i]
		out = append(out, InputDecl{
			Name:       in.Name,
			Doc:        in.Doc,
			Type:       in.Type,
			Descriptor: in.Descriptor,
			Required:   in.Required,
			Default:    in.Default,
			Constraint: in.Constraint,
		})
	}
	return out
}

type Import struct {
	Path          string   `json:"path"`
	LibraryID     pulid.ID `json:"libraryId"`
	VersionNumber int64    `json:"versionNumber"`
}

type TablePin struct {
	Code          string   `json:"code"`
	RateMatrixID  pulid.ID `json:"rateMatrixId"`
	VersionNumber int64    `json:"versionNumber"`
}

type LineKind string

const (
	LineCharge   = LineKind("Charge")
	LineMinimum  = LineKind("Minimum")
	LineMaximum  = LineKind("Maximum")
	LineRounding = LineKind("Rounding")
)

type Line struct {
	Kind         LineKind         `json:"kind"`
	Label        string           `json:"label,omitempty"`
	Code         string           `json:"code,omitempty"`
	Freight      bool             `json:"freight"`
	Amount       decimal.Decimal  `json:"amount"`
	Quantity     *decimal.Decimal `json:"quantity,omitempty"`
	QuantityUnit string           `json:"quantityUnit,omitempty"`
	Rate         *decimal.Decimal `json:"rate,omitempty"`
	RateUnit     string           `json:"rateUnit,omitempty"`
	Limit        *LineLimit       `json:"limit,omitempty"`
	On           string           `json:"on,omitempty"`
}

type LineLimit struct {
	Kind     LineKind        `json:"kind"`
	Computed decimal.Decimal `json:"computed"`
}

type Note struct {
	Text string `json:"text"`
}

type Exchange struct {
	Currency string          `json:"currency"`
	Rate     decimal.Decimal `json:"rate"`
}

type Bill struct {
	Currency         string          `json:"currency"`
	Lines            []Line          `json:"lines"`
	Notes            []Note          `json:"notes,omitempty"`
	Reviews          []Note          `json:"reviews,omitempty"`
	FreightTotal     decimal.Decimal `json:"freightTotal"`
	AccessorialTotal decimal.Decimal `json:"accessorialTotal"`
	Total            decimal.Decimal `json:"total"`
	Exchange         *Exchange       `json:"exchange,omitempty"`
}

func (b *Bill) FreightLines() []Line {
	out := make([]Line, 0, len(b.Lines))
	for i := range b.Lines {
		if b.Lines[i].Code == "" {
			out = append(out, b.Lines[i])
		}
	}
	return out
}

func (b *Bill) CodedLines() []Line {
	out := make([]Line, 0, len(b.Lines))
	for i := range b.Lines {
		if b.Lines[i].Code != "" {
			out = append(out, b.Lines[i])
		}
	}
	return out
}

func (b *Bill) BillsCode(code string) bool {
	for i := range b.Lines {
		if b.Lines[i].Code == code {
			return true
		}
	}
	return false
}
