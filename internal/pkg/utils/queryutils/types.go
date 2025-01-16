package queryutils

import "github.com/rotisserie/eris"

var (
	ErrTableRequired          = eris.New("table name is required")
	ErrFieldNameRequired      = eris.New("field name is required")
	ErrFieldValueRequired     = eris.New("field value is required")
	ErrErrorFieldNameRequired = eris.New("error field name is required")
	ErrErrorModelNameRequired = eris.New("error model name is required")
)
