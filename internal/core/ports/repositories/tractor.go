package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ListTractorOptions struct {
	Filter                  *ports.LimitOffsetQueryOptions
	IncludeWorkerDetails    bool `query:"includeWorkerDetails"`
	IncludeEquipmentDetails bool `query:"includeEquipmentDetails"`
	IncludeFleetDetails     bool `query:"includeFleetDetails"`
}

type GetTractorByIDOptions struct {
	ID                      pulid.ID
	OrgID                   pulid.ID
	BuID                    pulid.ID
	UserID                  pulid.ID
	IncludeWorkerDetails    bool `query:"includeWorkerDetails"`
	IncludeEquipmentDetails bool `query:"includeEquipmentDetails"`
	IncludeFleetDetails     bool `query:"includeFleetDetails"`
}

type TractorRepository interface {
	List(ctx context.Context, opts *ListTractorOptions) (*ports.ListResult[*tractor.Tractor], error)
	GetByID(ctx context.Context, opts GetTractorByIDOptions) (*tractor.Tractor, error)
	Create(ctx context.Context, t *tractor.Tractor) (*tractor.Tractor, error)
	Update(ctx context.Context, t *tractor.Tractor) (*tractor.Tractor, error)
}
