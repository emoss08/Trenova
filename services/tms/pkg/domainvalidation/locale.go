package domainvalidation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/shared/i18n"
)

var ErrLocaleMustBeString = errors.New("locale must be a string")

func ValidateLocale(value any) error {
	tag, ok := value.(string)
	if !ok {
		return ErrLocaleMustBeString
	}

	if tag == "" {
		return nil
	}

	if i18n.Locale(tag).IsValid() {
		return nil
	}

	supported := i18n.Supported()
	names := make([]string, len(supported))
	for i, locale := range supported {
		names[i] = locale.String()
	}

	return fmt.Errorf("unsupported language %q, expected one of: %s", tag, strings.Join(names, ", "))
}
