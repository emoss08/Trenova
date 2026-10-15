package domainvalidation

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

func EmailList(message string) validation.Rule {
	return validation.By(func(value any) error {
		raw, ok := value.(string)
		if !ok {
			return errors.New(message)
		}
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		addresses := stringutils.SplitEmailList(raw)
		if len(addresses) == 0 {
			return errors.New(message)
		}
		for _, address := range addresses {
			if err := is.EmailFormat.Validate(address); err != nil {
				return errors.New(message)
			}
		}
		return nil
	})
}
