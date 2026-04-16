package weatheralert

import "errors"

var ErrInvalidAlertCategory = errors.New("invalid alert category")

func ValidateAlertCategory(value any) error {
	category, ok := value.(*AlertCategory)
	if !ok || category == nil || !category.IsValid() {
		return ErrInvalidAlertCategory
	}

	return nil
}
