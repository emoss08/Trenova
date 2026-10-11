package sim

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	fieldAttributesKey   = "attributes"
	attributeRangePrefix = "range("
	maxAttributesPerBody = 100
)

type attributeEntityType string

const (
	attributeEntityDriver attributeEntityType = keyDriver
	attributeEntityAsset  attributeEntityType = keyAsset
)

type attributeFilterMode uint8

const (
	attributeFilterDriver attributeFilterMode = iota
	attributeFilterVehicle
	attributeFilterAsset
)

type attributeCondition struct {
	Name     string
	Value    string
	IsRange  bool
	LowerNum *float64
	UpperNum *float64
	LowerDay string
	UpperDay string
}

type attributeFilter struct {
	valueIDs   []string
	conditions []attributeCondition
}

func sanitizeAttributes(raw any, path string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(path, "must be an array of attribute objects")
	}
	if len(items) > maxAttributesPerBody {
		return nil, invalidField(
			path,
			fmt.Sprintf("supports at most %d attributes", maxAttributesPerBody),
		)
	}
	out := make([]any, 0, len(items))
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, idx)
		mapped, isMap := anyAsMap(item)
		if !isMap {
			return nil, invalidField(itemPath, "must be an object")
		}
		attribute, err := sanitizeAttribute(mapped, itemPath)
		if err != nil {
			return nil, err
		}
		out = append(out, attribute)
	}
	return out, nil
}

func sanitizeAttribute(raw map[string]any, path string) (map[string]any, error) {
	id, _ := raw[keyID].(string)
	name, _ := raw[keyName].(string)
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" && name == "" {
		return nil, invalidField(path, "requires an id or a name")
	}
	if id != "" && !uuidLike(id) {
		return nil, invalidField(path+".id", "must be a UUID")
	}
	out := map[string]any{}
	if id != "" {
		out[keyID] = id
	}
	if name != "" {
		out[keyName] = name
	}
	stringValues, err := attributeStrings(raw["stringValues"], path+".stringValues")
	if err != nil {
		return nil, err
	}
	dateValues, err := attributeStrings(raw["dateValues"], path+".dateValues")
	if err != nil {
		return nil, err
	}
	for idx, value := range dateValues {
		if _, parseErr := time.Parse(dateLayout, stringOf(value)); parseErr != nil {
			return nil, invalidField(
				fmt.Sprintf("%s.dateValues[%d]", path, idx),
				"must be a YYYY-MM-DD date",
			)
		}
	}
	numberValues, err := attributeNumbers(raw["numberValues"], path+".numberValues")
	if err != nil {
		return nil, err
	}
	if len(stringValues)+len(dateValues)+len(numberValues) == 0 {
		return nil, invalidField(path, "requires stringValues, numberValues or dateValues")
	}
	if len(stringValues) > 0 {
		out["stringValues"] = stringValues
	}
	if len(dateValues) > 0 {
		out["dateValues"] = dateValues
	}
	if len(numberValues) > 0 {
		out["numberValues"] = numberValues
	}
	return out, nil
}

func attributeStrings(raw any, path string) ([]any, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(path, "must be an array of strings")
	}
	out := make([]any, 0, len(items))
	for idx, item := range items {
		text, isString := item.(string)
		if !isString || strings.TrimSpace(text) == "" {
			return nil, invalidField(fmt.Sprintf("%s[%d]", path, idx), "must be a non-empty string")
		}
		out = append(out, strings.TrimSpace(text))
	}
	return out, nil
}

func attributeNumbers(raw any, path string) ([]any, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(path, "must be an array of numbers")
	}
	out := make([]any, 0, len(items))
	for idx, item := range items {
		number, isNumber := item.(float64)
		if !isNumber || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, invalidField(fmt.Sprintf("%s[%d]", path, idx), "must be a number")
		}
		out = append(out, number)
	}
	return out, nil
}

func uuidLike(value string) bool {
	return len(value) == 36 && strings.Count(value, "-") == 4
}

func resolveAttributeIDs(attributes []any, entityType attributeEntityType) []any {
	out := make([]any, 0, len(attributes))
	for _, item := range attributes {
		mapped, ok := anyAsMap(item)
		if !ok {
			continue
		}
		attribute := cloneMap(mapped)
		if stringValue(Record(attribute), keyID) == "" {
			attribute[keyID] = attributeIDForName(
				entityType,
				stringValue(Record(attribute), keyName),
			)
		}
		out = append(out, attribute)
	}
	return out
}

func attributeIDForName(entityType attributeEntityType, name string) string {
	return deterministicUUID(
		"attribute",
		string(entityType),
		strings.ToLower(strings.TrimSpace(name)),
	)
}

func attributeValueID(attributeID, value string) string {
	return deterministicUUID(
		"attribute-value",
		strings.TrimSpace(attributeID),
		strings.TrimSpace(value),
	)
}

func renderAttributes(record Record) []any {
	items, ok := record[fieldAttributesKey].([]any)
	if !ok {
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		if mapped, isMap := anyAsMap(item); isMap {
			out = append(out, cloneMap(mapped))
		}
	}
	return out
}

func parseAttributeFilter(
	values url.Values,
	mode attributeFilterMode,
) (attributeFilter, error) {
	filter := attributeFilter{valueIDs: csvQueryValues(values, "attributeValueIds")}
	for _, raw := range values["attributes"] {
		for _, expression := range splitAttributeExpressions(raw) {
			condition, err := parseAttributeCondition(expression, mode)
			if err != nil {
				return attributeFilter{}, err
			}
			filter.conditions = append(filter.conditions, condition)
		}
	}
	return filter, nil
}

func csvQueryValues(values url.Values, name string) []string {
	raw := values[name]
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		out = append(out, splitCSV(entry)...)
	}
	return out
}

func splitAttributeExpressions(raw string) []string {
	out := []string{}
	depth := 0
	start := 0
	for idx := 0; idx < len(raw); idx++ {
		switch raw[idx] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',', ';':
			if depth == 0 {
				if part := strings.TrimSpace(raw[start:idx]); part != "" {
					out = append(out, part)
				}
				start = idx + 1
			}
		}
	}
	if part := strings.TrimSpace(raw[start:]); part != "" {
		out = append(out, part)
	}
	return out
}

func parseAttributeCondition(
	expression string,
	mode attributeFilterMode,
) (attributeCondition, error) {
	name, value, found := strings.Cut(expression, ":")
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	if !found || name == "" || value == "" {
		return attributeCondition{}, invalidParameter(
			"attributes",
			fmt.Sprintf("%q must be name:value or name:range(min,max)", expression),
		)
	}
	if !strings.HasPrefix(value, attributeRangePrefix) || !strings.HasSuffix(value, ")") {
		if mode == attributeFilterAsset {
			return attributeCondition{}, invalidParameter(
				"attributes",
				fmt.Sprintf("%q must be a range query such as name:range(min,max)", expression),
			)
		}
		return attributeCondition{Name: name, Value: value}, nil
	}
	bounds := strings.TrimSuffix(strings.TrimPrefix(value, attributeRangePrefix), ")")
	lower, upper, comma := strings.Cut(bounds, ",")
	lower = strings.TrimSpace(lower)
	upper = strings.TrimSpace(upper)
	if !comma || (lower == "" && upper == "") {
		return attributeCondition{}, invalidParameter(
			"attributes",
			fmt.Sprintf("%q must provide at least one range bound", expression),
		)
	}
	condition := attributeCondition{Name: name, IsRange: true}
	if numeric, err := parseNumericBounds(lower, upper); err == nil {
		condition.LowerNum, condition.UpperNum = numeric[0], numeric[1]
		return condition, nil
	}
	if mode == attributeFilterDriver {
		return attributeCondition{}, invalidParameter(
			"attributes",
			fmt.Sprintf("%q range bounds must be numbers", expression),
		)
	}
	if !validOptionalDate(lower) || !validOptionalDate(upper) {
		return attributeCondition{}, invalidParameter(
			"attributes",
			fmt.Sprintf("%q range bounds must be numbers or YYYY-MM-DD dates", expression),
		)
	}
	condition.LowerDay, condition.UpperDay = lower, upper
	return condition, nil
}

func parseNumericBounds(lower, upper string) ([2]*float64, error) {
	out := [2]*float64{}
	for idx, raw := range []string{lower, upper} {
		if raw == "" {
			continue
		}
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return out, fmt.Errorf("parse range bound %q: %w", raw, err)
		}
		out[idx] = &parsed
	}
	return out, nil
}

func validOptionalDate(raw string) bool {
	if raw == "" {
		return true
	}
	_, err := time.Parse(dateLayout, raw)
	return err == nil
}

func (f attributeFilter) active() bool {
	return len(f.valueIDs) > 0 || len(f.conditions) > 0
}

func (f attributeFilter) matches(record Record) bool {
	if !f.active() {
		return true
	}
	attributes := renderAttributes(record)
	if len(f.valueIDs) > 0 {
		owned := make(map[string]struct{}, len(attributes)*2)
		for _, item := range attributes {
			attribute := Record(mapOf(item))
			attributeID := stringValue(attribute, keyID)
			for _, value := range stringListValues(attribute["stringValues"]) {
				owned[attributeValueID(attributeID, value)] = struct{}{}
			}
		}
		for _, valueID := range f.valueIDs {
			if _, ok := owned[valueID]; !ok {
				return false
			}
		}
	}
	for idx := range f.conditions {
		if !conditionMatches(&f.conditions[idx], attributes) {
			return false
		}
	}
	return true
}

func conditionMatches(condition *attributeCondition, attributes []any) bool {
	for _, item := range attributes {
		attribute := Record(mapOf(item))
		if !strings.EqualFold(stringValue(attribute, keyName), condition.Name) {
			continue
		}
		if condition.IsRange {
			if rangeMatches(condition, attribute) {
				return true
			}
			continue
		}
		if valueMatches(condition.Value, attribute) {
			return true
		}
	}
	return false
}

func valueMatches(value string, attribute Record) bool {
	for _, key := range []string{"stringValues", "dateValues"} {
		for _, candidate := range stringListValues(attribute[key]) {
			if candidate == value {
				return true
			}
		}
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return false
	}
	for _, candidate := range attributeNumberValues(attribute) {
		if candidate == number {
			return true
		}
	}
	return false
}

func rangeMatches(condition *attributeCondition, attribute Record) bool {
	if condition.LowerDay != "" || condition.UpperDay != "" {
		for _, day := range stringListValues(attribute["dateValues"]) {
			if (condition.LowerDay == "" || day >= condition.LowerDay) &&
				(condition.UpperDay == "" || day <= condition.UpperDay) {
				return true
			}
		}
		return false
	}
	for _, number := range attributeNumberValues(attribute) {
		if (condition.LowerNum == nil || number >= *condition.LowerNum) &&
			(condition.UpperNum == nil || number <= *condition.UpperNum) {
			return true
		}
	}
	return false
}

func attributeNumberValues(attribute Record) []float64 {
	items, ok := attribute["numberValues"].([]any)
	if !ok {
		return nil
	}
	out := make([]float64, 0, len(items))
	for _, item := range items {
		out = append(out, floatFromAny(item))
	}
	return out
}
