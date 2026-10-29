package carrierintelrule

import (
	"strconv"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

func ValidateRuleSettings(settings RuleSettings, multiErr *errortypes.MultiError) {
	for code, setting := range settings {
		def, ok := RuleByCode(code)
		if !ok {
			multiErr.Add(code.String(), errortypes.ErrInvalid, "Unknown rule")
			continue
		}
		if !setting.Action.IsValid() {
			multiErr.Add(code.String()+".action", errortypes.ErrInvalid, "Rule action is invalid")
		}
		for key, raw := range setting.Params {
			spec := def.paramSpec(key)
			if spec == nil {
				multiErr.Add(code.String()+".params."+key, errortypes.ErrInvalid,
					"Unknown rule parameter")
				continue
			}
			if msg := validateParam(spec, raw); msg != "" {
				multiErr.Add(code.String()+".params."+key, errortypes.ErrInvalid, msg)
			}
		}
	}
}

func validateParam(spec *RuleParamSpec, raw string) string {
	if raw == "" {
		return ""
	}
	var value float64
	switch spec.Type {
	case RuleParamTypeInteger:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return spec.Label + " must be a whole number"
		}
		value = float64(v)
	case RuleParamTypeDecimal:
		v, err := decimal.NewFromString(raw)
		if err != nil {
			return spec.Label + " must be a number"
		}
		value = v.InexactFloat64()
	case RuleParamTypeNumber:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return spec.Label + " must be a number"
		}
		value = v
	case RuleParamTypeSelect:
		for _, option := range spec.Options {
			if option == raw {
				return ""
			}
		}
		return spec.Label + " has an invalid option"
	case RuleParamTypeMultiSelect:
		ctx := &RuleContext{params: map[string]string{spec.Key: raw}, spec: &RuleDefinition{}}
		for _, item := range ctx.paramList(spec.Key) {
			found := false
			for _, option := range spec.Options {
				if option == item {
					found = true
					break
				}
			}
			if !found {
				return spec.Label + " has an invalid option"
			}
		}
		return ""
	default:
		return ""
	}
	if spec.Min != nil && value < *spec.Min {
		return spec.Label + " is below the minimum"
	}
	if spec.Max != nil && value > *spec.Max {
		return spec.Label + " is above the maximum"
	}
	return ""
}
