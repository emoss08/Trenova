package reportdashboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) PlanDeleteDashboard(
	ctx context.Context,
	req *GetDashboardRequest,
) (*report.Dashboard, error) {
	existing, err := s.dashboardRepo.GetByID(ctx, &reportports.GetReportDashboardRequest{
		TenantInfo:  req.TenantInfo,
		DashboardID: req.DashboardID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the dashboard owner can delete this dashboard",
		)
	}

	return existing, nil
}

func NewDashboard(req *SaveDashboardRequest) *report.Dashboard {
	return &report.Dashboard{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		Tags:           req.Tags,
		OwnerID:        req.TenantInfo.UserID,
		Visibility:     reporting.DefaultVisibility(req.Visibility),
		Layout:         defaultLayout(req.Layout),
	}
}

func ApplyDashboardSave(existing *report.Dashboard, req *SaveDashboardRequest) {
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Category = req.Category
	existing.Tags = req.Tags
	existing.Visibility = reporting.DefaultVisibility(req.Visibility)
	existing.Layout = defaultLayout(req.Layout)
	existing.Version = req.Version
}
