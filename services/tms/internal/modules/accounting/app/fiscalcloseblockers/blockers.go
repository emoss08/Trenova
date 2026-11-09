package fiscalcloseblockers

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalclose"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func Append(
	blockers []*fiscalclose.Blocker,
	err error,
	field, category string,
) ([]*fiscalclose.Blocker, error) {
	if err == nil {
		return blockers, nil
	}

	if multiErr, ok := errortypes.RefusalOf(err); ok {
		for _, item := range multiErr.Errors {
			if item == nil {
				continue
			}
			blockers = append(blockers, &fiscalclose.Blocker{
				Field:    item.Field(),
				Code:     item.Code,
				Message:  item.Error(),
				Category: category,
			})
		}
		return blockers, nil
	}

	errorable, ok := errors.AsType[errortypes.Errorable](err)
	if !ok || !errortypes.IsRefusal(err) {
		return blockers, err
	}

	return append(blockers, &fiscalclose.Blocker{
		Field:    field,
		Code:     errorable.GetCode(),
		Message:  err.Error(),
		Category: category,
	}), nil
}
