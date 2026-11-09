package errortypes

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/fieldpath"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/textcase"
)

func IsRefusal(err error) bool {
	if err == nil {
		return false
	}
	if IsMultiError(err) {
		return true
	}
	return IsBusinessError(err) || IsError(err) || IsNotFoundError(err) || IsConflictError(err)
}

func Summary(err error) string {
	if err == nil {
		return ""
	}
	if multiErr, ok := RefusalOf(err); ok && len(multiErr.top().Errors) > 0 {
		fieldErrs := multiErr.top().Errors
		messages := make([]string, 0, len(fieldErrs))
		for _, fieldErr := range fieldErrs {
			messages = append(messages, located(fieldErr.Path, fieldErr.Error()))
		}
		return strings.Join(messages, "; ")
	}
	return err.Error()
}

func SummaryOfText(text string) string {
	text = strings.TrimSpace(text)
	body, isValidation := strings.CutPrefix(text, validationFailedHeader)
	if !isValidation {
		return text
	}

	lines := strings.Split(strings.TrimSpace(body), "\n")
	messages := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if line == "" {
			continue
		}
		field, message, named := strings.Cut(line, ": ")
		if !named || strings.ContainsAny(field, " ") {
			messages = append(messages, line)
			continue
		}
		messages = append(messages, located(fieldpath.Parse(field), message))
	}
	if len(messages) == 0 {
		return text
	}

	return strings.Join(messages, "; ")
}

func located(path fieldpath.Path, message string) string {
	place := fieldPlace(path)
	if place == "" {
		return message
	}

	return place + ": " + message
}

func fieldPlace(path fieldpath.Path) string {
	parts := make([]string, 0, 2)
	for i, segment := range path {
		if !segment.IsIndex() || i == 0 || path[i-1].IsIndex() || path[i-1].Key() == "" {
			continue
		}
		name := path[i-1].Key()
		noun := inflect.SingularNoun(strings.ToLower(textcase.HumanizeCamelCaseSentence(name)))
		parts = append(parts, noun+" "+strconv.Itoa(segment.Index()+1))
	}
	if len(parts) == 0 {
		return ""
	}

	return textcase.CapitalizeFirst(strings.Join(parts, ", "))
}
