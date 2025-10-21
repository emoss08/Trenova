package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type TractorFilterOptions struct {
	IncludeWorkerDetails    bool   `query:"includeWorkerDetails"`
	IncludeEquipmentDetails bool   `query:"includeEquipmentDetails"`
	IncludeFleetDetails     bool   `query:"includeFleetDetails"`
	Status                  string `query:"status"`
}

type ListTractorRequest struct {
	Filter        *pagination.QueryOptions
	FilterOptions TractorFilterOptions `query:"filterOptions"`
}

type GetTractorByIDRequest struct {
	TractorID     pulid.ID
	OrgID         pulid.ID
	BuID          pulid.ID
	UserID        pulid.ID
	FilterOptions TractorFilterOptions `query:"filterOptions"`
}

type GetTractorByPrimaryWorkerIDRequest struct {
	WorkerID pulid.ID
	OrgID    pulid.ID
	BuID     pulid.ID
}

type TractorAssignmentRequest struct {
	TractorID pulid.ID `json:"tractorId"`
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
}

type AssignmentResponse struct {
	PrimaryWorkerID   pulid.ID  `json:"primaryWorkerId"`
	SecondaryWorkerID *pulid.ID `json:"secondaryWorkerId"`
}

type TractorRepository interface {
	Assignment(ctx context.Context, req TractorAssignmentRequest) (*AssignmentResponse, error)
	List(
		ctx context.Context,
		req *ListTractorRequest,
	) (*pagination.ListResult[*tractor.Tractor], error)
	GetByID(ctx context.Context, req *GetTractorByIDRequest) (*tractor.Tractor, error)
	GetByPrimaryWorkerID(
		ctx context.Context,
		req GetTractorByPrimaryWorkerIDRequest,
	) (*tractor.Tractor, error)
	Create(ctx context.Context, t *tractor.Tractor) (*tractor.Tractor, error)
	Update(ctx context.Context, t *tractor.Tractor) (*tractor.Tractor, error)
}
