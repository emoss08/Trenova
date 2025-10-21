package hazmatsegregationrule

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/hazmatsegregationrulevalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.HazmatSegregationRuleRepository
	AuditService services.AuditService
	Validator    *hazmatsegregationrulevalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.HazmatSegregationRuleRepository
	as   services.AuditService
	v    *hazmatsegregationrulevalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.hazmatsegregationrule"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.ListHazmatSegregationRuleRequest,
) (*pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
	return s.repo.List(ctx, opts)
}

func (s *Service) Get(
	ctx context.Context,
	opts *repositories.GetHazmatSegregationRuleByIDRequest,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	return s.repo.GetByID(ctx, opts)
}

func (s *Service) Create(
	ctx context.Context,
	hsr *hazmatsegregationrule.HazmatSegregationRule,
	userID pulid.ID,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("hazmatSegregationRuleID", hsr.ID.String()),
		zap.String("buID", hsr.BusinessUnitID.String()),
		zap.String("orgID", hsr.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, hsr); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, hsr)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHazmatSegregationRule,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Hazmat Segregation Rule created"),
	)
	if err != nil {
		log.Error("failed to log action", zap.Error(err))
		return nil, err
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	hsr *hazmatsegregationrule.HazmatSegregationRule,
	userID pulid.ID,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("hazmatSegregationRuleID", hsr.ID.String()),
		zap.String("buID", hsr.BusinessUnitID.String()),
		zap.String("orgID", hsr.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, hsr); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetHazmatSegregationRuleByIDRequest{
		ID:    hsr.ID,
		OrgID: hsr.OrganizationID,
		BuID:  hsr.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, hsr)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHazmatSegregationRule,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Hazmat Segregation Rule updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log action", zap.Error(err))
		return nil, err
	}

	return updatedEntity, nil
}
