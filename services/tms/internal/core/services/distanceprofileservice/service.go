package distanceprofileservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.DistanceProfileRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.DistanceProfileRepository
	validator    *Validator
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.distance-profile"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListDistanceProfileRequest,
) (*pagination.ListResult[*distanceprofile.DistanceProfile], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetDistanceProfileByIDRequest,
) (*distanceprofile.DistanceProfile, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) EnsureDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*distanceprofile.DistanceProfile, error) {
	return s.repo.EnsureDefault(ctx, tenantInfo)
}

func (s *Service) Create(
	ctx context.Context,
	entity *distanceprofile.DistanceProfile,
	userID pulid.ID,
) (*distanceprofile.DistanceProfile, error) {
	entity.ApplyDefaults()
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(created, nil, permission.OpCreate, userID, "Distance profile created")
	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *distanceprofile.DistanceProfile,
	userID pulid.ID,
) (*distanceprofile.DistanceProfile, error) {
	entity.ApplyDefaults()
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetDistanceProfileByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(updated, original, permission.OpUpdate, userID, "Distance profile updated")
	return updated, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteDistanceProfileRequest,
	userID pulid.ID,
) error {
	existing, err := s.repo.GetByID(ctx, repositories.GetDistanceProfileByIDRequest(req))
	if err != nil {
		return err
	}
	if existing.IsDefault {
		return errortypes.NewValidationError(
			"isDefault",
			errortypes.ErrInvalid,
			"Default profile cannot be deleted",
		)
	}
	if err = s.repo.Delete(ctx, req); err != nil {
		return err
	}
	s.logAudit(existing, existing, permission.OpDelete, userID, "Distance profile deleted")
	return nil
}

func (s *Service) SetDefault(
	ctx context.Context,
	req repositories.GetDistanceProfileByIDRequest,
	userID pulid.ID,
) (*distanceprofile.DistanceProfile, error) {
	updated, err := s.repo.SetDefault(ctx, req)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, nil, permission.OpUpdate, userID, "Distance profile set as default")
	return updated, nil
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.DistanceProfileSelectOptionsRequest,
) (*pagination.ListResult[*distanceprofile.DistanceProfile], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) logAudit(
	current *distanceprofile.DistanceProfile,
	previous *distanceprofile.DistanceProfile,
	op permission.Operation,
	userID pulid.ID,
	comment string,
) {
	if current == nil {
		return
	}
	params := &services.LogActionParams{
		Resource:       permission.ResourceDistanceProfile,
		ResourceID:     current.ID.String(),
		Operation:      op,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	opts := []services.LogOption{auditservice.WithComment(comment)}
	if previous != nil && op == permission.OpUpdate {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log distance profile audit action", zap.Error(err))
	}
}
