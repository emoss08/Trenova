package validator

import "fmt"

// DBValidationError is an error that occurs during database validation
type DBValidationError struct {
	Field   string
	Message string
}

func (e DBValidationError) Error() string {
	return fmt.Sprintf("field %s: %s", e.Field, e.Message)
}

// BusinessLogicError is an error that occurs during business logic validation
type BusinessLogicError struct {
	Message string
}

func (e BusinessLogicError) Error() string {
	return e.Message
}

// MultiValidationError is a collection of DBValidationErrors
type MultiValidationError struct {
	Errors []DBValidationError
}

func (m MultiValidationError) Error() string {
	return fmt.Sprintf("multiple validation errors occurred (%d)", len(m.Errors))
}
