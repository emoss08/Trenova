package domainvalidation

import (
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Enum interface {
	IsValid() bool
}

func ValidEnum[T Enum](message string) validation.Rule {
	err := errors.New(message)

	return validation.By(func(value any) error {
		indirect, isNil := validation.Indirect(value)
		if isNil || validation.IsEmpty(indirect) {
			return nil
		}

		enum, ok := indirect.(T)
		if !ok || !enum.IsValid() {
			return err
		}

		return nil
	})
}
