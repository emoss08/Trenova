package base

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func DerefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func PulidsToStrings(ids []pulid.ID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func ParsePulids(field string, raw []string) ([]pulid.ID, error) {
	out := make([]pulid.ID, 0, len(raw))
	for i, value := range raw {
		id, err := pulid.MustParse(value)
		if err != nil {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("%s[%d]", field, i), errortypes.ErrInvalid, "Invalid identifier",
			)
		}
		out = append(out, id)
	}
	return out, nil
}

func DerefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
