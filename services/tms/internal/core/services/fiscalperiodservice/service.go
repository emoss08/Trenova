package fiscalperiodservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalcloseblockers"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           *postgres.Connection
	Repo         repositories.FiscalPeriodRepository
	Validator    *Validator
	AuditService services.AuditService
	Policy       *accountingcontrolpolicyservice.Service
}

type Service struct {
	l            *zap.Logger
	db           *postgres.Connection
	repo         repositories.FiscalPeriodRepository
	validator    *Validator
	auditService services.AuditService
	policy       *accountingcontrolpolicyservice.Service
}

const fiscalPeriodLockTimeout = 250 * time.Millisecond

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.fiscalperiod"),
		db:           p.DB,
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		policy:       p.Policy,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListFiscalPeriodsRequest,
) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetFiscalPeriodByIDRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetCloseBlockers(
	ctx context.Context,
	req repositories.GetFiscalPeriodByIDRequest,
) (*fiscalclose.Result, error) {
	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	periods, err := s.repo.ListByFiscalYearID(
		ctx,
		repositories.ListByFiscalYearIDRequest{
			FiscalYearID: entity.FiscalYearID,
			OrgID:        entity.OrganizationID,
			BuID:         entity.BusinessUnitID,
		},
	)
	if err != nil {
		return nil, err
	}

	blockers := make([]*fiscalclose.Blocker, 0)
	blockers = fiscalcloseblockers.AppendFromMultiError(
		blockers,
		s.validateCloseWithPeriods(entity, periods),
		"period",
	)
	blockers = fiscalcloseblockers.AppendFromError(
		blockers,
		s.validateCloseControl(ctx, entity),
		"accounting",
		"period",
	)
	if s.validator != nil {
		blockers = fiscalcloseblockers.AppendFromMultiError(
			blockers,
			s.validator.ValidateClose(ctx, entity),
			"period",
		)
	}

	return &fiscalclose.Result{CanClose: len(blockers) == 0, Blockers: blockers}, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create fiscal period", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalPeriod,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal period created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original fiscal period", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update fiscal period", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalPeriod,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal period updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteFiscalPeriodRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
		zap.String("userID", userID.String()),
	)

	existing, err := s.repo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get fiscal period for delete", zap.Error(err))
		return err
	}

	if multiErr := s.validateDelete(existing); multiErr != nil {
		return multiErr
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete fiscal period", zap.Error(err))
		return err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalPeriod,
		ResourceID:     existing.GetID().String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: existing.OrganizationID,
		BusinessUnitID: existing.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal period deleted"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}

func (s *Service) Close(
	ctx context.Context,
	req repositories.CloseFiscalPeriodRequest,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Close"),
		zap.String("id", req.ID.String()),
	)

	if s.db == nil {
		existing, err := s.repo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			log.Error("failed to get fiscal period", zap.Error(err))
			return nil, err
		}

		if multiErr := s.validateClose(ctx, existing); multiErr != nil {
			return nil, multiErr
		}
		if err = s.validateCloseControl(ctx, existing); err != nil {
			return nil, err
		}
		if multiErr := s.validator.ValidateClose(ctx, existing); multiErr != nil {
			return nil, multiErr
		}

		req.ClosedByID = userID
		req.ClosedAt = timeutils.NowUnix()
		closedEntity, err := s.repo.Close(ctx, req)
		if err != nil {
			log.Error("failed to close fiscal period", zap.Error(err))
			return nil, err
		}

		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     closedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(closedEntity),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: closedEntity.OrganizationID,
			BusinessUnitID: closedEntity.BusinessUnitID,
		},
			auditservice.WithComment("Fiscal period closed"),
			auditservice.WithDiff(existing, closedEntity),
		); err != nil {
			log.Error("failed to log audit action", zap.Error(err))
		}

		return closedEntity, nil
	}

	var existing *fiscalperiod.FiscalPeriod
	var closedEntity *fiscalperiod.FiscalPeriod
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalPeriodLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			var txErr error
			existing, txErr = s.repo.GetByIDForUpdate(
				txCtx,
				repositories.GetFiscalPeriodByIDRequest{
					ID:         req.ID,
					TenantInfo: req.TenantInfo,
				},
			)
			if txErr != nil {
				return txErr
			}

			periods, txErr := s.repo.ListByFiscalYearIDForUpdate(
				txCtx,
				repositories.ListByFiscalYearIDRequest{
					FiscalYearID: existing.FiscalYearID,
					OrgID:        existing.OrganizationID,
					BuID:         existing.BusinessUnitID,
				},
			)
			if txErr != nil {
				return txErr
			}

			if multiErr := s.validateCloseWithPeriods(existing, periods); multiErr != nil {
				return multiErr
			}
			if txErr = s.validateCloseControl(txCtx, existing); txErr != nil {
				return txErr
			}
			if multiErr := s.validator.ValidateClose(txCtx, existing); multiErr != nil {
				return multiErr
			}

			req.ClosedByID = userID
			req.ClosedAt = timeutils.NowUnix()
			closedEntity, txErr = s.repo.Close(txCtx, req)
			return txErr
		},
	)
	if err != nil {
		log.Error("failed to close fiscal period", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal period is busy. Retry the request.",
		)
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalPeriod,
		ResourceID:     closedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(closedEntity),
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: closedEntity.OrganizationID,
		BusinessUnitID: closedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal period closed"),
		auditservice.WithDiff(existing, closedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return closedEntity, nil
}

func (s *Service) Reopen(
	ctx context.Context,
	req repositories.ReopenFiscalPeriodRequest,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Reopen"),
		zap.String("id", req.ID.String()),
	)

	if s.db == nil {
		existing, err := s.repo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			log.Error("failed to get fiscal period", zap.Error(err))
			return nil, err
		}

		if multiErr := s.validateReopen(ctx, existing); multiErr != nil {
			return nil, multiErr
		}

		reopenedEntity, err := s.repo.Reopen(ctx, req)
		if err != nil {
			log.Error("failed to reopen fiscal period", zap.Error(err))
			return nil, err
		}

		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     reopenedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(reopenedEntity),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: reopenedEntity.OrganizationID,
			BusinessUnitID: reopenedEntity.BusinessUnitID,
		},
			auditservice.WithComment("Fiscal period reopened"),
			auditservice.WithDiff(existing, reopenedEntity),
		); err != nil {
			log.Error("failed to log audit action", zap.Error(err))
		}

		return reopenedEntity, nil
	}

	var existing *fiscalperiod.FiscalPeriod
	var reopenedEntity *fiscalperiod.FiscalPeriod
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalPeriodLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			var txErr error
			existing, txErr = s.repo.GetByIDForUpdate(
				txCtx,
				repositories.GetFiscalPeriodByIDRequest{
					ID:         req.ID,
					TenantInfo: req.TenantInfo,
				},
			)
			if txErr != nil {
				return txErr
			}

			periods, txErr := s.repo.ListByFiscalYearIDForUpdate(
				txCtx,
				repositories.ListByFiscalYearIDRequest{
					FiscalYearID: existing.FiscalYearID,
					OrgID:        existing.OrganizationID,
					BuID:         existing.BusinessUnitID,
				},
			)
			if txErr != nil {
				return txErr
			}

			if multiErr := s.validateReopenWithPeriods(existing, periods); multiErr != nil {
				return multiErr
			}

			reopenedEntity, txErr = s.repo.Reopen(txCtx, req)
			return txErr
		},
	)
	if err != nil {
		log.Error("failed to reopen fiscal period", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal period is busy. Retry the request.",
		)
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalPeriod,
		ResourceID:     reopenedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(reopenedEntity),
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: reopenedEntity.OrganizationID,
		BusinessUnitID: reopenedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal period reopened"),
		auditservice.WithDiff(existing, reopenedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return reopenedEntity, nil
}

func (s *Service) Lock(
	ctx context.Context,
	req repositories.LockFiscalPeriodRequest,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Lock"),
		zap.String("id", req.ID.String()),
	)

	if s.db == nil {
		existing, err := s.repo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			log.Error("failed to get fiscal period", zap.Error(err))
			return nil, err
		}

		if multiErr := s.validateLock(existing); multiErr != nil {
			return nil, multiErr
		}

		lockedEntity, err := s.repo.Lock(ctx, req)
		if err != nil {
			log.Error("failed to lock fiscal period", zap.Error(err))
			return nil, err
		}

		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     lockedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(lockedEntity),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: lockedEntity.OrganizationID,
			BusinessUnitID: lockedEntity.BusinessUnitID,
		},
			auditservice.WithComment("Fiscal period locked"),
			auditservice.WithDiff(existing, lockedEntity),
		); err != nil {
			log.Error("failed to log audit action", zap.Error(err))
		}

		return lockedEntity, nil
	}

	var existing *fiscalperiod.FiscalPeriod
	var lockedEntity *fiscalperiod.FiscalPeriod
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalPeriodLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			var txErr error
			existing, txErr = s.repo.GetByIDForUpdate(
				txCtx,
				repositories.GetFiscalPeriodByIDRequest{
					ID:         req.ID,
					TenantInfo: req.TenantInfo,
				},
			)
			if txErr != nil {
				return txErr
			}

			if multiErr := s.validateLock(existing); multiErr != nil {
				return multiErr
			}

			lockedEntity, txErr = s.repo.Lock(txCtx, req)
			return txErr
		},
	)
	if err != nil {
		log.Error("failed to lock fiscal period", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal period is busy. Retry the request.",
		)
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalPeriod,
		ResourceID:     lockedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(lockedEntity),
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: lockedEntity.OrganizationID,
		BusinessUnitID: lockedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal period locked"),
		auditservice.WithDiff(existing, lockedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return lockedEntity, nil
}

func (s *Service) Unlock(
	ctx context.Context,
	req repositories.UnlockFiscalPeriodRequest,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Unlock"),
		zap.String("id", req.ID.String()),
	)

	if s.db == nil {
		existing, err := s.repo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			log.Error("failed to get fiscal period", zap.Error(err))
			return nil, err
		}

		if multiErr := s.validateUnlock(existing); multiErr != nil {
			return nil, multiErr
		}

		unlockedEntity, err := s.repo.Unlock(ctx, req)
		if err != nil {
			log.Error("failed to unlock fiscal period", zap.Error(err))
			return nil, err
		}

		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     unlockedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(unlockedEntity),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: unlockedEntity.OrganizationID,
			BusinessUnitID: unlockedEntity.BusinessUnitID,
		},
			auditservice.WithComment("Fiscal period unlocked"),
			auditservice.WithDiff(existing, unlockedEntity),
		); err != nil {
			log.Error("failed to log audit action", zap.Error(err))
		}

		return unlockedEntity, nil
	}

	var existing *fiscalperiod.FiscalPeriod
	var unlockedEntity *fiscalperiod.FiscalPeriod
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalPeriodLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			var txErr error
			existing, txErr = s.repo.GetByIDForUpdate(
				txCtx,
				repositories.GetFiscalPeriodByIDRequest{
					ID:         req.ID,
					TenantInfo: req.TenantInfo,
				},
			)
			if txErr != nil {
				return txErr
			}

			if multiErr := s.validateUnlock(existing); multiErr != nil {
				return multiErr
			}

			unlockedEntity, txErr = s.repo.Unlock(txCtx, req)
			return txErr
		},
	)
	if err != nil {
		log.Error("failed to unlock fiscal period", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal period is busy. Retry the request.",
		)
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalPeriod,
		ResourceID:     unlockedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(unlockedEntity),
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: unlockedEntity.OrganizationID,
		BusinessUnitID: unlockedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal period unlocked"),
		auditservice.WithDiff(existing, unlockedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return unlockedEntity, nil
}

func (s *Service) validateDelete(entity *fiscalperiod.FiscalPeriod) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status != fiscalperiod.StatusOpen {
		multiErr.Add("status", errortypes.ErrInvalid, "Only Open fiscal periods can be deleted")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) validateClose(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	periods, err := s.repo.ListByFiscalYearID(ctx, repositories.ListByFiscalYearIDRequest{
		FiscalYearID: entity.FiscalYearID,
		OrgID:        entity.OrganizationID,
		BuID:         entity.BusinessUnitID,
	})
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"fiscalYearId",
			errortypes.ErrSystemError,
			fmt.Sprintf("Failed to validate sequential close: %v", err),
		)
		return multiErr
	}

	return s.validateCloseWithPeriods(entity, periods)
}

func (s *Service) validateCloseWithPeriods(
	entity *fiscalperiod.FiscalPeriod,
	periods []*fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status != fiscalperiod.StatusOpen {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Only Open fiscal periods can be closed. Current status: %s",
				entity.Status,
			),
		)
		return multiErr
	}

	if entity.PeriodNumber > 1 {
		for _, p := range periods {
			if p.PeriodNumber < entity.PeriodNumber && p.Status == fiscalperiod.StatusOpen {
				multiErr.Add(
					"status",
					errortypes.ErrInvalid,
					fmt.Sprintf(
						"Cannot close period %d: period %d is still open. Close periods sequentially.",
						entity.PeriodNumber,
						p.PeriodNumber,
					),
				)
				break
			}
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) validateCloseControl(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) error {
	if s.validator == nil || s.validator.accountingRepo == nil {
		return nil
	}

	control, err := s.validator.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}

	return s.accountingPolicyService().ValidateManualPeriodClose(control)
}

func (s *Service) accountingPolicyService() *accountingcontrolpolicyservice.Service {
	if s.policy != nil {
		return s.policy
	}
	return accountingcontrolpolicyservice.New(
		accountingcontrolpolicyservice.Params{Logger: zap.NewNop()},
	)
}

func (s *Service) validateReopen(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	periods, err := s.repo.ListByFiscalYearID(ctx, repositories.ListByFiscalYearIDRequest{
		FiscalYearID: entity.FiscalYearID,
		OrgID:        entity.OrganizationID,
		BuID:         entity.BusinessUnitID,
	})
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"fiscalYearId",
			errortypes.ErrSystemError,
			fmt.Sprintf("Failed to validate reopen order: %v", err),
		)
		return multiErr
	}

	return s.validateReopenWithPeriods(entity, periods)
}

func (s *Service) validateReopenWithPeriods(
	entity *fiscalperiod.FiscalPeriod,
	periods []*fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status != fiscalperiod.StatusClosed {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Only Closed fiscal periods can be reopened. Current status: %s",
				entity.Status,
			),
		)
		return multiErr
	}

	for _, p := range periods {
		if p.PeriodNumber > entity.PeriodNumber &&
			(p.Status == fiscalperiod.StatusClosed || p.Status == fiscalperiod.StatusLocked) {
			multiErr.Add(
				"status",
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"Cannot reopen period %d: period %d is already closed. Reopen periods in reverse order.",
					entity.PeriodNumber,
					p.PeriodNumber,
				),
			)
			break
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) validateLock(entity *fiscalperiod.FiscalPeriod) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status != fiscalperiod.StatusOpen {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Only Open fiscal periods can be locked. Current status: %s",
				entity.Status,
			),
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) validateUnlock(entity *fiscalperiod.FiscalPeriod) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status != fiscalperiod.StatusLocked {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Only Locked fiscal periods can be unlocked. Current status: %s",
				entity.Status,
			),
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
