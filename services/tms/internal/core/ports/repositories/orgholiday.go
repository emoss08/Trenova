package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListOrgHolidaysRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// From/To bound one-off holidays by date; recurring holidays are always
	// returned so a calendar can be built for any year.
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type GetOrgHolidayByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type OrgHolidayRepository interface {
	List(ctx context.Context, req *ListOrgHolidaysRequest) ([]*worker.OrgHoliday, error)
	GetByID(ctx context.Context, req *GetOrgHolidayByIDRequest) (*worker.OrgHoliday, error)
	Create(ctx context.Context, entity *worker.OrgHoliday) (*worker.OrgHoliday, error)
	Update(ctx context.Context, entity *worker.OrgHoliday) (*worker.OrgHoliday, error)
	Delete(ctx context.Context, req *GetOrgHolidayByIDRequest) error
}
