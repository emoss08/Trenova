// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package pronumber

import "github.com/emoss08/trenova/internal/pkg/errors"

// ErrSequenceNotFound indicates that a sequence wasn't found for the given period
var ErrSequenceNotFound = errors.NewNotFoundError("Sequence not found for the given period")

// ErrSequenceUpdateConflict indicates a concurrent update conflict when generating numbers
var ErrSequenceUpdateConflict = errors.NewBusinessError("Sequence update conflict occurred")
