/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package worker

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/workervalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Validator    *workervalidator.Validator
	Repo         repositories.WorkerRepository
	PermService  services.PermissionService
	AuditService services.AuditService
}

type Service struct {
	l    *zerolog.Logger
	v    *workervalidator.Validator
	repo repositories.WorkerRepository
	ps   services.PermissionService
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "worker").
		Logger()

	return &Service{
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		l:    &log,
		v:    p.Validator,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.ListWorkerRequest,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, eris.Wrap(err, "failed to list workers")
	}

	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, worker := range result.Items {
		options = append(options, &types.SelectOption{
			Value: worker.ID.String(),
			Label: worker.FullName(),
		})
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListWorkerRequest,
) (*ports.ListResult[*worker.Worker], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceWorker,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read workers")
	}

	entities, err := s.repo.List(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to list workers")
		return nil, eris.Wrap(err, "failed to list workers")
	}

	return entities, nil
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("id", req.WorkerID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceWorker,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this worker")
	}

	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get worker")
		return nil, eris.Wrap(err, "failed to get worker")
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	wrk *worker.Worker,
	userID pulid.ID,
) (*worker.Worker, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("id", wrk.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceWorker,
				Action:         permission.ActionCreate,
				BusinessUnitID: wrk.BusinessUnitID,
				OrganizationID: wrk.OrganizationID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a worker")
	}

	// Validate the worker
	valCtx := &validator.ValidationContext{
		IsCreate: true,
	}
	if err := s.v.Validate(ctx, valCtx, wrk); err != nil {
		return nil, err
	}

	createdWorker, err := s.repo.Create(ctx, wrk)
	if err != nil {
		log.Error().Err(err).Msg("failed to create worker")
		return nil, eris.Wrap(err, "create worker")
	}

	// Log the create if the insert was successful
	if err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorker,
			ResourceID:     createdWorker.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdWorker),
			OrganizationID: createdWorker.OrganizationID,
			BusinessUnitID: createdWorker.BusinessUnitID,
		},
		audit.WithComment("Worker created"),
	); err != nil {
		log.Error().Err(err).Msg("failed to log worker creation")
	}

	return createdWorker, nil
}

func (s *Service) Update(
	ctx context.Context,
	wrk *worker.Worker,
	userID pulid.ID,
) (*worker.Worker, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("id", wrk.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceWorker,
				Action:         permission.ActionUpdate,
				BusinessUnitID: wrk.BusinessUnitID,
				OrganizationID: wrk.OrganizationID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this worker")
	}

	// Validate the worker
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, wrk); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetWorkerByIDRequest{
		WorkerID: wrk.ID,
		OrgID:    wrk.OrganizationID,
		BuID:     wrk.BusinessUnitID,
		FilterOptions: repositories.WorkerFilterOptions{
			IncludeProfile: true,
			IncludePTO:     true,
		},
	})
	if err != nil {
		return nil, eris.Wrap(err, "get worker")
	}

	updatedWorker, err := s.repo.Update(ctx, wrk)
	if err != nil {
		log.Error().Err(err).Msg("failed to update worker")
		return nil, eris.Wrap(err, "update worker")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorker,
			ResourceID:     updatedWorker.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedWorker),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedWorker.OrganizationID,
			BusinessUnitID: updatedWorker.BusinessUnitID,
		},
		audit.WithComment("Worker updated"),
		audit.WithDiff(original, updatedWorker),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log worker update")
	}

	return updatedWorker, nil
}

func (s *Service) ListUpcomingPTO(
	ctx context.Context,
	req *repositories.ListUpcomingWorkerPTORequest,
) (*ports.ListResult[*worker.WorkerPTO], error) {
	log := s.l.With().Str("operation", "ListUpcomingPTO").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceWorkerPTO,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read upcoming PTOs")
	}

	return s.repo.ListUpcomingPTO(ctx, req)
}

func (s *Service) ApprovePTO(
	ctx context.Context,
	req *repositories.ApprovePTORequest,
) error {
	log := s.l.With().
		Str("operation", "ApprovePTO").
		Interface("req", req).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.ApproverID,
				Resource:       permission.ResourceWorkerPTO,
				Action:         permission.ActionApprove,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to approve this PTO")
	}

	return s.repo.ApprovePTO(ctx, req)
}

func (s *Service) RejectPTO(
	ctx context.Context,
	req *repositories.RejectPTORequest,
) error {
	log := s.l.With().
		Str("operation", "RejectPTO").
		Interface("req", req).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.RejectorID,
				Resource:       permission.ResourceWorkerPTO,
				Action:         permission.ActionReject,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to reject this PTO")
	}

	return s.repo.RejectPTO(ctx, req)
}

func (s *Service) ListWorkerPTO(
	ctx context.Context,
	req *repositories.ListWorkerPTORequest,
) (*ports.ListResult[*worker.WorkerPTO], error) {
	log := s.l.With().
		Str("operation", "ListWorkerPTO").
		Interface("req", req).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceWorkerPTO,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read worker PTOs")
	}

	return s.repo.ListWorkerPTO(ctx, req)
}

func (s *Service) GetPTOChartData(
	ctx context.Context,
	req *repositories.PTOChartDataRequest,
) ([]*repositories.PTOChartDataPoint, error) {
	log := s.l.With().
		Str("operation", "GetPTOChartData").
		Interface("req", req).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceWorkerPTO,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read PTO chart data",
		)
	}

	return s.repo.GetPTOChartData(ctx, req)
}

func (s *Service) GetPTOCalendarData(
	ctx context.Context,
	req *repositories.PTOCalendarDataRequest,
) ([]*repositories.PTOCalendarEvent, error) {
	log := s.l.With().
		Str("operation", "GetPTOCalendarData").
		Interface("req", req).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceWorkerPTO,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read PTO calendar data",
		)
	}

	return s.repo.GetPTOCalendarData(ctx, req)
}
