package edix12

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

type RenderInput struct {
	Profile         *edi.EDIPartnerDocumentProfile
	TemplateVersion *edi.EDITemplateVersion
	Payload         edi.LoadTenderPayload
	X12Version      string
	Runtime         map[string]any
}

type RenderResult struct {
	RawX12       string
	SegmentCount int64
	Diagnostics  []Diagnostic
}

type Diagnostic struct {
	Severity        edi.ValidationSeverity `json:"severity"`
	Code            string                 `json:"code"`
	SegmentID       string                 `json:"segmentId"`
	ElementPosition int                    `json:"elementPosition"`
	Path            string                 `json:"path"`
	Message         string                 `json:"message"`
	SuggestedFix    string                 `json:"suggestedFix"`
}

func Render204(ctx *RenderInput) (*RenderResult, error) {
	payloadMap, err := jsonutils.ToJSON(ctx.Payload)
	if err != nil {
		return nil, err
	}
	segments := append([]*edi.EDITemplateSegment{}, ctx.TemplateVersion.Segments...)
	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].Sequence < segments[j].Sequence
	})

	rendered := make([]string, 0, len(segments)+8)
	diagnostics := make([]Diagnostic, 0)
	for _, segment := range segments {
		repeats := repeatValues(payloadMap, segment.RepeatPath)
		if len(repeats) == 0 {
			repeats = []any{nil}
		}
		for _, repeatValue := range repeats {
			renderCtx := renderContext(
				payloadMap,
				ctx.Profile.PartnerSettings,
				ctx.Runtime,
				repeatValue,
			)
			include, evalErr := evaluateCondition(segment.Condition)
			if evalErr != nil {
				diagnostics = append(
					diagnostics,
					unsupportedConditionDiagnostic(segment, 0, segment.Condition),
				)
				continue
			}
			if !include {
				continue
			}
			elements := make([]string, 0, len(segment.Elements))
			segmentHasValue := segment.Required
			for i := range segment.Elements {
				element := &segment.Elements[i]
				value, elementDiagnostics := resolveElement(segment, element, renderCtx)
				diagnostics = append(diagnostics, elementDiagnostics...)
				if value != "" {
					segmentHasValue = true
				}
				elements = append(elements, sanitizeX12Value(value, &ctx.Profile.Envelope))
			}
			if !segmentHasValue && !segment.Required {
				continue
			}
			rendered = append(rendered, strings.Join(
				append([]string{segment.SegmentID}, trimTrailingEmpty(elements)...),
				ctx.Profile.Envelope.ElementSeparator,
			))
		}
	}

	applyTrailerCounts(rendered, ctx.Profile.Envelope.ElementSeparator)
	raw := strings.Join(
		rendered,
		ctx.Profile.Envelope.SegmentTerminator,
	) + ctx.Profile.Envelope.SegmentTerminator
	return &RenderResult{
		RawX12:       raw,
		SegmentCount: int64(len(rendered)),
		Diagnostics:  filterDiagnostics(diagnostics, ctx.Profile.ValidationMode),
	}, nil
}

func RuntimeValues(profile *edi.EDIPartnerDocumentProfile, x12Version string) map[string]any {
	now := time.Now().UTC()
	envelope := profile.Envelope
	return map[string]any{
		"interchangeSenderId":   padISAID(envelope.InterchangeSenderID),
		"interchangeReceiverId": padISAID(envelope.InterchangeReceiverID),
		"applicationSenderCode": stringutils.FirstNonEmpty(
			envelope.ApplicationSenderCode,
			envelope.InterchangeSenderID,
		),
		"applicationReceiverCode": stringutils.FirstNonEmpty(
			envelope.ApplicationReceiverCode,
			envelope.InterchangeReceiverID,
		),
		"usageIndicator": stringutils.FirstNonEmpty(
			envelope.InterchangeUsageIndicator,
			"T",
		),
		"componentSeparator":  stringutils.FirstNonEmpty(envelope.ComponentSeparator, ">"),
		"repetitionSeparator": stringutils.FirstNonEmpty(envelope.RepetitionSeparator, "^"),
		"functionalGroupId":   stringutils.FirstNonEmpty(profile.FunctionalGroupID, "SM"),
		"x12Version":          x12Version,
		"interchangeDate":     now.Format("060102"),
		"interchangeTime":     now.Format("1504"),
		"groupDate":           now.Format("20060102"),
		"groupTime":           now.Format("1504"),
	}
}

func SetProvisionalControlNumbers(runtime map[string]any) {
	runtime["isaControlNumber"] = "000000000"
	runtime["groupControlNumber"] = "0"
	runtime["transactionControlNumber"] = "0000"
}

func HasBlockingDiagnostics(diagnostics []Diagnostic, mode edi.ValidationMode) bool {
	if mode != edi.ValidationModeStrict {
		return false
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == edi.ValidationSeverityError {
			return true
		}
	}
	return false
}

func resolveElement(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	env map[string]any,
) (string, []Diagnostic) {
	diagnostics := []Diagnostic{}
	include, err := evaluateCondition(element.Condition)
	if err != nil {
		return "", []Diagnostic{
			unsupportedConditionDiagnostic(segment, element.Position, element.Condition),
		}
	}
	if !include {
		return "", diagnostics
	}

	rawValue, valueErr := resolveElementValue(element, env)
	value := formatElementValue(segment, element, rawValue)
	if valueErr != nil {
		diagnostics = append(
			diagnostics,
			renderDiagnostic(renderDiagnosticParams{
				Segment:      segment,
				Position:     element.Position,
				Path:         sourcePath(element),
				Message:      valueErr.Error(),
				SuggestedFix: suggestedFixForSource(element.Source),
			}),
		)
	}
	if value == "" {
		value = element.Default
	}
	if element.Validation.Required && strings.TrimSpace(value) == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Severity:        edi.ValidationSeverityError,
			Code:            stringutils.FirstNonEmpty(element.Validation.Code, "required"),
			SegmentID:       segment.SegmentID,
			ElementPosition: element.Position,
			Path:            sourcePath(element),
			Message: stringutils.FirstNonEmpty(
				element.Validation.Message,
				element.Name+" is required",
			),
		})
	}
	if element.Validation.MaxLength > 0 && len(value) > element.Validation.MaxLength {
		diagnostics = append(diagnostics, Diagnostic{
			Severity:        edi.ValidationSeverityWarning,
			Code:            "max_length",
			SegmentID:       segment.SegmentID,
			ElementPosition: element.Position,
			Message: fmt.Sprintf(
				"%s exceeds max length %d",
				element.Name,
				element.Validation.MaxLength,
			),
		})
		value = value[:element.Validation.MaxLength]
	}
	return value, diagnostics
}

func resolveElementValue(element *edi.TemplateElement, env map[string]any) (any, error) {
	switch element.Source {
	case edi.TemplateElementSourceConstant:
		return element.Value, nil
	case edi.TemplateElementSourceFieldPath:
		return maputils.Path(env, qualifyFieldPath(element.FieldPath)), nil
	case edi.TemplateElementSourcePartnerSetting:
		path := stringutils.FirstNonEmpty(element.PartnerSettingPath, element.Name)
		return maputils.Path(env, "partner."+path), nil
	case edi.TemplateElementSourceRuntime:
		return maputils.Path(env, "runtime."+element.RuntimeKey), nil
	case edi.TemplateElementSourceRepeat:
		return maputils.Path(env, "repeat."+element.RepeatPath), nil
	case edi.TemplateElementSourceMapping:
		return maputils.Path(env, "mapping."+element.MappingSourcePath), nil
	case edi.TemplateElementSourceTransform, edi.TemplateElementSourceStarlark:
		return "", fmt.Errorf("%s source rendering is not supported yet", element.Source)
	default:
		return "", nil
	}
}

func qualifyFieldPath(path string) string {
	if strings.HasPrefix(path, "repeat.") || strings.HasPrefix(path, "runtime.") ||
		strings.HasPrefix(path, "partner.") || strings.HasPrefix(path, "mapping.") {
		return path
	}
	return "shipment." + path
}

func formatElementValue(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	value any,
) string {
	if segment.SegmentID == "G62" {
		switch element.Position {
		case 2:
			return formatX12Date(value)
		case 4:
			return formatX12Time(value)
		}
	}
	return valueToString(value)
}

func formatX12Date(value any) string {
	timestamp, ok := unixTimestamp(value)
	if !ok {
		return valueToString(value)
	}
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).UTC().Format("20060102")
}

func formatX12Time(value any) string {
	timestamp, ok := unixTimestamp(value)
	if !ok {
		return valueToString(value)
	}
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).UTC().Format("1504")
}

func unixTimestamp(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case float32:
		return int64(typed), true
	default:
		return 0, false
	}
}

func applyTrailerCounts(rendered []string, separator string) {
	stIndex := -1
	transactionCount := 0
	controlNumber := ""
	for i, segment := range rendered {
		parts := strings.Split(segment, separator)
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "ST":
			stIndex = i
			if len(parts) > 2 {
				controlNumber = parts[2]
			}
		case "SE":
			if stIndex >= 0 {
				transactionCount = i - stIndex + 1
			}
			if len(parts) > 1 {
				parts[1] = strconv.Itoa(transactionCount)
			}
			if len(parts) > 2 && controlNumber != "" {
				parts[2] = controlNumber
			}
			rendered[i] = strings.Join(parts, separator)
		}
	}
}

func renderContext(
	shipment map[string]any,
	partner map[string]any,
	runtime map[string]any,
	repeat any,
) map[string]any {
	if partner == nil {
		partner = map[string]any{}
	}
	return map[string]any{
		"shipment": shipment,
		"partner":  partner,
		"mapping":  map[string]any{},
		"runtime":  runtime,
		"repeat":   repeat,
	}
}

func repeatValues(payload map[string]any, path string) []any {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	items, ok := maputils.Path(map[string]any{"shipment": payload}, "shipment."+path).([]any)
	if !ok {
		return nil
	}
	return items
}

func evaluateCondition(condition string) (bool, error) {
	if strings.TrimSpace(condition) == "" {
		return true, nil
	}
	return false, fmt.Errorf("template conditions are not supported until Starlark condition rendering is available")
}

func valueToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case decimal.NullDecimal:
		if !typed.Valid {
			return ""
		}
		return typed.Decimal.StringFixed(2)
	case decimal.Decimal:
		return typed.StringFixed(2)
	case fmt.Stringer:
		return typed.String()
	case float64:
		return trimFloat(typed)
	case float32:
		return trimFloat(float64(typed))
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		if typed {
			return "Y"
		}
		return "N"
	case map[string]any:
		if valid, ok := typed["Valid"].(bool); ok && !valid {
			return ""
		}
		if decimalValue, ok := typed["Decimal"]; ok {
			return valueToString(decimalValue)
		}
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func trimFloat(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func sanitizeX12Value(value string, envelope *edi.X12EnvelopeSettings) string {
	replacer := strings.NewReplacer(
		envelope.ElementSeparator, " ",
		envelope.SegmentTerminator, " ",
		envelope.ComponentSeparator, " ",
	)
	return strings.TrimSpace(replacer.Replace(value))
}

func trimTrailingEmpty(values []string) []string {
	last := len(values)
	for last > 0 && values[last-1] == "" {
		last--
	}
	return values[:last]
}

func filterDiagnostics(diagnostics []Diagnostic, mode edi.ValidationMode) []Diagnostic {
	if mode == edi.ValidationModeDisabled {
		filtered := make([]Diagnostic, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "render_error" {
				filtered = append(filtered, diagnostic)
			}
		}
		return filtered
	}
	if mode == edi.ValidationModeWarnOnly {
		for i := range diagnostics {
			if diagnostics[i].Severity == edi.ValidationSeverityError {
				diagnostics[i].Severity = edi.ValidationSeverityWarning
			}
		}
	}
	return diagnostics
}

type renderDiagnosticParams struct {
	Segment      *edi.EDITemplateSegment
	Position     int
	Path         string
	Message      string
	SuggestedFix string
}

func renderDiagnostic(params renderDiagnosticParams) Diagnostic {
	return Diagnostic{
		Severity:        edi.ValidationSeverityError,
		Code:            "render_error",
		SegmentID:       params.Segment.SegmentID,
		ElementPosition: params.Position,
		Path:            params.Path,
		Message:         params.Message,
		SuggestedFix:    params.SuggestedFix,
	}
}

func unsupportedConditionDiagnostic(
	segment *edi.EDITemplateSegment,
	position int,
	condition string,
) Diagnostic {
	return renderDiagnostic(renderDiagnosticParams{
		Segment:  segment,
		Position: position,
		Path:     condition,
		Message:  "Template conditions are not supported until Starlark condition rendering is available",
		SuggestedFix: "Remove this condition for now, or wait for Starlark condition support " +
			"before using conditional EDI rendering.",
	})
}

func sourcePath(element *edi.TemplateElement) string {
	path := stringutils.FirstNonEmpty(
		element.FieldPath,
		element.RepeatPath,
		element.RuntimeKey,
		element.MappingSourcePath,
		element.PartnerSettingPath,
		element.StarlarkFunction,
		element.StarlarkScript,
		element.Value,
		element.Default,
	)
	if path != "" || element.BaseSource == nil {
		return path
	}
	return baseSourcePath(element.BaseSource)
}

func baseSourcePath(source *edi.TemplateElementBaseSource) string {
	return stringutils.FirstNonEmpty(
		source.FieldPath,
		source.RepeatPath,
		source.RuntimeKey,
		source.MappingSourcePath,
		source.PartnerSettingPath,
		source.Value,
		source.Default,
	)
}

func suggestedFixForSource(source edi.TemplateElementSource) string {
	switch source {
	case edi.TemplateElementSourceTransform:
		return "Use a direct source until transform pipeline rendering is implemented."
	case edi.TemplateElementSourceStarlark:
		return "Use a direct source until the restricted Starlark runtime is implemented."
	default:
		return ""
	}
}

func padISAID(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) > 15 {
		return value[:15]
	}
	return value + strings.Repeat(" ", 15-len(value))
}
