package distancecontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/integration"
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

	Logger              *zap.Logger
	Repo                repositories.DistanceControlRepository
	DistanceProfileRepo repositories.DistanceProfileRepository
	AuditService        services.AuditService
}

type Service struct {
	l                   *zap.Logger
	repo                repositories.DistanceControlRepository
	distanceProfileRepo repositories.DistanceProfileRepository
	auditService        services.AuditService
}

func New(p Params) services.DistanceControlService {
	return &Service{
		l:                   p.Logger.Named("service.distance-control"),
		repo:                p.Repo,
		distanceProfileRepo: p.DistanceProfileRepo,
		auditService:        p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*distancecontrol.DistanceControl, error) {
	return s.repo.EnsureDefault(ctx, tenantInfo)
}

func (s *Service) Update(
	ctx context.Context,
	entity *distancecontrol.DistanceControl,
	userID pulid.ID,
) (*distancecontrol.DistanceControl, error) {
	entity.ApplyDefaults()
	if multiErr := s.validate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.Get(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, original, userID)
	return updated, nil
}

func (s *Service) validate(
	ctx context.Context,
	entity *distancecontrol.DistanceControl,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	profileFields := map[string]pulid.ID{
		"loadedMoveDistanceProfileId":                  entity.LoadedMoveDistanceProfileID,
		"emptyMoveDistanceProfileId":                   entity.EmptyMoveDistanceProfileID,
		"payDistanceProfileId":                         entity.PayDistanceProfileID,
		"billingDistanceProfileId":                     entity.BillingDistanceProfileID,
		"fuelDistanceProfileId":                        entity.FuelDistanceProfileID,
		"etaOutOfRouteDistanceProfileId":               entity.EtaOutOfRouteDistanceProfileID,
		"distanceCalculatorShortestDistanceProfileId":  entity.DistanceCalculatorShortestDistanceProfileID,
		"distanceCalculatorPracticalDistanceProfileId": entity.DistanceCalculatorPracticalDistanceProfileID,
	}
	for field, id := range profileFields {
		if id.IsNil() {
			continue
		}
		profile, err := s.distanceProfileRepo.GetByID(ctx, repositories.GetDistanceProfileByIDRequest{
			ID: id,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
		if err != nil {
			multiErr.Add(field, errortypes.ErrInvalid, "Distance profile must exist in the same tenant")
			continue
		}
		if profile.Status != distanceprofile.StatusActive || profile.Provider != integration.TypePCMiler {
			multiErr.Add(field, errortypes.ErrInvalid, "Distance profile must be an active PC*Miler profile")
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) logAudit(
	current *distancecontrol.DistanceControl,
	previous *distancecontrol.DistanceControl,
	userID pulid.ID,
) {
	if current == nil {
		return
	}
	params := &services.LogActionParams{
		Resource:       permission.ResourceDistanceControl,
		ResourceID:     current.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	opts := []services.LogOption{auditservice.WithComment("Distance control updated")}
	if previous != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log distance control audit action", zap.Error(err))
	}
}
