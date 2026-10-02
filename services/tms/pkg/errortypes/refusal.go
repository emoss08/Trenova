package errortypes

import (
	"errors"
	"strings"
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
			messages = append(messages, fieldErr.Error())
		}
		return strings.Join(messages, "; ")
	}
	return err.Error()
}
