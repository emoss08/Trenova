// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package rptmetavalidator

import (
	"github.com/rotisserie/eris"
)

var (
	ErrEmptySQL        = eris.New("sql query cannot be empty")
	ErrNoPlaceholder   = eris.New("no placeholder found in sql query")
	ErrInvalidMetadata = eris.New("invalid metadata structure")
)
