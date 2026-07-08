package dataretentionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultAuditRetentionDays = 120

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.DataRetentionRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.DataRetentionRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.data-retention"),
		repo: p.Repo,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DataRetention, error) {
	entity, err := s.repo.Get(ctx, repositories.GetDataRetentionRequest{
		UserID: tenantInfo.UserID,
		OrgID:  tenantInfo.OrgID,
		BuID:   tenantInfo.BuID,
	})
	if err == nil {
		return entity, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	return &tenant.DataRetention{
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		AuditRetentionPeriod: defaultAuditRetentionDays,
	}, nil
}

type UpdateDataRetentionRequest struct {
	TenantInfo                    pagination.TenantInfo `json:"-"`
	AuditRetentionPeriod          int                   `json:"auditRetentionPeriod"`
	EDIInboundFileRetentionPeriod int                   `json:"ediInboundFileRetentionPeriod"`
	EDIMessageRetentionPeriod     int                   `json:"ediMessageRetentionPeriod"`
}

func (s *Service) Update(
	ctx context.Context,
	req *UpdateDataRetentionRequest,
) (*tenant.DataRetention, error) {
	entity := &tenant.DataRetention{
		OrganizationID:                req.TenantInfo.OrgID,
		BusinessUnitID:                req.TenantInfo.BuID,
		AuditRetentionPeriod:          req.AuditRetentionPeriod,
		EDIInboundFileRetentionPeriod: req.EDIInboundFileRetentionPeriod,
		EDIMessageRetentionPeriod:     req.EDIMessageRetentionPeriod,
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return s.repo.Upsert(ctx, entity)
}
