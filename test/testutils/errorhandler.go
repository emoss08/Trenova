package testutils

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type ErrorMatcher struct {
	t        *testing.T
	multiErr *errors.MultiError
}

func NewErrorMatcher(t *testing.T, multiErr *errors.MultiError) *ErrorMatcher {
	return &ErrorMatcher{
		t:        t,
		multiErr: multiErr,
	}
}

// Debug prints the current errors for debugging
func (m *ErrorMatcher) Debug() {
	if m.multiErr == nil {
		fmt.Println("No errors (multiErr is nil)")
		return
	}
	fmt.Println("Current Errors:")
	for i, err := range m.multiErr.Errors {
		fmt.Printf("%d. Field: %q, Code: %q, Message: %q\n", i+1, err.Field, err.Code, err.Message)
	}
}

// HasExactErrors checks if the MultiError contains exactly these errors
func (m *ErrorMatcher) HasExactErrors(expectedErrors []struct {
	Field   string
	Code    errors.ErrorCode
	Message string
},
) bool {
	if m.multiErr == nil {
		assert.Fail(m.t, "MultiError is nil but expected errors")
		return false
	}

	if len(m.multiErr.Errors) != len(expectedErrors) {
		assert.Fail(m.t, fmt.Sprintf("Expected %d errors but got %d",
			len(expectedErrors), len(m.multiErr.Errors)))
		m.Debug()
		return false
	}

	for _, expected := range expectedErrors {
		found := false
		for _, actual := range m.multiErr.Errors {
			if actual.Field == expected.Field &&
				actual.Code == expected.Code &&
				actual.Message == expected.Message {
				found = true
				break
			}
		}
		if !found {
			assert.Fail(
				m.t,
				fmt.Sprintf("Expected error not found - Field: %s, Code: %s, Message: %s",
					expected.Field, expected.Code, expected.Message),
			)
			m.Debug() // Print current errors for debugging
			return false
		}
	}
	return true
}

func (m *ErrorMatcher) HasError(field string, code errors.ErrorCode, message string) bool {
	if m.multiErr == nil {
		assert.Fail(m.t, "MultiError is nil but expected an error")
		return false
	}

	found := false
	for _, err := range m.multiErr.Errors {
		if err.Field == field && err.Code == code && err.Message == message {
			found = true
			break
		}
	}

	if !found {
		assert.Fail(m.t, fmt.Sprintf("Expected error not found - Field: %s, Code: %s, Message: %s",
			field, code, message))
		m.Debug() // Print current errors for debugging
	}
	return found
}

func (m *ErrorMatcher) HasNoErrors() {
	if m.multiErr == nil {
		return
	}
	assert.Empty(m.t, m.multiErr.Errors, "Expected no errors but got: %v", m.multiErr.Errors)
}

func (m *ErrorMatcher) ErrorCount(expected int) {
	if m.multiErr == nil {
		assert.Zero(m.t, expected, "Expected %d errors but MultiError is nil", expected)
		return
	}
	assert.Len(m.t, m.multiErr.Errors, expected,
		"Expected %d errors but got %d\nCurrent errors: %v",
		expected, len(m.multiErr.Errors), m.multiErr.Errors)
}

// HasFieldError checks if there's an error for a specific field regardless of code/message
func (m *ErrorMatcher) HasFieldError(field string) bool {
	if m.multiErr == nil {
		assert.Fail(m.t, "MultiError is nil but expected an error")
		return false
	}

	for _, err := range m.multiErr.Errors {
		if err.Field == field {
			return true
		}
	}

	assert.Fail(m.t, fmt.Sprintf("No error found for field: %s", field))
	m.Debug()
	return false
}
