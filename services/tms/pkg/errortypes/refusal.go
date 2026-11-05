package errortypes

import (
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/textcase"
)

func IsRefusal(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := errors.AsType[*MultiError](err); ok {
		return true
	}
	return IsBusinessError(err) || IsError(err) || IsNotFoundError(err) || IsConflictError(err)
}

func Summary(err error) string {
	if err == nil {
		return ""
	}
	if multiErr, ok := errors.AsType[*MultiError](err); ok && len(multiErr.Errors) > 0 {
		messages := make([]string, 0, len(multiErr.Errors))
		for _, fieldErr := range multiErr.Errors {
			messages = append(messages, located(fieldErr.Field, fieldErr.Error()))
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
		messages = append(messages, located(field, message))
	}
	if len(messages) == 0 {
		return text
	}

	return strings.Join(messages, "; ")
}

func located(field, message string) string {
	place := fieldPlace(field)
	if place == "" {
		return message
	}

	return place + ": " + message
}

func fieldPlace(field string) string {
	parts := make([]string, 0, 2)
	for segment := range strings.SplitSeq(field, ".") {
		name, rest, indexed := strings.Cut(segment, "[")
		if !indexed {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSuffix(rest, "]"))
		if err != nil || name == "" {
			continue
		}
		noun := inflect.SingularNoun(strings.ToLower(textcase.HumanizeCamelCaseSentence(name)))
		parts = append(parts, noun+" "+strconv.Itoa(index+1))
	}
	if len(parts) == 0 {
		return ""
	}

	return textcase.CapitalizeFirst(strings.Join(parts, ", "))
}
