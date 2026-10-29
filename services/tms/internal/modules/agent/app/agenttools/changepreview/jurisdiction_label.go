package changepreview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/shared/pulid"
)

type JurisdictionLabeler interface {
	GetJurisdiction(ctx context.Context, id pulid.ID) (*ifta.Jurisdiction, error)
}

func JurisdictionLabel(
	ctx context.Context,
	jurisdictions JurisdictionLabeler,
	id pulid.ID,
) string {
	if id.IsNil() {
		return ""
	}
	jurisdiction, err := jurisdictions.GetJurisdiction(ctx, id)
	if err != nil || jurisdiction == nil {
		return id.String()
	}

	return jurisdiction.Code + " (" + jurisdiction.Name + ")"
}
