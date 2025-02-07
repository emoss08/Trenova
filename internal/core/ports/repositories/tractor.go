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

type AssignmentOptions struct {
	TractorID pulid.ID `json:"tractorId"`
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
}

type AssignmentResponse struct {
	PrimaryWorkerID   pulid.ID  `json:"primaryWorkerId"`
	SecondaryWorkerID *pulid.ID `json:"secondaryWorkerId"`
}

type TractorRepository interface {
	Assignment(ctx context.Context, opts AssignmentOptions) (*AssignmentResponse, error)
	List(ctx context.Context, opts *ListTractorOptions) (*ports.ListResult[*tractor.Tractor], error)
	GetByID(ctx context.Context, opts GetTractorByIDOptions) (*tractor.Tractor, error)
	Create(ctx context.Context, t *tractor.Tractor) (*tractor.Tractor, error)
	Update(ctx context.Context, t *tractor.Tractor) (*tractor.Tractor, error)
}
