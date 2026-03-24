package sequenceconfigservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.SequenceConfigRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.SequenceConfigRepository
	validator    *Validator
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.sequenceconfig"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetSequenceConfigRequest,
) (*tenant.SequenceConfigDocument, error) {
	return s.repo.GetByTenant(ctx, req)
}

func (s *Service) Update(
	ctx context.Context,
	doc *tenant.SequenceConfigDocument,
	userID pulid.ID,
) (*tenant.SequenceConfigDocument, error) {
	if multiErr := s.validator.ValidateUpdate(ctx, doc); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByTenant(ctx, repositories.GetSequenceConfigRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: doc.OrganizationID,
			BuID:  doc.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateByTenant(ctx, doc)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceSequenceConfig,
			ResourceID:     fmt.Sprintf("%s:%s", updated.OrganizationID, updated.BusinessUnitID),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updated),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updated.OrganizationID,
			BusinessUnitID: updated.BusinessUnitID,
		},
		auditservice.WithComment("Sequence configuration updated"),
		auditservice.WithDiff(original, updated),
		auditservice.WithCritical(),
	); err != nil {
		s.l.Error("failed to log sequence config update audit", zap.Error(err))
	}

	return updated, nil
}
