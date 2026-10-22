package tarifftypes

import (
	"bytes"
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

func (i Inputs) Equal(other Inputs) bool {
	if len(i) != len(other) {
		return false
	}
	for name, value := range i {
		theirs, ok := other[name]
		if !ok || !bytes.Equal(compact(value), compact(theirs)) {
			return false
		}
	}
	return true
}

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

type InputDecl struct {
	Name       string        `json:"name"`
	Doc        string        `json:"doc,omitempty"`
	Type       string        `json:"type"`
	Descriptor rules.RawJSON `json:"descriptor"`
	Required   bool          `json:"required"`
	Default    rules.RawJSON `json:"default,omitempty"`
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
		})
	}
	return out
}

type Import struct {
	Path          string   `json:"path"`
	LibraryID     pulid.ID `json:"libraryId"`
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
