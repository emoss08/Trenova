//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package fiscalyearservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalcloseblockers"
	"github.com/emoss08/trenova/internal/core/services/fiscalcloseservice"
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

const fiscalYearLockTimeout = 250 * time.Millisecond

type Params struct {
	fx.In

	Logger           *zap.Logger
	DB               *postgres.Connection
	Repo             repositories.FiscalYearRepository
	FiscalPeriodRepo repositories.FiscalPeriodRepository
	Validator        *Validator
	AuditService     services.AuditService
	Transformer      services.DataTransformer
	CloseService     *fiscalcloseservice.Service
}

type Service struct {
	l                *zap.Logger
	db               *postgres.Connection
	repo             repositories.FiscalYearRepository
	fiscalPeriodRepo repositories.FiscalPeriodRepository
	validator        *Validator
	auditService     services.AuditService
	transformer      services.DataTransformer
	closeService     *fiscalcloseservice.Service
}

func New(p Params) *Service {
	return &Service{
		l:                p.Logger.Named("service.fiscalyear"),
		db:               p.DB,
		repo:             p.Repo,
		fiscalPeriodRepo: p.FiscalPeriodRepo,
		validator:        p.Validator,
		auditService:     p.AuditService,
		transformer:      p.Transformer,
		closeService:     p.CloseService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListFiscalYearsRequest,
) (*pagination.ListResult[*fiscalyear.FiscalYear], error) {
	result, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(result.Items) > 0 {
		return result, nil
	}

	if req == nil || req.Filter == nil {
		return result, nil
	}

	if _, err = s.ensureBootstrapCurrentFiscalYear(
		ctx,
		req.Filter.TenantInfo.OrgID,
		req.Filter.TenantInfo.BuID,
	); err != nil {
		if errortypes.IsConflictError(err) {
			return result, nil
		}

		return nil, err
	}

	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListFiscalYearConnectionRequest,
) (*pagination.CursorListResult[*fiscalyear.FiscalYear], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetFiscalYearByIDRequest,
) (*fiscalyear.FiscalYear, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetCloseBlockers(
	ctx context.Context,
	req repositories.GetFiscalYearByIDRequest,
) (*fiscalclose.Result, error) {
	plan, err := s.GetClosePreview(ctx, req)
	if err != nil {
		return nil, err
	}

	return &fiscalclose.Result{CanClose: plan.CanClose, Blockers: plan.Blockers}, nil
}

// GetClosePreview reports what closing the year would post — the closing entry
// that empties the income statement into retained earnings and the opening entry
// that carries the balance sheet into the next year — together with everything
// that currently stands in the way. Nothing is written.
func (s *Service) GetClosePreview(
	ctx context.Context,
	req repositories.GetFiscalYearByIDRequest,
) (*fiscalclose.Plan, error) {
	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	plan, err := s.closeService.BuildPlan(ctx, entity)
	if err != nil {
		return nil, err
	}

	yearBlockers := fiscalcloseblockers.AppendFromMultiError(
		make([]*fiscalclose.Blocker, 0, len(plan.Blockers)),
		s.validateClose(ctx, entity),
		"year",
	)
	plan.Blockers = append(yearBlockers, plan.Blockers...)
	plan.CanClose = len(plan.Blockers) == 0

	return plan, nil
}

func (s *Service) GetCurrentFiscalYear(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	return s.ensureBootstrapCurrentFiscalYear(ctx, orgID, buID)
}

func (s *Service) ensureBootstrapCurrentFiscalYear(
	ctx context.Context,
	orgID pulid.ID,
	buID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	existing, err := s.repo.GetCurrentFiscalYear(ctx, repositories.GetCurrentFiscalYearRequest{
		OrgID: orgID,
		BuID:  buID,
	})
	if err == nil {
		return existing, nil
	}

	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	count, err := s.repo.CountByTenant(ctx, repositories.CountFiscalYearsByTenantRequest{
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		return nil, err
	}

	// If data already exists but nothing is current, do not guess.
	if count > 0 {
		return nil, errortypes.NewConflictError(
			"No current fiscal year is set. Activate an existing fiscal year before continuing.",
		)
	}

	createdFY, err := s.bootstrapCurrentCalendarFiscalYear(ctx, orgID, buID)
	if err != nil {
		return nil, err
	}

	return createdFY, nil
}

//nolint:funlen // existing workflow or route registration is intentionally kept together
func (s *Service) bootstrapCurrentCalendarFiscalYear(
	ctx context.Context,
	orgID pulid.ID,
	buID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	now := time.Now().UTC()
	year := now.Year()

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC)

	entity := &fiscalyear.FiscalYear{
		OrganizationID:        orgID,
		BusinessUnitID:        buID,
		Status:                fiscalyear.StatusOpen,
		Year:                  year,
		Name:                  fmt.Sprintf("FY %d", year),
		StartDate:             start.Unix(),
		EndDate:               end.Unix(),
		IsCurrent:             true,
		IsCalendarYear:        true,
		AllowAdjustingEntries: false,
	}

	if s.db == nil {
		createdEntity, err := s.repo.Create(ctx, entity)
		if err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return s.repo.GetCurrentFiscalYear(ctx, repositories.GetCurrentFiscalYearRequest{
					OrgID: orgID,
					BuID:  buID,
				})
			}

			return nil, err
		}

		if genErr := s.generatePeriods(ctx, createdEntity, s.l); genErr != nil {
			return nil, genErr
		}

		return createdEntity, nil
	}

	var createdEntity *fiscalyear.FiscalYear
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalYearLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			current, txErr := s.repo.GetCurrentFiscalYearForUpdate(
				txCtx,
				repositories.GetCurrentFiscalYearRequest{
					OrgID: orgID,
					BuID:  buID,
				},
			)
			if txErr == nil {
				createdEntity = current
				return nil
			}
			if !errortypes.IsNotFoundError(txErr) {
				return txErr
			}

			count, txErr := s.repo.CountByTenant(
				txCtx,
				repositories.CountFiscalYearsByTenantRequest{
					OrgID: orgID,
					BuID:  buID,
				},
			)
			if txErr != nil {
				return txErr
			}
			if count > 0 {
				return errortypes.NewConflictError(
					"No current fiscal year is set. Activate an existing fiscal year before continuing.",
				)
			}

			createdEntity, txErr = s.repo.Create(txCtx, entity)
			if txErr != nil {
				return txErr
			}

			return s.generatePeriods(txCtx, createdEntity, s.l)
		},
	)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return s.repo.GetCurrentFiscalYear(ctx, repositories.GetCurrentFiscalYearRequest{
				OrgID: orgID,
				BuID:  buID,
			})
		}

		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal year is busy. Retry the request.",
		)
	}

	return createdEntity, nil
}

func (s *Service) ListPeriods(
	ctx context.Context,
	fiscalYearID, orgID, buID pulid.ID,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return s.fiscalPeriodRepo.ListByFiscalYearID(ctx, repositories.ListByFiscalYearIDRequest{
		FiscalYearID: fiscalYearID,
		OrgID:        orgID,
		BuID:         buID,
	})
}

func (s *Service) ListPeriodsByFiscalYearIDs(
	ctx context.Context,
	req repositories.ListByFiscalYearIDsRequest,
) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
	return s.fiscalPeriodRepo.ListByFiscalYearIDs(ctx, req)
}

func (s *Service) GetPeriodForDate(
	ctx context.Context,
	orgID, buID pulid.ID,
	date int64,
) (*fiscalperiod.FiscalPeriod, error) {
	return s.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: orgID,
		BuID:  buID,
		Date:  date,
	})
}

func (s *Service) Create(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if err := s.transformer.TransformFiscalYear(ctx, entity); err != nil {
		log.Error("failed to transform fiscal year", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.createWithPeriods(ctx, entity, log)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalYear,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal year created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) createWithPeriods(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
	log *zap.Logger,
) (*fiscalyear.FiscalYear, error) {
	if s.db == nil {
		createdEntity, err := s.repo.Create(ctx, entity)
		if err != nil {
			log.Error("failed to create fiscal year", zap.Error(err))
			return nil, err
		}

		if genErr := s.generatePeriods(ctx, createdEntity, log); genErr != nil {
			log.Error("failed to generate fiscal periods", zap.Error(genErr))
			return nil, genErr
		}

		return createdEntity, nil
	}

	var createdEntity *fiscalyear.FiscalYear
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalYearLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			var txErr error
			createdEntity, txErr = s.repo.Create(txCtx, entity)
			if txErr != nil {
				return txErr
			}

			return s.generatePeriods(txCtx, createdEntity, log)
		},
	)
	if err != nil {
		log.Error("failed to create fiscal year with periods", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal year is busy. Retry the request.",
		)
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if err := s.transformer.TransformFiscalYear(ctx, entity); err != nil {
		log.Error("failed to transform fiscal year", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetFiscalYearByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original fiscal year", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update fiscal year", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalYear,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal year updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteFiscalYearRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
		zap.String("userID", userID.String()),
	)

	existing, err := s.repo.GetByID(ctx, repositories.GetFiscalYearByIDRequest(req))
	if err != nil {
		log.Error("failed to get fiscal year for delete", zap.Error(err))
		return err
	}

	if multiErr := s.validateDelete(existing); multiErr != nil {
		return multiErr
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete fiscal year", zap.Error(err))
		return err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalYear,
		ResourceID:     existing.GetID().String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: existing.OrganizationID,
		BusinessUnitID: existing.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal year deleted"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}

func (s *Service) Close(
	ctx context.Context,
	req repositories.CloseFiscalYearRequest,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Close"),
		zap.String("id", req.ID.String()),
	)

	existing, err := s.repo.GetByID(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get fiscal year", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validateClose(ctx, existing); multiErr != nil {
		return nil, multiErr
	}

	req.ClosedByID = userID
	req.ClosedAt = timeutils.NowUnix()

	closedEntity, postResult, err := s.closeWithAccounting(ctx, req, userID)
	if err != nil {
		log.Error("failed to close fiscal year", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalYear,
		ResourceID:     closedEntity.GetID().String(),
		Operation:      permission.OpClose,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(closedEntity),
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: closedEntity.OrganizationID,
		BusinessUnitID: closedEntity.BusinessUnitID,
	},
		auditservice.WithComment(closeAuditComment(postResult)),
		auditservice.WithDiff(existing, closedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return closedEntity, nil
}

// closeWithAccounting posts the year-end entries and flips the year to Closed in
// one transaction. The two have to move together: a year marked closed without
// its closing entries would leave revenue on the books forever, and closing
// entries without the status flip would be posted again on the next attempt.
func (s *Service) closeWithAccounting(
	ctx context.Context,
	req repositories.CloseFiscalYearRequest,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, *fiscalclose.PostResult, error) {
	if s.db == nil {
		return s.closeInTx(ctx, req, userID)
	}

	var (
		closedEntity *fiscalyear.FiscalYear
		postResult   *fiscalclose.PostResult
	)
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalYearLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			var txErr error
			closedEntity, postResult, txErr = s.closeInTx(txCtx, req, userID)

			return txErr
		},
	)
	if err != nil {
		return nil, nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal year is busy. Retry the request.",
		)
	}

	return closedEntity, postResult, nil
}

func (s *Service) closeInTx(
	ctx context.Context,
	req repositories.CloseFiscalYearRequest,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, *fiscalclose.PostResult, error) {
	current, err := s.repo.GetByIDForUpdate(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	if multiErr := s.validateClose(ctx, current); multiErr != nil {
		return nil, nil, multiErr
	}

	postResult, err := s.closeService.Post(ctx, current, userID)
	if err != nil {
		return nil, nil, err
	}

	closedEntity, err := s.repo.Close(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return closedEntity, postResult, nil
}

// Reopen unwinds a closed year so corrections can be posted. The closing and
// opening entries are reversed rather than deleted, so the ledger keeps a record
// of both the close and its undo.
func (s *Service) Reopen(
	ctx context.Context,
	req repositories.ReopenFiscalYearRequest,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Reopen"),
		zap.String("id", req.ID.String()),
	)

	existing, err := s.repo.GetByID(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get fiscal year", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validateReopen(existing, req.ReopenReason); multiErr != nil {
		return nil, multiErr
	}

	req.ReopenedByID = userID
	req.ReopenedAt = timeutils.NowUnix()

	reopenedEntity, err := s.reopenWithAccounting(ctx, req, userID)
	if err != nil {
		log.Error("failed to reopen fiscal year", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalYear,
		ResourceID:     reopenedEntity.GetID().String(),
		Operation:      permission.OpReopen,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(reopenedEntity),
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: reopenedEntity.OrganizationID,
		BusinessUnitID: reopenedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal year reopened: "+req.ReopenReason),
		auditservice.WithDiff(existing, reopenedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return reopenedEntity, nil
}

func (s *Service) reopenWithAccounting(
	ctx context.Context,
	req repositories.ReopenFiscalYearRequest,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	if s.db == nil {
		return s.reopenInTx(ctx, req, userID)
	}

	var reopenedEntity *fiscalyear.FiscalYear
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalYearLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			var txErr error
			reopenedEntity, txErr = s.reopenInTx(txCtx, req, userID)

			return txErr
		},
	)
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal year is busy. Retry the request.",
		)
	}

	return reopenedEntity, nil
}

func (s *Service) reopenInTx(
	ctx context.Context,
	req repositories.ReopenFiscalYearRequest,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	current, err := s.repo.GetByIDForUpdate(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if multiErr := s.validateReopen(current, req.ReopenReason); multiErr != nil {
		return nil, multiErr
	}

	if _, err = s.closeService.Reverse(ctx, current, userID, req.ReopenReason); err != nil {
		return nil, err
	}

	return s.repo.Reopen(ctx, req)
}

func closeAuditComment(result *fiscalclose.PostResult) string {
	if result == nil {
		return "Fiscal year closed"
	}

	return fmt.Sprintf(
		"Fiscal year closed; net income %d minor units posted across %d journal entries",
		result.NetIncomeMinor,
		len(result.Entries),
	)
}

//nolint:funlen // existing workflow or route registration is intentionally kept together
func (s *Service) Activate(
	ctx context.Context,
	req repositories.ActivateFiscalYearRequest,
	userID pulid.ID,
) (*fiscalyear.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Activate"),
		zap.String("id", req.ID.String()),
	)

	if s.db == nil {
		existing, err := s.repo.GetByID(ctx, repositories.GetFiscalYearByIDRequest(req))
		if err != nil {
			log.Error("failed to get fiscal year", zap.Error(err))
			return nil, err
		}

		if multiErr := s.validateActivate(existing); multiErr != nil {
			return nil, multiErr
		}

		activatedEntity, err := s.repo.Activate(ctx, req)
		if err != nil {
			log.Error("failed to activate fiscal year", zap.Error(err))
			return nil, err
		}

		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceFiscalYear,
			ResourceID:     activatedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(activatedEntity),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: activatedEntity.OrganizationID,
			BusinessUnitID: activatedEntity.BusinessUnitID,
		},
			auditservice.WithComment("Fiscal year activated"),
			auditservice.WithDiff(existing, activatedEntity),
		); err != nil {
			log.Error("failed to log audit action", zap.Error(err))
		}

		return activatedEntity, nil
	}

	var existing *fiscalyear.FiscalYear
	var activatedEntity *fiscalyear.FiscalYear
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: fiscalYearLockTimeout},
		func(txCtx context.Context, _ bun.Tx) error {
			var txErr error
			existing, txErr = s.repo.GetByIDForUpdate(
				txCtx,
				repositories.GetFiscalYearByIDRequest(req),
			)
			if txErr != nil {
				return txErr
			}

			if _, txErr = s.repo.GetCurrentFiscalYearForUpdate(
				txCtx,
				repositories.GetCurrentFiscalYearRequest{
					OrgID: req.TenantInfo.OrgID,
					BuID:  req.TenantInfo.BuID,
				},
			); txErr != nil &&
				!errortypes.IsNotFoundError(txErr) {
				return txErr
			}

			if multiErr := s.validateActivate(existing); multiErr != nil {
				return multiErr
			}

			activatedEntity, txErr = s.repo.Activate(txCtx, req)
			return txErr
		},
	)
	if err != nil {
		log.Error("failed to activate fiscal year", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal year is busy. Retry the request.",
		)
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalYear,
		ResourceID:     activatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(activatedEntity),
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: activatedEntity.OrganizationID,
		BusinessUnitID: activatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fiscal year activated"),
		auditservice.WithDiff(existing, activatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return activatedEntity, nil
}

func (s *Service) validateDelete(entity *fiscalyear.FiscalYear) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status != fiscalyear.StatusDraft {
		multiErr.Add("status", errortypes.ErrInvalid, "Only Draft fiscal years can be deleted")
	}

	if entity.IsCurrent {
		multiErr.Add("isCurrent", errortypes.ErrInvalid, "Cannot delete the current fiscal year")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) validateClose(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status != fiscalyear.StatusOpen {
		multiErr.Add("status", errortypes.ErrInvalid,
			"Only Open fiscal years can be closed. Current status: {0}", entity.Status)
		return multiErr
	}

	openCount, err := s.fiscalPeriodRepo.GetOpenPeriodsCountByFiscalYear(
		ctx,
		repositories.GetOpenPeriodsCountByFiscalYearRequest{
			FiscalYearID: entity.ID,
			OrgID:        entity.OrganizationID,
			BuID:         entity.BusinessUnitID,
		},
	)
	if err != nil {
		multiErr.Add("__all__", errortypes.ErrSystemError, "Failed to check open periods")
		return multiErr
	}

	if openCount > 0 {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Cannot close fiscal year: {0} period(s) are still open. Close all periods first.", openCount,
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// validateReopen guards the only path that un-closes a year. A reason is
// mandatory because the reversal it triggers is what an auditor will ask about.
func (s *Service) validateReopen(
	entity *fiscalyear.FiscalYear,
	reason string,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status != fiscalyear.StatusClosed {
		multiErr.Add("status", errortypes.ErrInvalid,
			"Only Closed fiscal years can be reopened. Current status: {0}", entity.Status)
	}

	if strings.TrimSpace(reason) == "" {
		multiErr.Add("reopenReason", errortypes.ErrRequired, "A reason for reopening is required")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) validateActivate(entity *fiscalyear.FiscalYear) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.Status == fiscalyear.StatusPermanentlyClosed {
		multiErr.Add("status", errortypes.ErrInvalid, "Closed fiscal years cannot be activated")
	}

	// A closed year carries posted closing entries. Flipping it back to Open
	// here would leave those entries standing against an open year, so the only
	// way back is Reopen, which reverses them first.
	if entity.Status == fiscalyear.StatusClosed {
		multiErr.Add("status", errortypes.ErrInvalid,
			"This fiscal year is closed. Reopen it first so its closing entries are reversed.")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) generatePeriods(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	log *zap.Logger,
) error {
	periods := fy.GenerateMonthlyPeriods()

	err := s.fiscalPeriodRepo.BulkCreate(ctx, &repositories.BulkCreateFiscalPeriodsRequest{
		Periods: periods,
		TenantInfo: pagination.TenantInfo{
			OrgID: fy.OrganizationID,
			BuID:  fy.BusinessUnitID,
		},
	})
	if err != nil {
		log.Error("failed to generate fiscal periods", zap.Error(err))
		return err
	}

	log.Info("successfully generated fiscal periods",
		zap.Int("count", len(periods)),
		zap.String("fiscalYearId", fy.ID.String()),
	)

	return nil
}
