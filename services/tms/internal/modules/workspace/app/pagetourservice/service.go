package pagetourservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/pagetourports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   pagetourports.Repository
	Clock  timeutils.Clock
}

type Service struct {
	l     *zap.Logger
	repo  pagetourports.Repository
	clock timeutils.Clock
}

func New(p Params) *Service {
	return &Service{
		l:     p.Logger.Named("service.pagetour"),
		repo:  p.Repo,
		clock: p.Clock,
	}
}

type RecordRequest struct {
	TenantInfo  pagination.TenantInfo
	Tour        string
	TourVersion int32
	Outcome     pagetour.Outcome
}

func (s *Service) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*pagetour.View, error) {
	return s.repo.List(ctx, tenantInfo)
}

func (s *Service) Record(ctx context.Context, req *RecordRequest) (*pagetour.View, error) {
	entity := &pagetour.View{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.TenantInfo.UserID,
		Tour:           req.Tour,
		TourVersion:    req.TourVersion,
		Outcome:        req.Outcome,
		SeenAt:         s.clock(),
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if err := s.ensureRoomFor(ctx, req); err != nil {
		return nil, err
	}

	saved, err := s.repo.Upsert(ctx, entity)
	if err != nil {
		s.l.Error("failed to record a page tour view",
			zap.String("tour", req.Tour),
			zap.Error(err),
		)
		return nil, err
	}

	return saved, nil
}

func (s *Service) ensureRoomFor(ctx context.Context, req *RecordRequest) error {
	seen, err := s.repo.Exists(ctx, &pagetourports.ViewRequest{
		TenantInfo: req.TenantInfo,
		Tour:       req.Tour,
	})
	if err != nil || seen {
		return err
	}

	count, err := s.repo.CountForUser(ctx, req.TenantInfo)
	if err != nil {
		return err
	}
	if count >= pagetour.MaxViewsPerUser {
		return errortypes.NewBusinessError(
			"You have already been shown every tour there is room to remember.",
		)
	}

	return nil
}
