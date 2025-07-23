// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package queryutils

import "github.com/rotisserie/eris"

var (
	ErrTableRequired          = eris.New("table name is required")
	ErrFieldNameRequired      = eris.New("field name is required")
	ErrFieldValueRequired     = eris.New("field value is required")
	ErrErrorFieldNameRequired = eris.New("error field name is required")
	ErrErrorModelNameRequired = eris.New("error model name is required")
)
