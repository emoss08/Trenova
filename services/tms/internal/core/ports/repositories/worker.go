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

var WorkerPTOFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"status":           true,
		"type":             true,
		"startDate":        true,
		"endDate":          true,
		"worker.firstName": true,
		"worker.lastName":  true,
	},
	SortableFields: map[string]bool{
		"status":           true,
		"type":             true,
		"startDate":        true,
		"endDate":          true,
		"worker.firstName": true,
		"worker.lastName":  true,
	},
	FieldMap: map[string]string{
		"status":    "status",
		"type":      "type",
		"startDate": "start_date",
		"endDate":   "end_date",
	},
	EnumMap: map[string]bool{
		"status": true,
		"type":   true,
	},
	NestedFields: map[string]ports.NestedFieldDefinition{
		"worker.firstName": {
			DatabaseField: "wrk.first_name",
			RequiredJoins: []ports.JoinDefinition{
				{
					Table:     "workers",
					Alias:     "wrk",
					Condition: "wpto.worker_id = wrk.id",
					JoinType:  ports.JoinTypeLeft,
				},
			},
			IsEnum: false,
		},
		"worker.lastName": {
			DatabaseField: "wrk.last_name",
			RequiredJoins: []ports.JoinDefinition{
				{
					Table:     "workers",
					Alias:     "wrk",
					Condition: "wpto.worker_id = wrk.id",
					JoinType:  ports.JoinTypeLeft,
				},
			},
			IsEnum: false,
		},
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

type ListWorkerRequest struct {
	Filter              *ports.QueryOptions `json:"filter"              query:"filter"`
	WorkerFilterOptions `json:"workerFilterOptions" query:"workerFilterOptions"`
}

func BuildWorkerPTOListOptions(
	filter *ports.QueryOptions,
) *ListWorkerPTORequest {
	return &ListWorkerPTORequest{
		Filter: filter,
	}
}

type ListWorkerPTORequest struct {
	Filter *ports.QueryOptions `json:"filter" query:"filter"`
}

type GetWorkerByIDRequest struct {
	WorkerID      pulid.ID            `json:"workerId"      query:"workerId"`
	BuID          pulid.ID            `json:"buId"          query:"buId"`
	OrgID         pulid.ID            `json:"orgId"         query:"orgId"`
	UserID        pulid.ID            `json:"userId"        query:"userId"`
	FilterOptions WorkerFilterOptions `json:"filterOptions" query:"filterOptions"`
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
	Status      string `json:"status"      query:"status"`
	Type        string `json:"type"        query:"type"`
	StartDate   int64  `json:"startDate"   query:"startDate"`
	EndDate     int64  `json:"endDate"     query:"endDate"`
	WorkerID    string `json:"workerId"    query:"workerId"`
	FleetCodeID string `json:"fleetCodeId" query:"fleetCodeId"`
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

type PTOChartDataRequest struct {
	Filter    *ports.LimitOffsetQueryOptions `json:"filter"    query:"filter"`
	StartDate int64                          `json:"startDate" query:"startDate"`
	EndDate   int64                          `json:"endDate"   query:"endDate"`
	Type      string                         `json:"type"      query:"type"`
	Timezone  string                         `json:"timezone"  query:"timezone"`
	WorkerID  string                         `json:"workerId"  query:"workerId"`
}

type PTOChartDataPoint struct {
	Date        string                    `json:"date"`
	Vacation    int                       `json:"vacation"`
	Sick        int                       `json:"sick"`
	Holiday     int                       `json:"holiday"`
	Bereavement int                       `json:"bereavement"`
	Maternity   int                       `json:"maternity"`
	Paternity   int                       `json:"paternity"`
	Personal    int                       `json:"personal"`
	Workers     map[string][]WorkerDetail `json:"workers"`
}

type WorkerDetail struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	PTOType   string `json:"ptoType"`
}

type PTOCalendarDataRequest struct {
	Filter    *ports.LimitOffsetQueryOptions `json:"filter"    query:"filter"`
	StartDate int64                          `json:"startDate" query:"startDate"`
	EndDate   int64                          `json:"endDate"   query:"endDate"`
	Type      string                         `json:"type"      query:"type"`
}

type PTOCalendarEvent struct {
	ID         string `json:"id"               bun:"id"`
	WorkerID   string `json:"workerId"         bun:"worker_id"`
	WorkerName string `json:"workerName"       bun:"worker_name"`
	StartDate  int64  `json:"startDate"        bun:"start_date"`
	EndDate    int64  `json:"endDate"          bun:"end_date"`
	Type       string `json:"type"             bun:"type"`
	Status     string `json:"status"           bun:"status"`
	Reason     string `json:"reason,omitempty" bun:"reason"`
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
	ListWorkerPTO(
		ctx context.Context,
		req *ListWorkerPTORequest,
	) (*ports.ListResult[*worker.WorkerPTO], error)
	GetPTOChartData(
		ctx context.Context,
		req *PTOChartDataRequest,
	) ([]*PTOChartDataPoint, error)
	GetPTOCalendarData(
		ctx context.Context,
		req *PTOCalendarDataRequest,
	) ([]*PTOCalendarEvent, error)
	CreateWorkerPTO(
		ctx context.Context,
		pto *worker.WorkerPTO,
	) (*worker.WorkerPTO, error)
}
