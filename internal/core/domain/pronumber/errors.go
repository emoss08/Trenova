package pronumber

import "github.com/emoss08/trenova/internal/pkg/errors"

// ErrSequenceNotFound indicates that a sequence wasn't found for the given period
var ErrSequenceNotFound = errors.NewNotFoundError("Sequence not found for the given period")

// ErrSequenceUpdateConflict indicates a concurrent update conflict when generating numbers
var ErrSequenceUpdateConflict = errors.NewBusinessError("Sequence update conflict occurred")
