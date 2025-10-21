package organization

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator/organizationvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.OrganizationRepository
	Validator    *organizationvalidator.Validator
	AuditService services.AuditService
}

type Service struct {
	l    *zap.Logger
	repo repositories.OrganizationRepository
	v    *organizationvalidator.Validator
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.organization"),
		repo: p.Repo,
		v:    p.Validator,
		as:   p.AuditService,
	}
}

func (s *Service) GetUserOrganizations(
	ctx context.Context,
	req *pagination.QueryOptions,
) (*pagination.ListResult[*tenant.Organization], error) {
	result, err := s.repo.GetUserOrganizations(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*tenant.Organization]{
		Items: result.Items,
		Total: result.Total,
	}, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Update(
	ctx context.Context,
	org *tenant.Organization,
	userID pulid.ID,
) (*tenant.Organization, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", org.ID.String()),
		zap.String("userID", userID.String()),
	)

	if err := s.v.Validate(ctx, org); err != nil {
		return nil, err
	}

	opts := repositories.GetOrganizationByIDRequest{
		OrgID:        org.ID,
		BuID:         org.BusinessUnitID,
		IncludeState: true,
		// IncludeBu:    true,
	}

	original, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		return nil, err
	}

	updatedOrg, err := s.repo.Update(ctx, org)
	if err != nil {
		log.Error("failed to update organization", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceOrganization,
			ResourceID:     org.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(org),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
		},
		audit.WithComment("Organization updated"),
		audit.WithDiff(original, org),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error("failed to log organization update", zap.Error(err))
	}

	return updatedOrg, nil
}
