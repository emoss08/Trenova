package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetReportByIDRequest struct {
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	ReportID   pulid.ID
	ExpandData bool
}

type ListReportRequest struct {
	Filter *pagination.QueryOptions
}

type ReportRepository interface {
	Create(ctx context.Context, rpt *report.Report) error
	Get(ctx context.Context, req GetReportByIDRequest) (*report.Report, error)
	Update(ctx context.Context, rpt *report.Report) error
	Delete(ctx context.Context, id pulid.ID) error
	List(ctx context.Context, req *ListReportRequest) (*pagination.ListResult[*report.Report], error)
}
