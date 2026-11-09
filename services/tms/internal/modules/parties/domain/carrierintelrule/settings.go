package carrierintelrule

import (
	"maps"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fieldpath"
	"github.com/shopspring/decimal"
)

func ValidateRuleSettings(settings RuleSettings, multiErr *errortypes.MultiError) {
	for _, code := range slices.Sorted(maps.Keys(settings)) {
		setting := settings[code]
		rule := fieldpath.Of(fieldpath.Key(code.String()))
		def, ok := RuleByCode(code)
		if !ok {
			multiErr.AddAt(rule, errortypes.ErrInvalid, "Unknown rule")
			continue
		}
		if !setting.Action.IsValid() {
			multiErr.AddAt(
				rule.Append(fieldpath.Key("action")),
				errortypes.ErrInvalid,
				"Rule action is invalid",
			)
		}
		for _, key := range slices.Sorted(maps.Keys(setting.Params)) {
			param := rule.Append(fieldpath.Key("params"), fieldpath.Key(key))
			spec := def.paramSpec(key)
			if spec == nil {
				multiErr.AddAt(param, errortypes.ErrInvalid, "Unknown rule parameter")
				continue
			}
			if msg := validateParam(spec, setting.Params[key]); msg != "" {
				multiErr.AddAt(param, errortypes.ErrInvalid, msg)
			}
		}
	}
}

func validateParam(spec *RuleParamSpec, raw string) string {
	if raw == "" {
		return ""
	}
	switch spec.Type {
	case RuleParamTypeInteger, RuleParamTypeDecimal, RuleParamTypeNumber:
		value, msg := parseNumericParam(spec, raw)
		if msg != "" {
			return msg
		}
		return validateParamRange(spec, value)
	case RuleParamTypeSelect:
		if slices.Contains(spec.Options, raw) {
			return ""
		}
		return spec.Label + " has an invalid option"
	case RuleParamTypeMultiSelect:
		ctx := &RuleContext{params: map[string]string{spec.Key: raw}, spec: &RuleDefinition{}}
		for _, item := range ctx.paramList(spec.Key) {
			if !slices.Contains(spec.Options, item) {
				return spec.Label + " has an invalid option"
			}
		}
		return ""
	default:
		return ""
	}
}

func parseNumericParam(spec *RuleParamSpec, raw string) (value float64, msg string) {
	switch spec.Type {
	case RuleParamTypeInteger:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return 0, spec.Label + " must be a whole number"
		}
		return float64(v), ""
	case RuleParamTypeDecimal:
		v, err := decimal.NewFromString(raw)
		if err != nil {
			return 0, spec.Label + " must be a number"
		}
		return v.InexactFloat64(), ""
	case RuleParamTypeNumber:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, spec.Label + " must be a number"
		}
		return v, ""
	case RuleParamTypeSelect, RuleParamTypeMultiSelect:
		return 0, ""
	default:
		return 0, ""
	}
}

func validateParamRange(spec *RuleParamSpec, value float64) string {
	if spec.Min != nil && value < *spec.Min {
		return spec.Label + " is below the minimum"
	}
	if spec.Max != nil && value > *spec.Max {
		return spec.Label + " is above the maximum"
	}
	return ""
}
