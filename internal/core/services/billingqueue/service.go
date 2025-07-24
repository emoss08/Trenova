/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package billingqueue

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.BillingQueueRepository
	PermService  services.PermissionService
	AuditService services.AuditService
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.BillingQueueRepository
	ps   services.PermissionService
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "billingqueue").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListBillingQueueRequest,
) (*ports.ListResult[*billingqueue.QueueItem], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.Filter.TenantOpts.UserID,
			Resource:       permission.ResourceBillingQueue,
			Action:         permission.ActionRead,
			BusinessUnitID: req.Filter.TenantOpts.BuID,
			OrganizationID: req.Filter.TenantOpts.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read billing queue items",
		)
	}

	entities, err := s.repo.List(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to list billing queue items")
		return nil, err
	}

	return entities, nil
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetBillingQueueItemRequest,
) (*billingqueue.QueueItem, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("billingQueueItemID", string(req.BillingQueueItemID)).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceBillingQueue,
			Action:         permission.ActionRead,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read billing queue items",
		)
	}

	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get billing queue item")
		return nil, err
	}

	return entity, nil
}

func (s *Service) BulkTransfer(ctx context.Context, req *repositories.BulkTransferRequest) error {
	log := s.l.With().Str("operation", "BulkTransfer").Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceBillingQueue,
			Action:         permission.ActionBulkTransfer,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to bulk transfer shipments to the billing queue",
		)
	}

	err = s.repo.BulkTransfer(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk transfer shipments to the billing queue")
		return err
	}

	return nil
}
