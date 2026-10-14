package usermapping

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func TimeFormatFromDomain(value domaintypes.TimeFormat) gqlmodel.TimeFormat {
	if value == domaintypes.TimeFormat24Hour {
		return gqlmodel.TimeFormatTwentyFourHour
	}

	return gqlmodel.TimeFormatTwelveHour
}

func TimeFormatToDomain(value gqlmodel.TimeFormat) (domaintypes.TimeFormat, error) {
	switch value {
	case gqlmodel.TimeFormatTwelveHour:
		return domaintypes.TimeFormat12Hour, nil
	case gqlmodel.TimeFormatTwentyFourHour:
		return domaintypes.TimeFormat24Hour, nil
	default:
		return "", errortypes.NewValidationError(
			"timeFormat",
			errortypes.ErrInvalid,
			"Time format must be either 12-hour or 24-hour",
		)
	}
}
