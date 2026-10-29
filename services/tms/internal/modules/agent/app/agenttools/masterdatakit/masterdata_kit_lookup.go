package masterdatakit

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/shared/pulid"
)

type StateReader interface {
	GetByAbbreviation(ctx context.Context, abbreviation string) (*usstate.UsState, error)
}

func IDText(id pulid.ID) string {
	if id.IsNil() {
		return ""
	}

	return id.String()
}
