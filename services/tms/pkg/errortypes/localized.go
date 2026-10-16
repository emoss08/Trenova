package errortypes

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/shared/i18n"
)

func LocalizedText(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	var localized Localizable
	if errors.As(err, &localized) {
		message, args := localized.LocalizedMessage()
		return i18n.T(ctx, message, args...)
	}
	return err.Error()
}
