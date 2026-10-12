package masterdatatools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func masterOptionalIDPointer[T any](
	key string,
	resource permission.Resource,
	description string,
	at func(*T) **pulid.ID,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(update, true,
				agenttoolschema.RecordIDText(resource, description+keepSuffix(update, true)))
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			id, err := optionalMasterID(in, key)
			if err != nil {
				return err
			}
			if id.IsNil() {
				*at(entity) = nil
				return nil
			}
			*at(entity) = &id

			return nil
		},
	}
}

func masterDateTime[T any](key, description string, at func(*T) *int64) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return agenttoolschema.DateTime(description + keepSuffix(update, false))
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			text, err := in.text(key)
			if err != nil {
				return err
			}
			if text == "" {
				return errortypes.NewValidationError(key, errortypes.ErrRequired,
					"Send a date and time")
			}
			value, err := common.ParseDateTime(key, text)
			if err != nil {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid, err.Error())
			}
			*at(entity) = value

			return nil
		},
	}
}

func masterNullDecimal[T any](
	key, description string,
	at func(*T) *decimal.NullDecimal,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(
				update,
				true,
				common.StringProperty(description+" A decimal such as 2.35."+
					keepSuffix(update, true), 0),
			)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, present, err := common.OptionalDecimal(in.values, key)
			if err != nil || (present && value.IsNegative()) {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"Send a decimal that is zero or more, such as 2.35")
			}
			if !present {
				*at(entity) = decimal.NullDecimal{}
				return nil
			}
			*at(entity) = decimal.NewNullDecimal(value.Round(4))

			return nil
		},
	}
}
