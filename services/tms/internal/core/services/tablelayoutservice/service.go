package tablelayoutservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tablelayout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.TableLayoutRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.TableLayoutRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.tablelayout"),
		repo: p.Repo,
	}
}

type Request struct {
	TenantInfo pagination.TenantInfo
	Resource   string
}

type SaveRequest struct {
	Request

	Layout *tablelayout.Layout
}

func (s *Service) Get(ctx context.Context, req *Request) (*tablelayout.TableLayout, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	entity, found, err := s.repo.Get(ctx, &repositories.TableLayoutRequest{
		TenantInfo: req.TenantInfo,
		Resource:   req.Resource,
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil //nolint:nilnil // a person who never arranged the table has no layout
	}

	return entity, nil
}

func (s *Service) Save(ctx context.Context, req *SaveRequest) (*tablelayout.TableLayout, error) {
	if err := validateRequest(&req.Request); err != nil {
		return nil, err
	}

	entity := &tablelayout.TableLayout{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.TenantInfo.UserID,
		Resource:       req.Resource,
		Layout:         req.Layout,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if err := s.ensureRoomFor(ctx, &req.Request); err != nil {
		return nil, err
	}

	saved, err := s.repo.Upsert(ctx, entity)
	if err != nil {
		s.l.Error("failed to save table layout",
			zap.String("resource", req.Resource),
			zap.Error(err),
		)
		return nil, err
	}

	return saved, nil
}

func (s *Service) Reset(ctx context.Context, req *Request) error {
	if err := validateRequest(req); err != nil {
		return err
	}

	return s.repo.Delete(ctx, &repositories.TableLayoutRequest{
		TenantInfo: req.TenantInfo,
		Resource:   req.Resource,
	})
}

func (s *Service) ensureRoomFor(ctx context.Context, req *Request) error {
	_, found, err := s.repo.Get(ctx, &repositories.TableLayoutRequest{
		TenantInfo: req.TenantInfo,
		Resource:   req.Resource,
	})
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	count, err := s.repo.CountForUser(ctx, req.TenantInfo)
	if err != nil {
		return err
	}
	if count >= tablelayout.MaxLayoutsPerUser {
		return errortypes.NewBusinessError(
			fmt.Sprintf(
				"You have saved layouts for %d tables, the most allowed. Reset one you no longer use.",
				tablelayout.MaxLayoutsPerUser,
			),
		)
	}

	return nil
}

func validateRequest(req *Request) error {
	if req.TenantInfo.UserID.IsNil() {
		return errortypes.NewAuthorizationError("Table layouts belong to a signed-in person")
	}

	multiErr := errortypes.NewMultiError()
	probe := &tablelayout.TableLayout{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.TenantInfo.UserID,
		Resource:       req.Resource,
	}
	probe.ValidateKey(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
