package inboundmessageservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateViewRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Name       string
	Query      string
}

type RenameViewRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ID         pulid.ID
	Name       string
}

type ViewRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ID         pulid.ID
}

func (s *Service) MyViews(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) ([]*triage.View, error) {
	return s.viewRepo.List(ctx, inboundmessageports.ListViewsRequest{
		TenantInfo: tenantInfo,
		UserID:     userID,
	})
}

func (s *Service) CreateView(ctx context.Context, req *CreateViewRequest) (*triage.View, error) {
	view := &triage.View{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.UserID,
		Name:           strings.TrimSpace(req.Name),
		Query:          strings.TrimSpace(req.Query),
	}
	multiErr := errortypes.NewMultiError()
	view.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.viewRepo.Create(ctx, view, triage.MaxViewsPerUser)
}

func (s *Service) RenameView(ctx context.Context, req *RenameViewRequest) (*triage.View, error) {
	view, err := s.viewRepo.Get(ctx, inboundmessageports.GetViewRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}

	view.Name = strings.TrimSpace(req.Name)
	multiErr := errortypes.NewMultiError()
	view.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	view.UpdatedAt = s.now()

	return s.viewRepo.Rename(ctx, view)
}

func (s *Service) DeleteView(ctx context.Context, req *ViewRequest) error {
	return s.viewRepo.Delete(ctx, inboundmessageports.GetViewRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
		ID:         req.ID,
	})
}
