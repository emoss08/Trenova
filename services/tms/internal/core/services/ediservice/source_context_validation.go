//nolint:gocritic // Diagnostic parameter structs are passed by value for immutable validation flow.
package ediservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	sourceContextPathUnknownCode    = "source_context_path_unknown"
	sourceContextRootInvalidCode    = "source_context_root_invalid"
	sourceContextRepeatMismatchCode = "source_context_repeat_mismatch"
	sourceContextPathDeprecatedCode = "source_context_path_deprecated"
	sourceContextPathFutureCode     = "source_context_path_future"
	sourceContextSchemaMissingCode  = "source_context_schema_missing"
)

type sourceContextIndex struct {
	fields map[string][]*edi.EDISourceContextField
}

type sourceContextReference struct {
	Path     string
	Field    string
	Segment  *edi.EDITemplateSegment
	Element  *edi.TemplateElement
	Repeated bool
}

type sourceContextDiagnosticParams struct {
	Reference    sourceContextReference
	Severity     edi.ValidationSeverity
	Code         string
	Message      string
	SuggestedFix string
}

func newSourceContextIndex(fields []*edi.EDISourceContextField) *sourceContextIndex {
	index := &sourceContextIndex{
		fields: make(map[string][]*edi.EDISourceContextField, len(fields)),
	}
	for _, field := range fields {
		if field == nil {
			continue
		}
		path := strings.TrimSpace(field.Path)
		if path == "" {
			continue
		}
		index.fields[path] = append(index.fields[path], field)
	}
	return index
}

func validateTemplateSourceContext(
	version *edi.EDITemplateVersion,
	index *sourceContextIndex,
	schemaMissing bool,
) []edix12.Diagnostic {
	if schemaMissing {
		return []edix12.Diagnostic{
			{
				Severity:     edi.ValidationSeverityWarning,
				Code:         sourceContextSchemaMissingCode,
				Path:         "sourceContext",
				Message:      "Source context schema metadata is not available",
				SuggestedFix: "Seed source context metadata to enable source path validation.",
			},
		}
	}
	if version == nil || index == nil {
		return nil
	}

	diagnostics := make([]edix12.Diagnostic, 0)
	for _, segment := range version.Segments {
		if segment == nil {
			continue
		}
		diagnostics = append(
			diagnostics,
			validateSourceContextReferences(index, conditionSourceReferences(segment, nil))...,
		)
		for idx := range segment.Elements {
			element := &segment.Elements[idx]
			references := elementSourceReferences(segment, element)
			diagnostics = append(
				diagnostics,
				validateSourceContextReferences(index, references)...,
			)
		}
	}
	return diagnostics
}

func validateSourceContextReferences(
	index *sourceContextIndex,
	references []sourceContextReference,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	for _, reference := range references {
		path := strings.TrimSpace(reference.Path)
		if path == "" {
			continue
		}

		root := sourceContextRoot(path)
		if root == string(edi.SourceContextKindPartner) {
			continue
		}
		if !isSupportedSourceContextRoot(root) {
			diagnostics = append(diagnostics, sourceContextDiagnostic(sourceContextDiagnosticParams{
				Reference:    reference,
				Severity:     edi.ValidationSeverityError,
				Code:         sourceContextRootInvalidCode,
				Message:      fmt.Sprintf("Source context root %q is not supported", root),
				SuggestedFix: "Use shipment, shipmentStatus, repeat, partner, runtime, or mapping source paths.",
			}))
			continue
		}

		fields := index.fields[path]
		if len(fields) == 0 {
			diagnostics = append(diagnostics, unknownSourceContextDiagnostic(reference, root))
			continue
		}

		matched := false
		for _, field := range fields {
			if field == nil {
				continue
			}
			repeatMismatch := reference.Repeated &&
				field.Repeated &&
				field.RepeatPath != reference.Segment.RepeatPath
			if repeatMismatch {
				continue
			}
			matched = true
			diagnostics = append(diagnostics, sourceContextStatusDiagnostics(reference, field)...)
		}
		if !matched && reference.Repeated {
			message := fmt.Sprintf(
				"Source path %s belongs to a different repeat context",
				reference.Path,
			)
			diagnostics = append(diagnostics, sourceContextDiagnostic(sourceContextDiagnosticParams{
				Reference:    reference,
				Severity:     edi.ValidationSeverityError,
				Code:         sourceContextRepeatMismatchCode,
				Message:      message,
				SuggestedFix: "Use a repeat source path registered for this segment repeat path.",
			}))
		}
	}
	return diagnostics
}

func elementSourceReferences(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
) []sourceContextReference {
	references := make([]sourceContextReference, 0, 4)
	//nolint:exhaustive // Constant and Starlark sources do not reference schema paths.
	switch element.Source {
	case edi.TemplateElementSourceFieldPath:
		references = append(
			references,
			directSourceReference(segment, element, element.FieldPath, "fieldPath"),
		)
	case edi.TemplateElementSourcePartnerSetting:
		references = append(references, directSourceReference(
			segment,
			element,
			"partner."+element.PartnerSettingPath,
			"partnerSettingPath",
		))
	case edi.TemplateElementSourceRuntime:
		references = append(references, directSourceReference(
			segment,
			element,
			"runtime."+element.RuntimeKey,
			"runtimeKey",
		))
	case edi.TemplateElementSourceRepeat:
		references = append(references, directSourceReference(
			segment,
			element,
			"repeat."+element.RepeatPath,
			"repeatPath",
		))
	case edi.TemplateElementSourceMapping:
		references = append(references, directSourceReference(
			segment,
			element,
			"mapping."+element.MappingSourcePath,
			"mappingSourcePath",
		))
	case edi.TemplateElementSourceTransform:
		references = append(references, transformSourceReferences(segment, element)...)
	}
	references = append(references, conditionSourceReferences(segment, element)...)
	return references
}

func transformSourceReferences(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
) []sourceContextReference {
	references := make([]sourceContextReference, 0, 3)
	if element.BaseSource != nil {
		references = append(references, baseSourceReference(segment, element, element.BaseSource))
	}
	for _, step := range element.TransformPipeline {
		stepReferences := transformArgumentReferences(segment, element, step.Arguments)
		references = append(references, stepReferences...)
	}
	return references
}

func transformArgumentReferences(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	args map[string]any,
) []sourceContextReference {
	references := make([]sourceContextReference, 0)
	for _, value := range args {
		valueReferences := transformArgumentValueReferences(segment, element, value)
		references = append(references, valueReferences...)
	}
	return references
}

func transformArgumentValueReferences(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	value any,
) []sourceContextReference {
	switch typed := value.(type) {
	case string:
		if !strings.HasPrefix(typed, "$") {
			return nil
		}
		return []sourceContextReference{{
			Path:     strings.TrimPrefix(typed, "$"),
			Field:    "transformPipeline.arguments",
			Segment:  segment,
			Element:  element,
			Repeated: strings.HasPrefix(strings.TrimPrefix(typed, "$"), "repeat."),
		}}
	case []any:
		references := make([]sourceContextReference, 0, len(typed))
		for _, item := range typed {
			itemReferences := transformArgumentValueReferences(segment, element, item)
			references = append(references, itemReferences...)
		}
		return references
	case map[string]any:
		return transformArgumentReferences(segment, element, typed)
	default:
		return nil
	}
}

func baseSourceReference(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	source *edi.TemplateElementBaseSource,
) sourceContextReference {
	//nolint:exhaustive // Constant, transform, and Starlark cannot be direct base paths here.
	switch source.Source {
	case edi.TemplateElementSourceFieldPath:
		return directSourceReference(segment, element, source.FieldPath, "baseSource.fieldPath")
	case edi.TemplateElementSourcePartnerSetting:
		return directSourceReference(
			segment,
			element,
			"partner."+source.PartnerSettingPath,
			"baseSource.partnerSettingPath",
		)
	case edi.TemplateElementSourceRuntime:
		return directSourceReference(
			segment,
			element,
			"runtime."+source.RuntimeKey,
			"baseSource.runtimeKey",
		)
	case edi.TemplateElementSourceRepeat:
		return directSourceReference(
			segment,
			element,
			"repeat."+source.RepeatPath,
			"baseSource.repeatPath",
		)
	case edi.TemplateElementSourceMapping:
		return directSourceReference(
			segment,
			element,
			"mapping."+source.MappingSourcePath,
			"baseSource.mappingSourcePath",
		)
	default:
		return sourceContextReference{}
	}
}

func directSourceReference(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	path string,
	field string,
) sourceContextReference {
	normalized := normalizeSourceContextPath(path)
	return sourceContextReference{
		Path:     normalized,
		Field:    field,
		Segment:  segment,
		Element:  element,
		Repeated: strings.HasPrefix(normalized, "repeat."),
	}
}

func conditionSourceReferences(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
) []sourceContextReference {
	condition := segment.Condition
	position := 0
	if element != nil {
		condition = element.Condition
		position = element.Position
	}
	paths := edix12.DeclarativeConditionPaths(condition)
	references := make([]sourceContextReference, 0, len(paths))
	for _, path := range paths {
		references = append(references, sourceContextReference{
			Path:  strings.TrimSpace(path),
			Field: "condition",
			Segment: &edi.EDITemplateSegment{
				SegmentID:  segment.SegmentID,
				RepeatPath: segment.RepeatPath,
			},
			Element:  &edi.TemplateElement{Position: position},
			Repeated: strings.HasPrefix(path, "repeat."),
		})
	}
	return references
}

func normalizeSourceContextPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasSuffix(path, ".") {
		return ""
	}
	if isPrefixedSourceContextPath(path) {
		return path
	}
	return "shipment." + path
}

func isPrefixedSourceContextPath(path string) bool {
	root := sourceContextRoot(path)
	switch root {
	case "shipment", "shipmentStatus", "repeat", "partner", "runtime", "mapping":
		return true
	default:
		return false
	}
}

func sourceContextRoot(path string) string {
	root, _, ok := strings.Cut(path, ".")
	if !ok {
		return path
	}
	return root
}

func isSupportedSourceContextRoot(root string) bool {
	switch root {
	case string(edi.SourceContextKindShipment),
		"shipmentStatus",
		string(edi.SourceContextKindRepeat),
		string(edi.SourceContextKindPartner),
		string(edi.SourceContextKindRuntime),
		string(edi.SourceContextKindMapping):
		return true
	default:
		return false
	}
}

func unknownSourceContextDiagnostic(
	reference sourceContextReference,
	root string,
) edix12.Diagnostic {
	severity := edi.ValidationSeverityError
	if root == string(edi.SourceContextKindPartner) {
		severity = edi.ValidationSeverityWarning
	}
	message := fmt.Sprintf(
		"Source path %s is not registered in the source context schema",
		reference.Path,
	)
	return sourceContextDiagnostic(sourceContextDiagnosticParams{
		Reference:    reference,
		Severity:     severity,
		Code:         sourceContextPathUnknownCode,
		Message:      message,
		SuggestedFix: "Choose a registered source path for this template.",
	})
}

func sourceContextStatusDiagnostics(
	reference sourceContextReference,
	field *edi.EDISourceContextField,
) []edix12.Diagnostic {
	switch field.Status {
	case edi.SourceContextFieldStatusActive:
		return nil
	case edi.SourceContextFieldStatusDeprecated:
		return []edix12.Diagnostic{sourceContextDiagnostic(sourceContextDiagnosticParams{
			Reference:    reference,
			Severity:     edi.ValidationSeverityWarning,
			Code:         sourceContextPathDeprecatedCode,
			Message:      fmt.Sprintf("Source path %s is deprecated", reference.Path),
			SuggestedFix: "Use an active replacement source path.",
		})}
	case edi.SourceContextFieldStatusFuture:
		return []edix12.Diagnostic{sourceContextDiagnostic(sourceContextDiagnosticParams{
			Reference:    reference,
			Severity:     edi.ValidationSeverityError,
			Code:         sourceContextPathFutureCode,
			Message:      fmt.Sprintf("Source path %s is reserved for future use", reference.Path),
			SuggestedFix: "Use an active source path for live outbound templates.",
		})}
	default:
		return nil
	}
}

func sourceContextDiagnostic(params sourceContextDiagnosticParams) edix12.Diagnostic {
	position := 0
	if params.Reference.Element != nil {
		position = params.Reference.Element.Position
	}
	segmentID := ""
	if params.Reference.Segment != nil {
		segmentID = params.Reference.Segment.SegmentID
	}
	return edix12.Diagnostic{
		Severity:        params.Severity,
		Code:            params.Code,
		SegmentID:       segmentID,
		ElementPosition: position,
		Path:            stringutils.FirstNonEmpty(params.Reference.Field, params.Reference.Path),
		Message:         params.Message,
		SuggestedFix:    params.SuggestedFix,
	}
}
