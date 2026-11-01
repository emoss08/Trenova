package dothazmatreferenceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/dothazmatreference"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DotHazmatReferenceRepository interface {
	GetByIDs(
		ctx context.Context,
		ids []pulid.ID,
	) ([]*dothazmatreference.DotHazmatReference, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*dothazmatreference.DotHazmatReference], error)
}
