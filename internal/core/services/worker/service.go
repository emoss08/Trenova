package worker

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/domain/worker"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/core/services/audit"
	"github.com/trenova-app/transport/internal/core/services/search"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/jsonutils"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/internal/pkg/validator/workervalidator"
	"github.com/trenova-app/transport/pkg/types"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger        *logger.Logger
	Repo          repositories.WorkerRepository
	PermService   services.PermissionService
	AuditService  services.AuditService
	SearchService *search.Service
	Validator     *workervalidator.Validator
}

type Service struct {
	repo repositories.WorkerRepository
	l    *zerolog.Logger
	ps   services.PermissionService
	as   services.AuditService
	ss   *search.Service
	v    *workervalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "worker").
		Logger()

	return &Service{
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		ss:   p.SearchService,
		l:    &log,
		v:    p.Validator,
	}
}

func (s *Service) SelectOptions(ctx context.Context, opts *repositories.ListWorkerOptions) ([]types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, eris.Wrap(err, "failed to list workers")
	}

	options := make([]types.SelectOption, len(result.Items))
	for i, worker := range result.Items {
		options[i] = types.SelectOption{
			Value: worker.ID.String(),
			Label: worker.FullName(),
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *repositories.ListWorkerOptions) (*ports.ListResult[*worker.Worker], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceWorker,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		})
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read workers")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list workers")
		return nil, eris.Wrap(err, "failed to list workers")
	}

	return &ports.ListResult[*worker.Worker]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetWorkerByIDOptions) (*worker.Worker, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("id", opts.WorkerID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceWorker,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
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

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get worker")
		return nil, eris.Wrap(err, "failed to get worker")
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, wrk *worker.Worker, userID pulid.ID) (*worker.Worker, error) {
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

	// Update the search index
	if err = s.ss.Index(ctx, createdWorker); err != nil {
		log.Error().Err(err).Msg("failed to update search index")
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

func (s *Service) Update(ctx context.Context, wrk *worker.Worker, userID pulid.ID) (*worker.Worker, error) {
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

	original, err := s.repo.GetByID(ctx, repositories.GetWorkerByIDOptions{
		OrgID:    wrk.OrganizationID,
		BuID:     wrk.BusinessUnitID,
		WorkerID: wrk.ID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get worker")
	}

	updatedWorker, err := s.repo.Update(ctx, wrk)
	if err != nil {
		log.Error().Err(err).Msg("failed to update worker")
		return nil, eris.Wrap(err, "update worker")
	}

	// Update the search index
	if err = s.ss.Index(ctx, updatedWorker); err != nil {
		log.Error().
			Err(err).
			Interface("worker", updatedWorker).
			Msg("failed to index worker")
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
