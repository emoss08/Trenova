package queryutils

import "errors"

var (
	ErrTableRequired           = errors.New("table name is required")
	ErrModelNameRequired       = errors.New("model name is required")
	ErrFieldValueRequired      = errors.New("field value is required")
	ErrFieldErrorNameRequired  = errors.New("error field name is required")
	ErrAtLeastOneFieldRequired = errors.New("at least one field is required for validation")
	ErrFieldNameRequired       = errors.New("field name is required")
)
