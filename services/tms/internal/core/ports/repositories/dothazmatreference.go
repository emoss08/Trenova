package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dothazmatreference"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDotHazmatReferenceByIDRequest struct {
	DotHazmatReferenceID pulid.ID `json:"dotHazmatReferenceId"`
}

type DotHazmatReferenceRepository interface {
	GetByID(
		ctx context.Context,
		req GetDotHazmatReferenceByIDRequest,
	) (*dothazmatreference.DotHazmatReference, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*dothazmatreference.DotHazmatReference], error)
}
