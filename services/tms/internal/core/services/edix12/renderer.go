package edix12

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edistarlark"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

type RenderInput struct {
	Context         context.Context
	Profile         *edi.EDIPartnerDocumentProfile
	TemplateVersion *edi.EDITemplateVersion
	Payload         edi.LoadTenderPayload
	DocumentPayload edi.DocumentPayload
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

func RenderX12(input *RenderInput) (*RenderResult, error) {
	renderCtx := input.Context
	if renderCtx == nil {
		renderCtx = context.Background()
	}

	payload := input.DocumentPayload
	if !payload.HasBranch() && input.Payload.ShipmentID.IsNotNil() {
		payload = edi.NewLoadTenderDocumentPayload(input.Payload)
	}
	payload.Normalize()
	payloadMap, err := jsonutils.ToJSON(payload)
	if err != nil {
		return nil, err
	}
	segments := append([]*edi.EDITemplateSegment{}, input.TemplateVersion.Segments...)
	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].Sequence < segments[j].Sequence
	})
	libraries := templateScriptLibraries(input.TemplateVersion.ScriptLibraries)

	rendered := make([]string, 0, len(segments)+8)
	diagnostics := make([]Diagnostic, 0)
	for _, segment := range segments {
		repeats := repeatValues(payloadMap, segment.RepeatPath)
		if len(repeats) == 0 {
			repeats = []any{nil}
		}
		for _, repeatValue := range repeats {
			env := renderEnvironment(payloadMap, input.Profile.PartnerSettings, input.Runtime, repeatValue)
			include, conditionDiagnostic := evaluateCondition(conditionEvalParams{
				Context:   renderCtx,
				Condition: segment.Condition,
				Env:       env,
				Segment:   segment,
				Libraries: libraries,
			})
			if conditionDiagnostic != nil {
				diagnostics = append(diagnostics, *conditionDiagnostic)
				continue
			}
			if !include {
				continue
			}
			elements := make([]string, 0, len(segment.Elements))
			segmentHasValue := segment.Required
			for i := range segment.Elements {
				element := &segment.Elements[i]
				value, elementDiagnostics := resolveElement(elementResolveParams{
					Context:   renderCtx,
					Segment:   segment,
					Element:   element,
					Env:       env,
					Libraries: libraries,
				})
				diagnostics = append(diagnostics, elementDiagnostics...)
				if value != "" {
					segmentHasValue = true
				}
				elements = append(elements, sanitizeX12Value(value, &input.Profile.Envelope))
			}
			if !segmentHasValue && !segment.Required {
				continue
			}
			rendered = append(rendered, strings.Join(
				append([]string{segment.SegmentID}, trimTrailingEmpty(elements)...),
				input.Profile.Envelope.ElementSeparator,
			))
		}
	}

	applyTrailerCounts(rendered, input.Profile.Envelope.ElementSeparator)
	raw := strings.Join(
		rendered,
		input.Profile.Envelope.SegmentTerminator,
	) + input.Profile.Envelope.SegmentTerminator
	return &RenderResult{
		RawX12:       raw,
		SegmentCount: int64(len(rendered)),
		Diagnostics:  filterDiagnostics(diagnostics, input.Profile.ValidationMode),
	}, nil
}

func Render204(input *RenderInput) (*RenderResult, error) {
	return RenderX12(input)
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
		"functionalGroupId": stringutils.FirstNonEmpty(
			profile.FunctionalGroupID,
			edi.FunctionalGroupDefault(profile.TransactionSet),
		),
		"x12Version":      x12Version,
		"interchangeDate": now.Format("060102"),
		"interchangeTime": now.Format("1504"),
		"groupDate":       now.Format("20060102"),
		"groupTime":       now.Format("1504"),
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

type elementResolveParams struct {
	Context   context.Context
	Segment   *edi.EDITemplateSegment
	Element   *edi.TemplateElement
	Env       map[string]any
	Libraries []edistarlark.ScriptLibrary
}

func resolveElement(params elementResolveParams) (string, []Diagnostic) {
	diagnostics := []Diagnostic{}
	include, conditionDiagnostic := evaluateCondition(conditionEvalParams{
		Context:   params.Context,
		Condition: params.Element.Condition,
		Env:       params.Env,
		Segment:   params.Segment,
		Element:   params.Element,
		Libraries: params.Libraries,
	})
	if conditionDiagnostic != nil {
		return "", []Diagnostic{*conditionDiagnostic}
	}
	if !include {
		return "", diagnostics
	}

	rawValue, sourceDiagnostics, valueErr := resolveElementValue(params)
	if len(sourceDiagnostics) > 0 {
		return "", sourceDiagnostics
	}

	value := formatElementValue(params.Segment, params.Element, rawValue)
	if valueErr != nil {
		diagnostics = append(
			diagnostics,
			renderDiagnostic(renderDiagnosticParams{
				Segment:      params.Segment,
				Position:     params.Element.Position,
				Path:         sourcePath(params.Element),
				Message:      valueErr.Error(),
				SuggestedFix: suggestedFixForSource(params.Element.Source),
			}),
		)
	}
	if value == "" {
		value = params.Element.Default
	}
	if params.Element.Validation.Required && strings.TrimSpace(value) == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Severity:        edi.ValidationSeverityError,
			Code:            stringutils.FirstNonEmpty(params.Element.Validation.Code, "required"),
			SegmentID:       params.Segment.SegmentID,
			ElementPosition: params.Element.Position,
			Path:            sourcePath(params.Element),
			Message: stringutils.FirstNonEmpty(
				params.Element.Validation.Message,
				params.Element.Name+" is required",
			),
		})
	}
	if params.Element.Validation.MaxLength > 0 && len(value) > params.Element.Validation.MaxLength {
		diagnostics = append(diagnostics, Diagnostic{
			Severity:        edi.ValidationSeverityWarning,
			Code:            "max_length",
			SegmentID:       params.Segment.SegmentID,
			ElementPosition: params.Element.Position,
			Message: fmt.Sprintf(
				"%s exceeds max length %d",
				params.Element.Name,
				params.Element.Validation.MaxLength,
			),
		})
		value = value[:params.Element.Validation.MaxLength]
	}
	return value, diagnostics
}

func resolveElementValue(params elementResolveParams) (any, []Diagnostic, error) {
	element := params.Element
	if isDirectElementSource(element.Source) {
		value, _ := resolveDirectSource(elementDirectSource(element), params.Env)
		return value, nil, nil
	}

	switch element.Source {
	case edi.TemplateElementSourceTransform:
		return resolveTransformElementValue(params.Segment, element, params.Env)
	case edi.TemplateElementSourceStarlark:
		value, diagnostics := resolveStarlarkElementValue(params)
		return value, diagnostics, nil
	default:
		return "", nil, nil
	}
}

func isDirectElementSource(source edi.TemplateElementSource) bool {
	switch source {
	case edi.TemplateElementSourceConstant,
		edi.TemplateElementSourceFieldPath,
		edi.TemplateElementSourcePartnerSetting,
		edi.TemplateElementSourceRuntime,
		edi.TemplateElementSourceRepeat,
		edi.TemplateElementSourceMapping:
		return true
	default:
		return false
	}
}

type directSource struct {
	Source             edi.TemplateElementSource
	Value              string
	FieldPath          string
	PartnerSettingPath string
	MappingSourcePath  string
	RuntimeKey         string
	RepeatPath         string
	Name               string
}

func elementDirectSource(element *edi.TemplateElement) directSource {
	return directSource{
		Source:             element.Source,
		Value:              element.Value,
		FieldPath:          element.FieldPath,
		PartnerSettingPath: element.PartnerSettingPath,
		MappingSourcePath:  element.MappingSourcePath,
		RuntimeKey:         element.RuntimeKey,
		RepeatPath:         element.RepeatPath,
		Name:               element.Name,
	}
}

func baseDirectSource(source *edi.TemplateElementBaseSource) directSource {
	return directSource{
		Source:             source.Source,
		Value:              source.Value,
		FieldPath:          source.FieldPath,
		PartnerSettingPath: source.PartnerSettingPath,
		MappingSourcePath:  source.MappingSourcePath,
		RuntimeKey:         source.RuntimeKey,
		RepeatPath:         source.RepeatPath,
	}
}

func resolveDirectSource(source directSource, env map[string]any) (any, bool) {
	switch source.Source {
	case edi.TemplateElementSourceConstant:
		return source.Value, true
	case edi.TemplateElementSourceFieldPath:
		return maputils.Path(env, qualifyFieldPath(source.FieldPath)), true
	case edi.TemplateElementSourcePartnerSetting:
		path := stringutils.FirstNonEmpty(source.PartnerSettingPath, source.Name)
		if strings.TrimSpace(path) == "" {
			return nil, true
		}
		return maputils.Path(env, "partner."+path), true
	case edi.TemplateElementSourceRuntime:
		return maputils.Path(env, "runtime."+source.RuntimeKey), true
	case edi.TemplateElementSourceRepeat:
		return maputils.Path(env, "repeat."+source.RepeatPath), true
	case edi.TemplateElementSourceMapping:
		return maputils.Path(env, "mapping."+source.MappingSourcePath), true
	default:
		return nil, false
	}
}

func qualifyFieldPath(path string) string {
	if isQualifiedFieldRoot(path) {
		return path
	}
	return "shipment." + path
}

func isQualifiedFieldRoot(path string) bool {
	roots := [...]string{
		"shipment.",
		"loadTender.",
		"invoice.",
		"shipmentStatus.",
		"tenderResponse.",
		"functionalAck.",
		"implementationAck.",
		"repeat.",
		"runtime.",
		"partner.",
		"mapping.",
	}
	for _, root := range roots {
		if strings.HasPrefix(path, root) {
			return true
		}
	}
	return false
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

func renderEnvironment(
	payload map[string]any,
	partner map[string]any,
	runtime map[string]any,
	repeat any,
) map[string]any {
	if partner == nil {
		partner = map[string]any{}
	}
	env := make(map[string]any, len(payload)+11)
	for key, value := range payload {
		env[key] = value
	}
	ensureDocumentRootMaps(env)
	env["partner"] = partner
	env["mapping"] = map[string]any{}
	env["runtime"] = runtime
	env["repeat"] = repeat
	return env
}

func ensureDocumentRootMaps(env map[string]any) {
	for _, root := range documentRootKeys {
		if _, ok := env[root].(map[string]any); !ok {
			env[root] = map[string]any{}
		}
	}
}

var documentRootKeys = [...]string{
	"shipment",
	"loadTender",
	"invoice",
	"shipmentStatus",
	"tenderResponse",
	"functionalAck",
	"implementationAck",
}

func resolveStarlarkElementValue(params elementResolveParams) (string, []Diagnostic) {
	starlarkCtx := starlarkContext(params.Env)

	repeatValue := params.Env["repeat"]
	if repeatValue != nil {
		starlarkCtx["repeat"] = repeatValue
		starlarkCtx["item"] = repeatValue
	}

	result := edistarlark.Evaluate(params.Context, edistarlark.EvalRequest{
		Script:          params.Element.StarlarkScript,
		FunctionName:    params.Element.StarlarkFunction,
		Libraries:       params.Libraries,
		Context:         starlarkCtx,
		Item:            repeatValue,
		SegmentID:       params.Segment.SegmentID,
		ElementPosition: params.Element.Position,
		Path:            sourcePath(params.Element),
	})
	if len(result.Diagnostics) > 0 {
		return "", starlarkDiagnostics(result.Diagnostics)
	}
	return result.Value, nil
}

func starlarkContext(env map[string]any) map[string]any {
	ctx := make(map[string]any, len(env))
	for key, value := range env {
		ctx[key] = value
	}
	ensureDocumentRootMaps(ctx)
	if _, ok := ctx["partner"].(map[string]any); !ok {
		ctx["partner"] = map[string]any{}
	}
	if _, ok := ctx["runtime"].(map[string]any); !ok {
		ctx["runtime"] = map[string]any{}
	}
	if _, ok := ctx["mapping"].(map[string]any); !ok {
		ctx["mapping"] = map[string]any{}
	}
	return ctx
}

func templateScriptLibraries(
	source []*edi.EDITemplateScriptLibrary,
) []edistarlark.ScriptLibrary {
	libraries := make([]edistarlark.ScriptLibrary, 0, len(source))
	for _, library := range source {
		if library == nil {
			continue
		}
		libraries = append(libraries, edistarlark.ScriptLibrary{
			Name:   library.Name,
			Script: library.Script,
		})
	}
	return libraries
}

func starlarkDiagnostics(diagnostics []edistarlark.Diagnostic) []Diagnostic {
	converted := make([]Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		converted = append(converted, Diagnostic{
			Severity:        edi.ValidationSeverity(diagnostic.Severity),
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
			SuggestedFix:    diagnostic.SuggestedFix,
		})
	}
	return converted
}

func repeatValues(payload map[string]any, path string) []any {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	items, ok := maputils.Path(payload, qualifyFieldPath(path)).([]any)
	if !ok {
		return nil
	}
	return items
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
			if diagnostic.Code == "render_error" ||
				strings.HasPrefix(diagnostic.Code, "starlark_") ||
				strings.HasPrefix(diagnostic.Code, "transform_") ||
				strings.HasPrefix(diagnostic.Code, "condition_") ||
				strings.HasPrefix(diagnostic.Code, "script_") {
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

func sourcePath(element *edi.TemplateElement) string {
	if element.Source == edi.TemplateElementSourceStarlark {
		functionName := strings.TrimSpace(element.StarlarkFunction)
		if functionName == "" {
			return "starlark:value"
		}
		return "starlark:" + functionName
	}

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
		return transformSuggestedFix
	case edi.TemplateElementSourceStarlark:
		return "Check the Starlark script, function name, helper arguments, and available context fields."
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
