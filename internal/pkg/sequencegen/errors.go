/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package sequencegen

import "github.com/rotisserie/eris"

// Errors
var (
	ErrSequenceUpdateConflict = eris.New("sequence update conflict")
	ErrInvalidYear            = eris.New("year out of range for int16")
	ErrInvalidMonth           = eris.New("month out of range for int16")
)
