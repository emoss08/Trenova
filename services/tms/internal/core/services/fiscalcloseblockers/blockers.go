package fiscalcloseblockers

import (
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func AppendFromMultiError(
	blockers []*fiscalclose.Blocker,
	multiErr *errortypes.MultiError,
	category string,
) []*fiscalclose.Blocker {
	if multiErr == nil || !multiErr.HasErrors() {
		return blockers
	}

	for _, item := range multiErr.Errors {
		if item == nil {
			continue
		}
		blockers = append(
			blockers,
			&fiscalclose.Blocker{
				Field:    item.Field,
				Code:     item.Code,
				Message:  item.Message,
				Category: category,
			},
		)
	}

	return blockers
}

func AppendFromError(
	blockers []*fiscalclose.Blocker,
	err error,
	field, category string,
) []*fiscalclose.Blocker {
	if err == nil {
		return blockers
	}

	var multiErr *errortypes.MultiError
	if errors.As(err, &multiErr) {
		return AppendFromMultiError(blockers, multiErr, category)
	}

	var errorable errortypes.Errorable
	if errors.As(err, &errorable) {
		return append(
			blockers,
			&fiscalclose.Blocker{
				Field:    field,
				Code:     errorable.GetCode(),
				Message:  err.Error(),
				Category: category,
			},
		)
	}

	return append(
		blockers,
		&fiscalclose.Blocker{
			Field:    field,
			Code:     errortypes.ErrSystemError,
			Message:  err.Error(),
			Category: category,
		},
	)
}
