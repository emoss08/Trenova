package config

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// ConfigurableBuilder builds transactions based on configuration
type ConfigurableBuilder struct {
	config    *TransactionConfig
	customer  *CustomerConfig
	registry  *segments.SegmentRegistry
	processor *segments.SegmentProcessor
	delims    x12.Delimiters
}

// NewConfigurableBuilder creates a new configurable builder
func NewConfigurableBuilder(
	config *TransactionConfig,
	customer *CustomerConfig,
	registry *segments.SegmentRegistry,
	delims x12.Delimiters,
) *ConfigurableBuilder {
	processor := segments.NewSegmentProcessor(registry)

	if customer != nil && customer.Active {
		processor.SetCustomerRequirements(&segments.CustomerRequirements{
			PartnerID:       customer.CustomerID,
			Version:         config.Version,
			TransactionType: config.TransactionType,
		})
	}

	return &ConfigurableBuilder{
		config:    config,
		customer:  customer,
		registry:  registry,
		processor: processor,
		delims:    delims,
	}
}

// BuildFromObject builds EDI from a business object using configuration
func (b *ConfigurableBuilder) BuildFromObject(ctx context.Context, data any) (string, error) {
	var result strings.Builder
	segmentCount := 0

	stSegment := b.buildSTSegment()
	result.WriteString(stSegment)
	result.WriteByte(b.delims.Segment)
	segmentCount++

	for _, req := range b.config.Structure.RequiredSegments {
		if req.SegmentID == "ST" || req.SegmentID == "SE" {
			continue
		}

		segment, err := b.buildSegmentFromMapping(req.SegmentID, data)
		if err != nil {
			if req.Required {
				return "", fmt.Errorf("failed to build required segment %s: %w", req.SegmentID, err)
			}
			continue
		}

		result.WriteString(segment)
		result.WriteByte(b.delims.Segment)
		segmentCount++
	}

	for _, loop := range b.config.Structure.Loops {
		loopSegments, count, err := b.buildLoop(loop, data)
		if err != nil {
			if loop.MinOccurs > 0 {
				return "", fmt.Errorf("failed to build required loop %s: %w", loop.LoopID, err)
			}
			continue
		}

		result.WriteString(loopSegments)
		segmentCount += count
	}

	for _, cond := range b.config.Structure.ConditionalSegments {
		if b.evaluateCondition(cond.Condition, data) {
			segment, err := b.buildSegmentFromMapping(cond.SegmentID, data)
			if err != nil {
				if cond.Required {
					return "", fmt.Errorf(
						"failed to build conditional segment %s: %w",
						cond.SegmentID,
						err,
					)
				}
				continue
			}

			result.WriteString(segment)
			result.WriteByte(b.delims.Segment)
			segmentCount++
		}
	}

	seSegment := b.buildSESegment(segmentCount + 1)
	result.WriteString(seSegment)
	result.WriteByte(b.delims.Segment)

	return result.String(), nil
}

// ParseToObject parses EDI segments into a business object using configuration
func (b *ConfigurableBuilder) ParseToObject(
	ctx context.Context,
	segments []x12.Segment,
) (any, error) {
	processed, err := b.processor.ProcessSegments(ctx, segments, b.config.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to process segments: %w", err)
	}

	result := make(map[string]any)

	segmentCounts := make(map[string]int)
	rawSegments := make([]map[string]any, 0, len(processed))

	for _, seg := range processed {
		segmentCounts[seg.Schema.ID]++

		segData := make(map[string]any)
		segData["_tag"] = seg.Schema.ID
		segData["_position"] = seg.Position.Index

		maps.Copy(segData, seg.Data)

		rawSegments = append(rawSegments, segData)

		for _, mapping := range b.config.Mappings {
			if mapping.SegmentID == seg.Schema.ID &&
				(mapping.Direction == "inbound" || mapping.Direction == "both") {
				if err := b.applyMapping(result, seg, mapping); err != nil {
					return nil, fmt.Errorf("failed to apply mapping for %s: %w", seg.Schema.ID, err)
				}
			}
		}
	}

	result["_segments"] = rawSegments
	result["_segment_counts"] = segmentCounts

	return result, nil
}

// Validate validates data against configuration rules
func (b *ConfigurableBuilder) Validate(ctx context.Context, data any) []validation.Issue {
	var issues []validation.Issue

	for _, rule := range b.config.ValidationRules {
		// ! For validation rules, the condition defines what violates the rule
		// ! If the condition is true, then the rule is violated
		if b.evaluateValidationCondition(rule.Condition, data) {
			issues = append(issues, validation.Issue{
				Severity: b.getSeverity(rule.Severity),
				Code:     rule.ErrorCode,
				Message:  rule.Message,
				Level:    "business",
			})
		}
	}

	if b.customer != nil {
		for _, rule := range b.customer.AdditionalRules {
			if b.evaluateValidationCondition(rule.Condition, data) {
				issues = append(issues, validation.Issue{
					Severity: b.getSeverity(rule.Severity),
					Code:     rule.ErrorCode,
					Message:  rule.Message,
					Level:    "customer",
				})
			}
		}
	}

	return issues
}

// buildSegmentFromMapping builds a segment using mapping configuration
func (b *ConfigurableBuilder) buildSegmentFromMapping(segmentID string, data any) (string, error) {
	mapping := b.findSegmentMapping(segmentID)
	if mapping == nil {
		return "", fmt.Errorf("no mapping found for segment %s", segmentID)
	}

	elements := []string{segmentID}
	maxPos := 0
	elementValues := make(map[int]string)

	for _, elem := range mapping.Elements {
		value := b.getValueFromPath(data, mapping.ObjectPath, elem.ObjectField)

		if elem.Transform != "" {
			value = b.applyTransform(value, elem.Transform)
		}

		value = b.applyDefaults(value, elem, segmentID)

		if elem.Required && value == "" {
			return "", fmt.Errorf(
				"element %d is required but empty for segment %s",
				elem.ElementPosition,
				segmentID,
			)
		}

		if value != "" && elem.Validation != "" {
			if err := b.validateElementPattern(value, elem.Validation, elem.ElementPosition); err != nil {
				return "", fmt.Errorf("segment %s: %w", segmentID, err)
			}
		}

		elementValues[elem.ElementPosition] = value
		if elem.ElementPosition > maxPos {
			maxPos = elem.ElementPosition
		}
	}

	for i := 1; i <= maxPos; i++ {
		elements = append(elements, elementValues[i])
	}

	return strings.Join(elements, string(b.delims.Element)), nil
}

// findSegmentMapping finds the mapping for a segment ID
func (b *ConfigurableBuilder) findSegmentMapping(segmentID string) *SegmentMapping {
	for _, m := range b.config.Mappings {
		if m.SegmentID == segmentID && (m.Direction == "outbound" || m.Direction == "both") {
			return &m
		}
	}

	if b.customer != nil {
		for _, m := range b.customer.CustomMappings {
			if m.SegmentID == segmentID && (m.Direction == "outbound" || m.Direction == "both") {
				return &m
			}
		}
	}

	return nil
}

// applyDefaults applies default values to an element
func (b *ConfigurableBuilder) applyDefaults(
	value string,
	elem ElementMapping,
	segmentID string,
) string {
	if value == "" && elem.DefaultValue != nil {
		value = fmt.Sprintf("%v", elem.DefaultValue)
	}

	if value == "" && b.customer != nil {
		if defaults, ok := b.customer.DefaultValues[segmentID]; ok {
			if defVal, ok := defaults[elem.ElementPosition]; ok {
				value = fmt.Sprintf("%v", defVal)
			}
		}
	}

	return value
}

// validateElementPattern validates an element value against a pattern
func (b *ConfigurableBuilder) validateElementPattern(value, pattern string, position int) error {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return fmt.Errorf("invalid validation pattern for element %d: %w", position, err)
	}
	if !matched {
		return fmt.Errorf(
			"element %d failed validation: %s does not match pattern %s",
			position,
			value,
			pattern,
		)
	}
	return nil
}

// buildLoop builds a loop from data
func (b *ConfigurableBuilder) buildLoop(loop LoopDefinition, data any) (string, int, error) {
	var result strings.Builder
	count := 0

	loopData := b.getLoopData(data, loop.StartSegment)
	if loopData == nil {
		if loop.MinOccurs > 0 {
			return "", 0, fmt.Errorf("required loop %s not found in data", loop.LoopID)
		}
		return "", 0, nil
	}

	loopItems := b.getArrayItems(loopData)
	if err := b.validateLoopOccurrences(loop, len(loopItems)); err != nil {
		return "", 0, err
	}

	for _, item := range loopItems {
		for _, seg := range loop.Segments {
			segment, err := b.buildSegmentFromMapping(seg.SegmentID, item)
			if err != nil {
				if seg.Required {
					return "", 0, fmt.Errorf("failed to build required segment %s in loop %s: %w",
						seg.SegmentID, loop.LoopID, err)
				}
				continue
			}

			result.WriteString(segment)
			result.WriteByte(b.delims.Segment)
			count++
		}

		for _, nested := range loop.NestedLoops {
			nestedSegments, nestedCount, err := b.buildLoop(nested, item)
			if err != nil {
				if nested.MinOccurs > 0 {
					return "", 0, fmt.Errorf(
						"failed to build nested loop %s: %w",
						nested.LoopID,
						err,
					)
				}
				continue
			}
			result.WriteString(nestedSegments)
			count += nestedCount
		}
	}

	return result.String(), count, nil
}

// validateLoopOccurrences validates the number of loop occurrences
func (b *ConfigurableBuilder) validateLoopOccurrences(loop LoopDefinition, count int) error {
	if count < loop.MinOccurs {
		return fmt.Errorf("loop %s has %d occurrences, minimum %d required",
			loop.LoopID, count, loop.MinOccurs)
	}

	if loop.MaxOccurs > 0 && count > loop.MaxOccurs {
		return fmt.Errorf("loop %s has %d occurrences, maximum %d allowed",
			loop.LoopID, count, loop.MaxOccurs)
	}

	return nil
}

// applyMapping applies a mapping to extract data from a segment
func (b *ConfigurableBuilder) applyMapping(
	result map[string]any,
	seg *segments.ProcessedSegment,
	mapping SegmentMapping,
) error {
	obj := make(map[string]any)

	for _, elem := range mapping.Elements {
		key := fmt.Sprintf("%s%02d", seg.Schema.ID, elem.ElementPosition)
		if value, ok := seg.Data[key]; ok && value != nil && value != "" {
			if elem.Transform != "" {
				value = b.applyTransform(value, elem.Transform)
			}

			convertedValue := b.convertValue(value, elem.DataType, elem.Format)
			obj[elem.ObjectField] = convertedValue
		} else if elem.DefaultValue != nil {
			obj[elem.ObjectField] = elem.DefaultValue
		}
	}

	if len(obj) > 0 {
		b.setValueAtPath(result, mapping.ObjectPath, obj)
	}

	return nil
}

func (b *ConfigurableBuilder) buildSTSegment() string {
	return fmt.Sprintf("ST%s%s%s0001",
		string(b.delims.Element),
		b.config.TransactionType,
		string(b.delims.Element))
}

func (b *ConfigurableBuilder) buildSESegment(count int) string {
	return fmt.Sprintf("SE%s%d%s0001",
		string(b.delims.Element),
		count,
		string(b.delims.Element))
}

// evaluateValidationCondition checks if a validation rule condition is met (meaning rule is violated)
func (b *ConfigurableBuilder) evaluateValidationCondition(cond Condition, data any) bool {
	if handler := b.getSpecialFieldHandler(cond.Field); handler != nil {
		return handler(cond, data)
	}

	if b.isSegmentCondition(cond) {
		return b.evaluateSegmentCondition(cond, data)
	}

	if b.isStopCondition(cond) {
		return b.evaluateStopCondition(cond, data)
	}

	return b.evaluateCondition(cond, data)
}

// getSpecialFieldHandler returns a handler for special field patterns
func (b *ConfigurableBuilder) getSpecialFieldHandler(field string) func(Condition, any) bool {
	// ! Handle N1[01=SH|SF] type patterns
	if strings.Contains(field, "[") && strings.Contains(field, "]") {
		return b.handleN1SpecialPattern
	}
	return nil
}

// handleN1SpecialPattern handles N1[01=value] type patterns
func (b *ConfigurableBuilder) handleN1SpecialPattern(cond Condition, data any) bool {
	parts := strings.Split(cond.Field, "[")
	if len(parts) != 2 {
		return false
	}

	segmentType := parts[0]
	condition := strings.TrimSuffix(parts[1], "]")

	if segmentType != "N1" || cond.Type != "not_exists" {
		return false
	}

	condParts := strings.Split(condition, "=")
	if len(condParts) != 2 {
		return false
	}

	expectedValues := strings.Split(condParts[1], "|")

	segments := b.getSegmentsFromData(data)
	if segments == nil {
		return true
	}

	for _, seg := range segments {
		if tag, ok := seg["_tag"].(string); ok && tag == "N1" {
			if n101, exists := seg["N101"]; exists {
				n101Str := fmt.Sprintf("%v", n101)
				for _, expectedValue := range expectedValues {
					if n101Str == strings.TrimSpace(expectedValue) {
						return false
					}
				}
			}
		}
	}

	return true
}

// isSegmentCondition checks if this is a segment-related condition
func (b *ConfigurableBuilder) isSegmentCondition(cond Condition) bool {
	// ! Check if it's a segment condition (not stops and not special patterns)
	return cond.Field != "stops" && cond.Field != "S5" &&
		!strings.Contains(cond.Field, "[") && !strings.Contains(cond.Field, ".")
}

// evaluateSegmentCondition evaluates segment-specific conditions
func (b *ConfigurableBuilder) evaluateSegmentCondition(cond Condition, data any) bool {
	segments := b.getSegmentsFromData(data)
	if segments == nil {
		return cond.Type == "not_exists"
	}

	switch cond.Type {
	case "not_exists":
		for _, seg := range segments {
			if tag, ok := seg["_tag"].(string); ok && tag == cond.Field {
				return false
			}
		}
		return true

	case "count_less_than":
		count := 0
		for _, seg := range segments {
			if tag, ok := seg["_tag"].(string); ok && tag == cond.Field {
				count++
			}
		}
		return count < b.getIntValue(cond.Value)

	default:
		return false
	}
}

// isStopCondition checks if this is a stop-related condition
func (b *ConfigurableBuilder) isStopCondition(cond Condition) bool {
	return cond.Field == "stops" || cond.Field == "S5"
}

// evaluateStopCondition evaluates stop-specific conditions
func (b *ConfigurableBuilder) evaluateStopCondition(cond Condition, data any) bool {
	if cond.Type != "count_less_than" {
		return false
	}

	minCount := b.getIntValue(cond.Value)

	segments := b.getSegmentsFromData(data)
	if segments != nil {
		s5Count := 0
		for _, seg := range segments {
			if tag, ok := seg["_tag"].(string); ok && tag == "S5" {
				s5Count++
			}
		}
		if s5Count > 0 {
			return s5Count < minCount
		}
	}

	if stops, ok := data.(map[string]any)["stops"]; ok {
		if stopList, ok := stops.([]any); ok {
			return len(stopList) < minCount
		}
	}

	return true
}

// getSegmentsFromData extracts segments from data
func (b *ConfigurableBuilder) getSegmentsFromData(data any) []map[string]any {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil
	}

	segments, ok := dataMap["_segments"]
	if !ok {
		return nil
	}

	segList, ok := segments.([]map[string]any)
	if !ok {
		return nil
	}

	return segList
}

// getIntValue extracts int value from interface{}
func (b *ConfigurableBuilder) getIntValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// evaluateCondition evaluates general conditions
func (b *ConfigurableBuilder) evaluateCondition(cond Condition, data any) bool {
	if len(cond.AndConditions) > 0 {
		for _, andCond := range cond.AndConditions {
			if !b.evaluateCondition(andCond, data) {
				return false
			}
		}
		return true
	}

	if len(cond.OrConditions) > 0 {
		for _, orCond := range cond.OrConditions {
			if b.evaluateCondition(orCond, data) {
				return true
			}
		}
		return false
	}

	fieldValue := b.getValueFromPath(data, cond.Field, "")
	condValue := fmt.Sprintf("%v", cond.Value)

	switch cond.Type {
	case "exists":
		return fieldValue != ""
	case "not_exists":
		return fieldValue == ""
	case "equals":
		return fieldValue == condValue
	case "not_equals":
		return fieldValue != condValue
	case "contains":
		return strings.Contains(fieldValue, condValue)
	case "not_contains":
		return !strings.Contains(fieldValue, condValue)
	case "starts_with":
		return strings.HasPrefix(fieldValue, condValue)
	case "ends_with":
		return strings.HasSuffix(fieldValue, condValue)
	case "greater_than":
		return b.compareNumeric(fieldValue, condValue, ">")
	case "less_than":
		return b.compareNumeric(fieldValue, condValue, "<")
	case "greater_or_equal":
		return b.compareNumeric(fieldValue, condValue, ">=")
	case "less_or_equal":
		return b.compareNumeric(fieldValue, condValue, "<=")
	case "regex_match":
		matched, _ := regexp.MatchString(condValue, fieldValue)
		return matched
	case "count_less_than":
		items := b.getArrayItems(b.getValueFromPath(data, cond.Field, ""))
		return len(items) < b.getIntValue(cond.Value)
	case "count_greater_than":
		items := b.getArrayItems(b.getValueFromPath(data, cond.Field, ""))
		return len(items) > b.getIntValue(cond.Value)
	case "count_equals":
		items := b.getArrayItems(b.getValueFromPath(data, cond.Field, ""))
		return len(items) == b.getIntValue(cond.Value)
	default:
		return false
	}
}

// compareNumeric compares two values numerically
func (b *ConfigurableBuilder) compareNumeric(value1, value2, operator string) bool {
	v1, err1 := strconv.ParseFloat(value1, 64)
	v2, err2 := strconv.ParseFloat(value2, 64)

	if err1 != nil || err2 != nil {
		return false
	}

	switch operator {
	case ">":
		return v1 > v2
	case "<":
		return v1 < v2
	case ">=":
		return v1 >= v2
	case "<=":
		return v1 <= v2
	default:
		return false
	}
}

// getValueFromPath retrieves a value from nested data structure
func (b *ConfigurableBuilder) getValueFromPath(data any, basePath, field string) string {
	if basePath == "" || strings.HasSuffix(basePath, "[]") {
		if m, ok := data.(map[string]any); ok {
			if val, exists := m[field]; exists && val != nil {
				return fmt.Sprintf("%v", val)
			}
		}
		return ""
	}

	if m, ok := data.(map[string]any); ok {
		fullPath := basePath
		if field != "" {
			fullPath = basePath + "." + field
		}

		parts := strings.Split(fullPath, ".")
		current := any(m)

		for _, part := range parts {
			part = strings.TrimSuffix(part, "[]")

			if currentMap, ok := current.(map[string]any); ok {
				current = currentMap[part]
			} else {
				return ""
			}
		}

		if current != nil {
			return fmt.Sprintf("%v", current)
		}
	}
	return ""
}

// setValueAtPath sets a value at a nested path in the result
func (b *ConfigurableBuilder) setValueAtPath(result map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := result

	for i, part := range parts {
		if strings.HasSuffix(part, "[]") {
			part = strings.TrimSuffix(part, "[]")

			if i == len(parts)-1 {
				if arr, ok := current[part].([]any); ok {
					current[part] = append(arr, value)
				} else {
					current[part] = []any{value}
				}
			} else {
				if _, ok := current[part]; !ok {
					current[part] = []any{}
				}
			}
		} else {
			if i == len(parts)-1 {
				current[part] = value
			} else {
				if _, ok := current[part]; !ok {
					current[part] = make(map[string]any)
				}
				current = current[part].(map[string]any)
			}
		}
	}
}

// getLoopData retrieves loop data based on configuration
func (b *ConfigurableBuilder) getLoopData(data any, startSegment string) any {
	if b.config.LoopMappings != nil {
		if mapping, ok := b.config.LoopMappings[startSegment]; ok {
			return b.getValueFromPath(data, mapping, "")
		}
	}

	// Fall back to common patterns
	if m, ok := data.(map[string]any); ok {
		switch startSegment {
		case "N1":
			// Check common party/partner fields
			for _, field := range []string{"parties", "partners", "trading_partners", "n1_loop"} {
				if val, ok := m[field]; ok {
					return val
				}
			}
		case "S5":
			// Check common stop fields
			for _, field := range []string{"stops", "stop_offs", "stop_details", "s5_loop"} {
				if val, ok := m[field]; ok {
					return val
				}
			}
		case "L11":
			// Check reference fields
			for _, field := range []string{"references", "reference_numbers", "l11_loop"} {
				if val, ok := m[field]; ok {
					return val
				}
			}
		case "AT7":
			// Check status fields
			for _, field := range []string{"statuses", "shipment_statuses", "at7_loop"} {
				if val, ok := m[field]; ok {
					return val
				}
			}
		case "MS3":
			// Check interline fields
			for _, field := range []string{"interline", "interline_info", "ms3_loop"} {
				if val, ok := m[field]; ok {
					return val
				}
			}
		case "NTE":
			// Check notes fields
			for _, field := range []string{"notes", "comments", "special_instructions", "nte_loop"} {
				if val, ok := m[field]; ok {
					return val
				}
			}
		default:
			// Try to find loop data by lowercase segment name
			lowerSegment := strings.ToLower(startSegment)
			if val, ok := m[lowerSegment+"_loop"]; ok {
				return val
			}
			if val, ok := m[lowerSegment+"s"]; ok {
				return val
			}
		}
	}

	return nil
}

// getArrayItems converts data to array of items
func (b *ConfigurableBuilder) getArrayItems(data any) []any {
	if data == nil {
		return []any{}
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice {
		items := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			items[i] = v.Index(i).Interface()
		}
		return items
	}

	return []any{data}
}

// applyTransform applies transformation to a value
func (b *ConfigurableBuilder) applyTransform(value any, transform string) string {
	str := fmt.Sprintf("%v", value)

	// Handle built-in transformations
	switch transform {
	case "uppercase":
		return strings.ToUpper(str)
	case "lowercase":
		return strings.ToLower(str)
	case "trim":
		return strings.TrimSpace(str)
	case "remove_spaces":
		return strings.ReplaceAll(str, " ", "")
	case "remove_special":
		// Remove non-alphanumeric characters
		reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
		return reg.ReplaceAllString(str, "")
	case "alpha_only":
		reg := regexp.MustCompile(`[^a-zA-Z]+`)
		return reg.ReplaceAllString(str, "")
	case "numeric_only":
		reg := regexp.MustCompile(`[^0-9]+`)
		return reg.ReplaceAllString(str, "")
	}

	// Handle parameterized transformations
	if strings.Contains(transform, ":") {
		parts := strings.SplitN(transform, ":", 2)
		switch parts[0] {
		case "truncate":
			if length, err := strconv.Atoi(parts[1]); err == nil && length > 0 {
				if len(str) > length {
					return str[:length]
				}
			}
		case "pad_left":
			if length, err := strconv.Atoi(parts[1]); err == nil {
				return fmt.Sprintf("%*s", length, str)
			}
		case "pad_right":
			if length, err := strconv.Atoi(parts[1]); err == nil {
				return fmt.Sprintf("%-*s", length, str)
			}
		case "pad_zero":
			if length, err := strconv.Atoi(parts[1]); err == nil {
				return fmt.Sprintf("%0*s", length, str)
			}
		}
	}

	// Check customer transformations
	if b.customer != nil {
		for _, trans := range b.customer.Transformations {
			if trans.Name == transform {
				return b.applyCustomTransform(str, trans)
			}
		}
	}

	return str
}

// applyCustomTransform applies customer-specific transformation
func (b *ConfigurableBuilder) applyCustomTransform(value string, trans TransformationRule) string {
	switch trans.Type {
	case "replace":
		if old, ok := trans.Parameters["old"]; ok {
			if new, ok := trans.Parameters["new"]; ok {
				return strings.ReplaceAll(value, old, new)
			}
		}
	case "regex_replace":
		if pattern, ok := trans.Parameters["pattern"]; ok {
			if replacement, ok := trans.Parameters["replacement"]; ok {
				if reg, err := regexp.Compile(pattern); err == nil {
					return reg.ReplaceAllString(value, replacement)
				}
			}
		}
	case "format":
		if template, ok := trans.Parameters["template"]; ok {
			return fmt.Sprintf(template, value)
		}
	case "map":
		if mappings, ok := trans.Parameters["mappings"]; ok {
			for _, mapping := range strings.Split(mappings, ",") {
				parts := strings.Split(mapping, ":")
				if len(parts) == 2 && strings.TrimSpace(parts[0]) == value {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	case "pad":
		if lengthStr, ok := trans.Parameters["length"]; ok {
			if length, err := strconv.Atoi(lengthStr); err == nil {
				if char, ok := trans.Parameters["char"]; ok {
					if trans.Parameters["direction"] == "left" {
						return strings.Repeat(char, length-len(value)) + value
					} else {
						return value + strings.Repeat(char, length-len(value))
					}
				}
			}
		}
	}
	return value
}

// convertValue converts a value to the specified data type
func (b *ConfigurableBuilder) convertValue(value any, dataType, format string) any {
	str := fmt.Sprintf("%v", value)

	switch dataType {
	case "number", "numeric", "decimal":
		if n, err := strconv.ParseFloat(str, 64); err == nil {
			return n
		}
	case "integer", "int":
		if n, err := strconv.Atoi(str); err == nil {
			return n
		}
	case "date":
		if format != "" {
			if t, err := time.Parse(format, str); err == nil {
				return t
			}
		}
		for _, fmt := range []string{"20060102", "2006-01-02", "01/02/2006", "01-02-2006"} {
			if t, err := time.Parse(fmt, str); err == nil {
				return t
			}
		}
	case "time":
		if format != "" {
			if t, err := time.Parse(format, str); err == nil {
				return t
			}
		}
		for _, fmt := range []string{"1504", "15:04", "15:04:05", "150405"} {
			if t, err := time.Parse(fmt, str); err == nil {
				return t
			}
		}
	case "datetime":
		if format != "" {
			if t, err := time.Parse(format, str); err == nil {
				return t
			}
		}
		for _, fmt := range []string{
			"20060102150405",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		} {
			if t, err := time.Parse(fmt, str); err == nil {
				return t
			}
		}
	case "boolean", "bool":
		return str == "true" || str == "1" || str == "Y" || str == "yes"
	}

	return str
}

// getSeverity converts string severity to validation.Severity
func (b *ConfigurableBuilder) getSeverity(sev string) validation.Severity {
	switch strings.ToLower(sev) {
	case "warning", "warn", "info", "information":
		return validation.Warning
	case "error", "fatal":
		return validation.Error
	default:
		return validation.Error
	}
}
