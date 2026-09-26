package toolpreview

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

// WouldFailPrefix leads the message of a would_fail warning; the rest is
// the refusal as the write's own rules worded it.
const WouldFailPrefix = "This would be refused as it stands: "

// WouldFail is the warning for a write its own rules would refuse. The
// message keeps the refusal whole; each problem it is made of is a reason,
// with the field it names where it names one, so a person can be told each
// and offered the value to change.
func WouldFail(err error) agent.PreviewWarning {
	refusal := strings.TrimSpace(err.Error())

	return agent.PreviewWarning{
		Code:    agent.PreviewWarningWouldFail,
		Args:    []string{refusal},
		Message: WouldFailPrefix + refusal,
		Reasons: Reasons(err),
	}
}

// Reasons are the problems a refusal is made of: one per field a validation
// error names, or the refusal itself when it names none.
func Reasons(err error) []agent.PreviewReason {
	if err == nil {
		return nil
	}

	if multiErr, ok := errors.AsType[*errortypes.MultiError](err); ok && multiErr.HasErrors() {
		reasons := make([]agent.PreviewReason, 0, min(len(multiErr.Errors), agent.MaxPreviewReasons))
		for _, fieldErr := range multiErr.Errors {
			if len(reasons) == agent.MaxPreviewReasons {
				break
			}
			if fieldErr != nil {
				reasons = append(reasons, fieldReason(fieldErr.Field, fieldErr.Error()))
			}
		}

		return reasons
	}

	if fieldErr, ok := errors.AsType[*errortypes.Error](err); ok {
		return []agent.PreviewReason{fieldReason(fieldErr.Field, fieldErr.Error())}
	}

	if businessErr, ok := errors.AsType[*errortypes.BusinessError](err); ok {
		return []agent.PreviewReason{{Message: strings.TrimSpace(businessErr.Error())}}
	}

	return []agent.PreviewReason{{Message: strings.TrimSpace(err.Error())}}
}

func fieldReason(field, message string) agent.PreviewReason {
	field = strings.TrimSpace(field)

	return agent.PreviewReason{
		Field:   field,
		Label:   fieldLabel(field),
		Message: strings.TrimSpace(message),
	}
}

// fieldLabel is a field path's last name in words: moves[0].locationId is
// "Location ID".
func fieldLabel(field string) string {
	if field == "" {
		return ""
	}

	last := field[strings.LastIndexByte(field, '.')+1:]
	if bracket := strings.IndexByte(last, '['); bracket >= 0 {
		last = last[:bracket]
	}

	return stringutils.HumanizeCamelCaseSentence(last)
}

// LocateReasons names, for each reason of a would_fail warning, the call's
// parameter that carries the field it names: the parameter itself, or the
// field inside the one object parameter that holds the record whole
// (shipment.bol). A field the schema does not reach, or reaches through
// more than one parameter, names none. A parameter already named is kept.
func LocateReasons(warnings []agent.PreviewWarning, schema map[string]any) {
	properties := schemaProperties(schema)
	if len(properties) == 0 {
		return
	}

	for i := range warnings {
		if warnings[i].Code != agent.PreviewWarningWouldFail {
			continue
		}
		reasons := warnings[i].Reasons
		for j := range reasons {
			if reasons[j].Param != "" || reasons[j].Field == "" {
				continue
			}
			reasons[j].Param = locateParam(properties, reasons[j].Field)
		}
	}
}

type pathStep struct {
	name    string
	indices int
}

func locateParam(properties map[string]any, field string) string {
	steps, ok := parsePath(field)
	if !ok {
		return ""
	}
	if reaches(properties, steps) {
		return field
	}

	holder := ""
	for name, raw := range properties {
		nested := schemaProperties(asSchema(raw))
		if len(nested) == 0 || !reaches(nested, steps) {
			continue
		}
		if holder != "" {
			return ""
		}
		holder = name
	}
	if holder == "" {
		return ""
	}

	return holder + "." + field
}

func parsePath(field string) ([]pathStep, bool) {
	segments := strings.Split(field, ".")
	steps := make([]pathStep, 0, len(segments))
	for _, segment := range segments {
		name, rest, _ := strings.Cut(segment, "[")
		if name == "" {
			return nil, false
		}
		step := pathStep{name: name}
		if rest != "" {
			rest = "[" + rest
			for rest != "" {
				closing := strings.IndexByte(rest, ']')
				if rest[0] != '[' || closing < 2 {
					return nil, false
				}
				step.indices++
				rest = rest[closing+1:]
			}
		}
		steps = append(steps, step)
	}

	return steps, true
}

func reaches(properties map[string]any, steps []pathStep) bool {
	for i, step := range steps {
		property := asSchema(properties[step.name])
		if property == nil {
			return false
		}
		for range step.indices {
			property = asSchema(property["items"])
			if property == nil {
				return false
			}
		}
		if i == len(steps)-1 {
			return true
		}
		properties = schemaProperties(property)
	}

	return false
}

func asSchema(raw any) map[string]any {
	schema, _ := raw.(map[string]any)

	return schema
}

func schemaProperties(schema map[string]any) map[string]any {
	properties, _ := schema["properties"].(map[string]any)

	return properties
}
