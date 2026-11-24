package fiscalcloseblockers

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalclose"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Refusal struct {
	Field    string
	Category string
	Check    fiscalclose.Check
}

func Append(
	ctx context.Context,
	blockers []*fiscalclose.Blocker,
	err error,
	refusal Refusal,
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
				Message:  errortypes.LocalizedText(ctx, item),
				Category: refusal.Category,
				Check:    refusal.Check,
			})
		}
		return blockers, nil
	}

	errorable, ok := errors.AsType[errortypes.Errorable](err)
	if !ok || !errortypes.IsRefusal(err) {
		return blockers, err
	}

	return append(blockers, &fiscalclose.Blocker{
		Field:    refusal.Field,
		Code:     errorable.GetCode(),
		Message:  errortypes.LocalizedText(ctx, err),
		Category: refusal.Category,
		Check:    refusal.Check,
	}), nil
}

func Counted(
	ctx context.Context,
	refusal Refusal,
	count int,
	message string,
) *fiscalclose.Blocker {
	refused := errortypes.NewValidationError(
		refusal.Field,
		errortypes.ErrInvalidOperation,
		message,
		count,
	)

	return &fiscalclose.Blocker{
		Field:    refusal.Field,
		Code:     errortypes.ErrInvalidOperation,
		Message:  errortypes.LocalizedText(ctx, refused),
		Category: refusal.Category,
		Check:    refusal.Check,
		Count:    &count,
	}
}
