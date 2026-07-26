package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetReportDefinitionRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
}

type ListReportDefinitionsRequest struct {
	TenantInfo pagination.TenantInfo
	Statuses   []report.DefinitionStatus
	OwnerID    pulid.ID
	Limit      int
	Offset     int
}

type DeleteReportDefinitionRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
}

type GetReportRevisionRequest struct {
	TenantInfo pagination.TenantInfo
	RevisionID pulid.ID
}

type ListReportRevisionsRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	Limit        int
}

type ListReportDefinitionConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"-"`
	// ViewerID scopes results to definitions the viewer may see:
	// shared definitions plus the viewer's own.
	ViewerID pulid.ID `json:"-"`
}

type ListReportRunConnectionRequest struct {
	Filter       *pagination.QueryOptions `json:"filter"`
	Cursor       pagination.CursorInfo    `json:"-"`
	DefinitionID pulid.ID                 `json:"-"`
	RequestedBy  pulid.ID                 `json:"-"`
	Statuses     []report.RunStatus       `json:"-"`
}

type ReportDefinitionRepository interface {
	ListConnection(
		ctx context.Context,
		req *ListReportDefinitionConnectionRequest,
	) (*pagination.CursorListResult[*report.ReportDefinition], error)
	Create(
		ctx context.Context,
		entity *report.ReportDefinition,
		createdBy pulid.ID,
	) (*report.ReportDefinition, error)
	Update(
		ctx context.Context,
		entity *report.ReportDefinition,
		updatedBy pulid.ID,
	) (*report.ReportDefinition, error)
	UpdateStatus(
		ctx context.Context,
		req *GetReportDefinitionRequest,
		status report.DefinitionStatus,
		diagnostics []string,
	) error
	GetByID(
		ctx context.Context,
		req *GetReportDefinitionRequest,
	) (*report.ReportDefinition, error)
	List(
		ctx context.Context,
		req *ListReportDefinitionsRequest,
	) ([]*report.ReportDefinition, error)
	Delete(ctx context.Context, req *DeleteReportDefinitionRequest) error
	GetRevision(
		ctx context.Context,
		req *GetReportRevisionRequest,
	) (*report.ReportDefinitionRevision, error)
	ListRevisions(
		ctx context.Context,
		req *ListReportRevisionsRequest,
	) ([]*report.ReportDefinitionRevision, error)
}

type GetReportDashboardRequest struct {
	TenantInfo  pagination.TenantInfo
	DashboardID pulid.ID
}

type ListReportDashboardsRequest struct {
	TenantInfo pagination.TenantInfo
	// ViewerID scopes results to dashboards the viewer may see: shared
	// dashboards plus the viewer's own.
	ViewerID pulid.ID
	Limit    int
	Offset   int
}

type ReportDashboardRepository interface {
	Create(ctx context.Context, entity *report.Dashboard) (*report.Dashboard, error)
	Update(ctx context.Context, entity *report.Dashboard) (*report.Dashboard, error)
	GetByID(
		ctx context.Context,
		req *GetReportDashboardRequest,
	) (*report.Dashboard, error)
	List(
		ctx context.Context,
		req *ListReportDashboardsRequest,
	) ([]*report.Dashboard, error)
	Delete(ctx context.Context, req *GetReportDashboardRequest) error
}

type GetReportRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
}

type ListReportRunsRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	RequestedBy  pulid.ID
	Statuses     []report.RunStatus
	Limit        int
	Offset       int
}

type CountActiveReportRunsRequest struct {
	TenantInfo pagination.TenantInfo
}

type ActiveReportRunCounts struct {
	Running int
	Queued  int
}

type ListExpiredReportRunsRequest struct {
	CutoffUnix int64
	Limit      int
}

type ListStaleReportRunsRequest struct {
	Statuses          []report.RunStatus
	UpdatedBeforeUnix int64
	Limit             int
}

type ReportRunRepository interface {
	ListConnection(
		ctx context.Context,
		req *ListReportRunConnectionRequest,
	) (*pagination.CursorListResult[*report.ReportRun], error)
	Create(ctx context.Context, entity *report.ReportRun) (*report.ReportRun, error)
	Update(ctx context.Context, entity *report.ReportRun) (*report.ReportRun, error)
	GetByID(ctx context.Context, req *GetReportRunRequest) (*report.ReportRun, error)
	List(ctx context.Context, req *ListReportRunsRequest) ([]*report.ReportRun, error)
	CountActive(
		ctx context.Context,
		req *CountActiveReportRunsRequest,
	) (*ActiveReportRunCounts, error)
	ListExpired(
		ctx context.Context,
		req *ListExpiredReportRunsRequest,
	) ([]*report.ReportRun, error)
	ListStale(
		ctx context.Context,
		req *ListStaleReportRunsRequest,
	) ([]*report.ReportRun, error)
}

type GetReportScheduleRequest struct {
	TenantInfo pagination.TenantInfo
	ScheduleID pulid.ID
}

type ListReportSchedulesRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	EnabledOnly  bool
	Limit        int
	Offset       int
}

type ReportScheduleRepository interface {
	Create(ctx context.Context, entity *report.ReportSchedule) (*report.ReportSchedule, error)
	Update(ctx context.Context, entity *report.ReportSchedule) (*report.ReportSchedule, error)
	GetByID(ctx context.Context, req *GetReportScheduleRequest) (*report.ReportSchedule, error)
	List(
		ctx context.Context,
		req *ListReportSchedulesRequest,
	) ([]*report.ReportSchedule, error)
	ListAllEnabled(ctx context.Context) ([]*report.ReportSchedule, error)
	ListDue(ctx context.Context, nowUnix int64, limit int) ([]*report.ReportSchedule, error)
	Delete(ctx context.Context, req *GetReportScheduleRequest) error
}

type GetReportViewRequest struct {
	TenantInfo pagination.TenantInfo
	ViewID     pulid.ID
}

// ListReportViewsRequest scopes the picker. VisibleToID is the caller: a view
// is listed when it is shared or when they own it, so one person's working set
// never appears in everyone else's list.
type ListReportViewsRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	VisibleToID  pulid.ID
	Limit        int
	Offset       int
}

// RecordReportViewRunRequest stamps usage on a view. It is deliberately not an
// Update: run bookkeeping must not collide with a concurrent edit by bumping
// the optimistic-lock version out from under it.
type RecordReportViewRunRequest struct {
	TenantInfo pagination.TenantInfo
	ViewID     pulid.ID
	RanAt      int64
}

type ReportViewRepository interface {
	Create(ctx context.Context, entity *report.ReportView) (*report.ReportView, error)
	Update(ctx context.Context, entity *report.ReportView) (*report.ReportView, error)
	GetByID(ctx context.Context, req *GetReportViewRequest) (*report.ReportView, error)
	List(ctx context.Context, req *ListReportViewsRequest) ([]*report.ReportView, error)
	RecordRun(ctx context.Context, req *RecordReportViewRunRequest) error
	Delete(ctx context.Context, req *GetReportViewRequest) error
}
