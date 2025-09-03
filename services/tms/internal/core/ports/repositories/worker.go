/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

var WorkerFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"status":    true,
		"firstName": true,
		"lastName":  true,
		"type":      true,
	},
	SortableFields: map[string]bool{
		"status":    true,
		"firstName": true,
		"lastName":  true,
	},
	FieldMap: map[string]string{
		"firstName": "first_name",
		"lastName":  "last_name",
		"status":    "status",
	},
	EnumMap: map[string]bool{
		"status": true,
		"type":   true,
	},
}

type WorkerFilterOptions struct {
	Status         string `query:"status"`
	IncludeProfile bool   `query:"includeProfile"`
	IncludePTO     bool   `query:"includePTO"`
}

func BuildWorkerListOptions(
	filter *ports.QueryOptions,
	additionalOpts *ListWorkerRequest,
) *ListWorkerRequest {
	return &ListWorkerRequest{
		Filter:              filter,
		WorkerFilterOptions: additionalOpts.WorkerFilterOptions,
	}
}

type GetWorkerByIDRequest struct {
	WorkerID      pulid.ID            `json:"workerId"      query:"workerId"`
	BuID          pulid.ID            `json:"buId"          query:"buId"`
	OrgID         pulid.ID            `json:"orgId"         query:"orgId"`
	UserID        pulid.ID            `json:"userId"        query:"userId"`
	FilterOptions WorkerFilterOptions `json:"filterOptions" query:"filterOptions"`
}

type ListWorkerRequest struct {
	Filter              *ports.QueryOptions `json:"filter"              query:"filter"`
	WorkerFilterOptions `json:"workerFilterOptions" query:"workerFilterOptions"`
}

type UpdateWorkerOptions struct {
	OrgID pulid.ID `json:"orgId" query:"orgId"`
	BuID  pulid.ID `json:"buId"  query:"buId"`
}

type GetWorkerPTORequest struct {
	PtoID    pulid.ID `json:"ptoId"    query:"ptoId"`
	WorkerID pulid.ID `json:"workerId" query:"workerId"`
	BuID     pulid.ID `json:"buId"     query:"buId"`
	OrgID    pulid.ID `json:"orgId"    query:"orgId"`
}

type ListWorkerPTOFilterOptions struct {
	Status    string `json:"status"    query:"status"`
	Type      string `json:"type"      query:"type"`
	StartDate int64  `json:"startDate" query:"startDate"`
	EndDate   int64  `json:"endDate"   query:"endDate"`
}

type ListUpcomingWorkerPTORequest struct {
	Filter                     *ports.LimitOffsetQueryOptions `json:"filter"        query:"filter"`
	ListWorkerPTOFilterOptions `json:"filterOptions" query:"filterOptions"`
}

type ApprovePTORequest struct {
	PtoID      pulid.ID `json:"ptoId"      query:"ptoId"`
	BuID       pulid.ID `json:"buId"       query:"buId"`
	OrgID      pulid.ID `json:"orgId"      query:"orgId"`
	ApproverID pulid.ID `json:"approverId" query:"approverId"`
}

type RejectPTORequest struct {
	PtoID      pulid.ID `json:"ptoId"      query:"ptoId"`
	BuID       pulid.ID `json:"buId"       query:"buId"`
	OrgID      pulid.ID `json:"orgId"      query:"orgId"`
	RejectorID pulid.ID `json:"rejectorId" query:"rejectorId"`
	Reason     string   `json:"reason"     query:"reason"`
}

type WorkerRepository interface {
	List(ctx context.Context, req *ListWorkerRequest) (*ports.ListResult[*worker.Worker], error)
	GetByID(ctx context.Context, req *GetWorkerByIDRequest) (*worker.Worker, error)
	Create(ctx context.Context, wrk *worker.Worker) (*worker.Worker, error)
	Update(ctx context.Context, wrk *worker.Worker) (*worker.Worker, error)
	GetWorkerPTO(
		ctx context.Context,
		req *GetWorkerPTORequest,
	) (*worker.WorkerPTO, error)
	ListUpcomingPTO(
		ctx context.Context,
		req *ListUpcomingWorkerPTORequest,
	) (*ports.ListResult[*worker.WorkerPTO], error)
	ApprovePTO(ctx context.Context, req *ApprovePTORequest) error
	RejectPTO(ctx context.Context, req *RejectPTORequest) error
}
