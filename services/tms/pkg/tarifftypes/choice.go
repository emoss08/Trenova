package tarifftypes

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

type ChoiceKind string

const (
	ChoiceBest  = ChoiceKind("best")
	ChoiceWorst = ChoiceKind("worst")
)

type OptionStatus string

const (
	OptionChosen  = OptionStatus("chosen")
	OptionLost    = OptionStatus("lost")
	OptionRefused = OptionStatus("refused")
)

var choiceKinds = map[rules.TariffChoiceKind]ChoiceKind{
	rules.TariffChoiceBest:  ChoiceBest,
	rules.TariffChoiceWorst: ChoiceWorst,
}

var optionStatuses = map[rules.TariffOptionStatus]OptionStatus{
	rules.TariffOptionChosen:  OptionChosen,
	rules.TariffOptionLost:    OptionLost,
	rules.TariffOptionRefused: OptionRefused,
}

type ChoiceOption struct {
	Label    string           `json:"label"`
	Status   OptionStatus     `json:"status"`
	Subtotal *decimal.Decimal `json:"subtotal,omitempty"`
	Refusal  string           `json:"refusal,omitempty"`
}

type Choice struct {
	Label    string         `json:"label"`
	Kind     ChoiceKind     `json:"kind"`
	Currency string         `json:"currency"`
	Chosen   *string        `json:"chosen,omitempty"`
	Options  []ChoiceOption `json:"options"`
}

type ChoicesRequest struct {
	Choices  []rules.TariffChoice
	Currency string
	Locale   i18n.Locale
}

func ChoicesFrom(req *ChoicesRequest) ([]Choice, error) {
	out := make([]Choice, 0, len(req.Choices))
	for i := range req.Choices {
		choice, err := choiceFrom(&req.Choices[i], req)
		if err != nil {
			return nil, err
		}
		out = append(out, choice)
	}
	return out, nil
}

func choiceFrom(choice *rules.TariffChoice, req *ChoicesRequest) (Choice, error) {
	kind, ok := choiceKinds[choice.Kind]
	if !ok {
		return Choice{}, fmt.Errorf("%w: choice kind %q", ErrBillShape, choice.Kind)
	}
	out := Choice{
		Label:    choice.Label,
		Kind:     kind,
		Currency: req.Currency,
		Options:  make([]ChoiceOption, 0, len(choice.Options)),
	}
	if choice.Chosen != nil {
		chosen := *choice.Chosen
		out.Chosen = &chosen
	}
	for i := range choice.Options {
		option, err := optionFrom(&choice.Options[i], req.Locale)
		if err != nil {
			return Choice{}, err
		}
		out.Options = append(out.Options, option)
	}
	return out, nil
}

func optionFrom(option *rules.TariffChoiceOption, locale i18n.Locale) (ChoiceOption, error) {
	status, ok := optionStatuses[option.Status]
	if !ok {
		return ChoiceOption{}, fmt.Errorf("%w: option status %q", ErrBillShape, option.Status)
	}
	out := ChoiceOption{Label: option.Label, Status: status}
	if !option.Subtotal.IsNone() {
		subtotal, err := rules.DecodeQuantity(option.Subtotal)
		if err != nil {
			return ChoiceOption{}, fmt.Errorf("%w: option subtotal: %w", ErrBillShape, err)
		}
		if !subtotal.IsMoney() {
			return ChoiceOption{}, fmt.Errorf(
				"%w: option subtotal in %q, not money",
				ErrBillShape,
				subtotal.Unit,
			)
		}
		out.Subtotal = &subtotal.Value
	}
	if option.Refusal != nil {
		out.Refusal = option.Refusal.Localize(locale)
	}
	return out, nil
}
