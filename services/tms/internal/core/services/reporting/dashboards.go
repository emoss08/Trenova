package reporting

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type SaveDashboardRequest struct {
	Request

	DashboardID pulid.ID
	Name        string
	Description string
	Category    string
	Tags        []string
	Visibility  report.Visibility
	Layout      *report.DashboardLayout
	Version     int64
}

type GetDashboardRequest struct {
	Request

	DashboardID pulid.ID
}

type ListDashboardsRequest struct {
	Request

	Limit  int
	Offset int
}

func (s *Service) CreateDashboard(
	ctx context.Context,
	req *SaveDashboardRequest,
) (*report.Dashboard, error) {
	entity := &report.Dashboard{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		Tags:           req.Tags,
		OwnerID:        req.TenantInfo.UserID,
		Visibility:     defaultVisibility(req.Visibility),
		Layout:         defaultLayout(req.Layout),
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err := s.validateTileTargets(ctx, req.TenantInfo, entity.Layout); err != nil {
		return nil, err
	}

	created, err := s.dashboardRepo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create dashboard", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (s *Service) UpdateDashboard(
	ctx context.Context,
	req *SaveDashboardRequest,
) (*report.Dashboard, error) {
	existing, err := s.dashboardRepo.GetByID(ctx, &repositories.GetReportDashboardRequest{
		TenantInfo:  req.TenantInfo,
		DashboardID: req.DashboardID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the dashboard owner can modify this dashboard",
		)
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Category = req.Category
	existing.Tags = req.Tags
	existing.Visibility = defaultVisibility(req.Visibility)
	existing.Layout = defaultLayout(req.Layout)
	existing.Version = req.Version

	multiErr := errortypes.NewMultiError()
	existing.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err = s.validateTileTargets(ctx, req.TenantInfo, existing.Layout); err != nil {
		return nil, err
	}

	updated, err := s.dashboardRepo.Update(ctx, existing)
	if err != nil {
		s.l.Error("failed to update dashboard", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

func (s *Service) GetDashboard(
	ctx context.Context,
	req *GetDashboardRequest,
) (*report.Dashboard, error) {
	entity, err := s.dashboardRepo.GetByID(ctx, &repositories.GetReportDashboardRequest{
		TenantInfo:  req.TenantInfo,
		DashboardID: req.DashboardID,
	})
	if err != nil {
		return nil, err
	}
	if entity.Visibility != report.VisibilityShared &&
		entity.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewNotFoundError("Dashboard not found")
	}

	return entity, nil
}

func (s *Service) ListDashboards(
	ctx context.Context,
	req *ListDashboardsRequest,
) ([]*report.Dashboard, error) {
	return s.dashboardRepo.List(ctx, &repositories.ListReportDashboardsRequest{
		TenantInfo: req.TenantInfo,
		ViewerID:   req.TenantInfo.UserID,
		Limit:      req.Limit,
		Offset:     req.Offset,
	})
}

func (s *Service) DeleteDashboard(ctx context.Context, req *GetDashboardRequest) error {
	existing, err := s.dashboardRepo.GetByID(ctx, &repositories.GetReportDashboardRequest{
		TenantInfo:  req.TenantInfo,
		DashboardID: req.DashboardID,
	})
	if err != nil {
		return err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return errortypes.NewAuthorizationError(
			"Only the dashboard owner can delete this dashboard",
		)
	}

	return s.dashboardRepo.Delete(ctx, &repositories.GetReportDashboardRequest{
		TenantInfo:  req.TenantInfo,
		DashboardID: req.DashboardID,
	})
}

// validateTileTargets fails a save that points at a report the author cannot
// open, so a broken tile is caught at edit time instead of at view time.
func (s *Service) validateTileTargets(
	ctx context.Context,
	tenant pagination.TenantInfo,
	layout *report.DashboardLayout,
) error {
	multiErr := errortypes.NewMultiError()

	for i := range layout.Tiles {
		tile := &layout.Tiles[i]
		if !tile.Kind.NeedsReport() {
			continue
		}
		fieldPath := fmt.Sprintf("layout.tiles[%d]", i)

		if tile.CannedKey != "" {
			if _, ok := s.canned.Get(tile.CannedKey); !ok {
				multiErr.Add(fieldPath+".cannedKey", errortypes.ErrInvalid,
					fmt.Sprintf("Unknown canned report %q", tile.CannedKey))
			}
			continue
		}

		_, err := s.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
			TenantInfo:   tenant,
			DefinitionID: tile.DefinitionID,
		})
		if err != nil {
			multiErr.Add(fieldPath+".definitionId", errortypes.ErrInvalid,
				"This report is not available")
		}
	}

	validateDashboardFilters(multiErr, layout.Filters)

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// validateDashboardFilters resolves each filter against the catalog so a
// dashboard cannot be saved pointing at a field that does not exist or cannot
// be filtered — the tiles would silently ignore it otherwise.
func validateDashboardFilters(
	multiErr *errortypes.MultiError,
	filters []report.DashboardFilter,
) {
	for i := range filters {
		filter := &filters[i]
		fieldPath := fmt.Sprintf("layout.filters[%d]", i)

		entity, ok := reportcatalog.Default.Entity(filter.Entity)
		if !ok {
			multiErr.Add(fieldPath+".entity", errortypes.ErrInvalid,
				fmt.Sprintf("Unknown entity %q", filter.Entity))
			continue
		}

		_, path, err := reportcatalog.Default.ResolvePath(entity.Key, filter.Ref.Path)
		if err != nil {
			multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid, err.Error())
			continue
		}
		if path.CrossesToMany() {
			multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid,
				"Dashboard filters cannot cross a to-many relationship")
			continue
		}

		field, ok := path.Terminal(entity).Field(filter.Ref.Field)
		if !ok {
			multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid,
				fmt.Sprintf("Unknown field %q", filter.Ref.String()))
			continue
		}
		if !field.Filterable {
			multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid,
				fmt.Sprintf("Field %q is not filterable", field.Label))
		}
	}
}

func defaultLayout(layout *report.DashboardLayout) *report.DashboardLayout {
	if layout == nil {
		return &report.DashboardLayout{Tiles: []report.DashboardTile{}}
	}
	if layout.Tiles == nil {
		layout.Tiles = []report.DashboardTile{}
	}
	return layout
}
