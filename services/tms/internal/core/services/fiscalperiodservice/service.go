//nolint:funlen // existing legacy workflow/API shape is intentionally kept stable
package fiscalperiodservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
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

	Logger         *zap.Logger
	DB             *postgres.Connection
	Repo           repositories.FiscalPeriodRepository
	FiscalYearRepo repositories.FiscalYearRepository
	Validator      *Validator
	AuditService   services.AuditService
	Policy         *accountingcontrolpolicyservice.Service
}

type Service struct {
	l              *zap.Logger
	db             *postgres.Connection
	repo           repositories.FiscalPeriodRepository
	fiscalYearRepo repositories.FiscalYearRepository
	validator      *Validator
	auditService   services.AuditService
	policy         *accountingcontrolpolicyservice.Service
}

const fiscalPeriodLockTimeout = 250 * time.Millisecond

func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.fiscalperiod"),
		db:             p.DB,
		repo:           p.Repo,
		fiscalYearRepo: p.FiscalYearRepo,
		validator:      p.Validator,
		auditService:   p.AuditService,
		policy:         p.Policy,
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

	if multiErr := validateEditable(original, entity); multiErr != nil {
		return nil, multiErr
	}

	preserveLifecycle(original, entity)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
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

	existing, err := s.repo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest(req))
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

type transitionState struct {
	period  *fiscalperiod.FiscalPeriod
	periods []*fiscalperiod.FiscalPeriod
	year    *fiscalyear.FiscalYear
}

type transition struct {
	name      string
	operation permission.Operation
	comment   string
	validate  func(ctx context.Context, state transitionState) error
	apply     func(ctx context.Context, state transitionState) (*fiscalperiod.FiscalPeriod, error)
}

func (s *Service) Close(
	ctx context.Context,
	req repositories.CloseFiscalPeriodRequest, //nolint:gocritic // stable API shape
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	return s.runTransition(ctx, req.ID, req.TenantInfo, userID, transition{
		name:      "Close",
		operation: permission.OpClose,
		comment:   "Fiscal period closed",
		validate: func(ctx context.Context, state transitionState) error {
			if multiErr := s.validateCloseWithPeriods(state.period, state.periods); multiErr != nil {
				return multiErr
			}
			if err := s.validateCloseControl(ctx, state.period); err != nil {
				return err
			}
			if s.validator != nil {
				if multiErr := s.validator.ValidateClose(ctx, state.period); multiErr != nil {
					return multiErr
				}
			}

			return nil
		},
		apply: func(ctx context.Context, _ transitionState) (*fiscalperiod.FiscalPeriod, error) {
			req.ClosedByID = userID
			req.ClosedAt = timeutils.NowUnix()
			return s.repo.Close(ctx, req)
		},
	})
}

func (s *Service) Reopen(
	ctx context.Context,
	req repositories.ReopenFiscalPeriodRequest, //nolint:gocritic // stable API shape
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	return s.runTransition(ctx, req.ID, req.TenantInfo, userID, transition{
		name:      "Reopen",
		operation: permission.OpReopen,
		comment:   "Fiscal period reopened",
		validate: func(_ context.Context, state transitionState) error {
			if multiErr := validateReopen(state, req.ReopenReason); multiErr != nil {
				return multiErr
			}

			return nil
		},
		apply: func(ctx context.Context, _ transitionState) (*fiscalperiod.FiscalPeriod, error) {
			req.ReopenReason = strings.TrimSpace(req.ReopenReason)
			req.ReopenedByID = userID
			req.ReopenedAt = timeutils.NowUnix()
			return s.repo.Reopen(ctx, req)
		},
	})
}

func (s *Service) Lock(
	ctx context.Context,
	req repositories.LockFiscalPeriodRequest,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	return s.runTransition(ctx, req.ID, req.TenantInfo, userID, transition{
		name:      "Lock",
		operation: permission.OpLock,
		comment:   "Fiscal period locked",
		validate: func(_ context.Context, state transitionState) error {
			if multiErr := validateLock(state.period); multiErr != nil {
				return multiErr
			}

			return nil
		},
		apply: func(ctx context.Context, _ transitionState) (*fiscalperiod.FiscalPeriod, error) {
			req.LockedByID = userID
			req.LockedAt = timeutils.NowUnix()
			return s.repo.Lock(ctx, req)
		},
	})
}

func (s *Service) Unlock(
	ctx context.Context,
	req repositories.UnlockFiscalPeriodRequest,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	return s.runTransition(ctx, req.ID, req.TenantInfo, userID, transition{
		name:      "Unlock",
		operation: permission.OpUnlock,
		comment:   "Fiscal period unlocked",
		validate: func(_ context.Context, state transitionState) error {
			if multiErr := validateUnlock(state); multiErr != nil {
				return multiErr
			}

			return nil
		},
		apply: func(ctx context.Context, _ transitionState) (*fiscalperiod.FiscalPeriod, error) {
			return s.repo.Unlock(ctx, req)
		},
	})
}

func (s *Service) Activate(
	ctx context.Context,
	req repositories.ActivateFiscalPeriodRequest,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	return s.runTransition(ctx, req.ID, req.TenantInfo, userID, transition{
		name:      "Activate",
		operation: permission.OpActivate,
		comment:   "Fiscal period opened",
		validate: func(_ context.Context, state transitionState) error {
			if multiErr := validateActivate(state); multiErr != nil {
				return multiErr
			}

			return nil
		},
		apply: func(ctx context.Context, _ transitionState) (*fiscalperiod.FiscalPeriod, error) {
			return s.repo.Activate(ctx, req)
		},
	})
}

func (s *Service) runTransition(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	t transition,
) (*fiscalperiod.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", t.name),
		zap.String("id", id.String()),
	)

	var (
		state   transitionState
		updated *fiscalperiod.FiscalPeriod
	)
	run := func(runCtx context.Context, forUpdate bool) error {
		var err error
		state, err = s.loadTransitionState(runCtx, id, tenantInfo, forUpdate)
		if err != nil {
			return err
		}

		if err = t.validate(runCtx, state); err != nil {
			return err
		}

		updated, err = t.apply(runCtx, state)
		return err
	}

	var err error
	if s.db == nil {
		err = run(ctx, false)
	} else {
		err = s.db.WithTx(
			ctx,
			ports.TxOptions{LockTimeout: fiscalPeriodLockTimeout},
			func(txCtx context.Context, _ bun.Tx) error {
				return run(txCtx, true)
			},
		)
		if err != nil {
			err = dberror.MapRetryableTransactionError(
				err,
				"The fiscal period is busy. Retry the request.",
			)
		}
	}
	if err != nil {
		log.Error("failed to transition fiscal period", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceFiscalPeriod,
		ResourceID:     updated.GetID().String(),
		Operation:      t.operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(state.period),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	},
		auditservice.WithComment(t.comment),
		auditservice.WithDiff(state.period, updated),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updated, nil
}

func (s *Service) loadTransitionState(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	forUpdate bool,
) (transitionState, error) {
	periodReq := repositories.GetFiscalPeriodByIDRequest{ID: id, TenantInfo: tenantInfo}

	probe, err := s.repo.GetByID(ctx, periodReq)
	if err != nil {
		return transitionState{}, err
	}

	yearReq := repositories.GetFiscalYearByIDRequest{
		ID:         probe.FiscalYearID,
		TenantInfo: tenantInfo,
	}
	periodsReq := repositories.ListByFiscalYearIDRequest{
		FiscalYearID: probe.FiscalYearID,
		OrgID:        tenantInfo.OrgID,
		BuID:         tenantInfo.BuID,
	}

	if !forUpdate {
		year, yearErr := s.fiscalYearRepo.GetByID(ctx, yearReq)
		if yearErr != nil {
			return transitionState{}, yearErr
		}

		periods, listErr := s.repo.ListByFiscalYearID(ctx, periodsReq)
		if listErr != nil {
			return transitionState{}, listErr
		}

		return transitionState{period: probe, periods: periods, year: year}, nil
	}

	year, err := s.fiscalYearRepo.GetByIDForUpdate(ctx, yearReq)
	if err != nil {
		return transitionState{}, err
	}

	period, err := s.repo.GetByIDForUpdate(ctx, periodReq)
	if err != nil {
		return transitionState{}, err
	}

	periods, err := s.repo.ListByFiscalYearIDForUpdate(ctx, periodsReq)
	if err != nil {
		return transitionState{}, err
	}

	return transitionState{period: period, periods: periods, year: year}, nil
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

func (s *Service) validateCloseWithPeriods(
	entity *fiscalperiod.FiscalPeriod,
	periods []*fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if !entity.Status.CanClose() {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Only Open or Locked fiscal periods can be closed. Current status: {0}",
			entity.Status,
		)
		return multiErr
	}

	for _, p := range periods {
		if p == nil || p.PeriodNumber >= entity.PeriodNumber {
			continue
		}

		switch p.Status { //nolint:exhaustive // closed and locked predecessors do not block
		case fiscalperiod.StatusOpen:
			multiErr.Add(
				"status",
				errortypes.ErrInvalid,
				"Cannot close period {0}: period {1} is still open. Close periods sequentially.",
				entity.PeriodNumber,
				p.PeriodNumber,
			)
		case fiscalperiod.StatusInactive:
			multiErr.Add(
				"status",
				errortypes.ErrInvalid,
				"Cannot close period {0}: period {1} has never been opened. Close periods sequentially.",
				entity.PeriodNumber,
				p.PeriodNumber,
			)
		}

		if multiErr.HasErrors() {
			return multiErr
		}
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

func validateReopen(state transitionState, reason string) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity := state.period

	if !entity.Status.CanReopen() {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Only Closed fiscal periods can be reopened. Current status: {0}", entity.Status,
		)
		return multiErr
	}

	if strings.TrimSpace(reason) == "" {
		multiErr.Add("reopenReason", errortypes.ErrRequired, "A reason for reopening is required")
	}

	addClosedYearError(multiErr, state.year, entity)

	for _, p := range state.periods {
		if p != nil && p.PeriodNumber > entity.PeriodNumber &&
			(p.Status == fiscalperiod.StatusClosed || p.Status == fiscalperiod.StatusLocked) {
			multiErr.Add(
				"status",
				errortypes.ErrInvalid,
				"Cannot reopen period {0}: period {1} is already closed. Reopen periods in reverse order.",
				entity.PeriodNumber,
				p.PeriodNumber,
			)
			break
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateLock(entity *fiscalperiod.FiscalPeriod) *errortypes.MultiError {
	if entity.Status.CanLock() {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"status",
		errortypes.ErrInvalid,
		"Only Open fiscal periods can be locked. Current status: {0}", entity.Status,
	)

	return multiErr
}

func validateUnlock(state transitionState) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if !state.period.Status.CanUnlock() {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Only Locked fiscal periods can be unlocked. Current status: {0}", state.period.Status,
		)
		return multiErr
	}

	addClosedYearError(multiErr, state.year, state.period)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateActivate(state transitionState) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity := state.period

	if !entity.Status.CanActivate() {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Only Inactive fiscal periods can be opened. Current status: {0}", entity.Status,
		)
		return multiErr
	}

	addClosedYearError(multiErr, state.year, entity)

	for _, p := range state.periods {
		if p != nil && p.PeriodNumber < entity.PeriodNumber &&
			p.Status == fiscalperiod.StatusInactive {
			multiErr.Add(
				"status",
				errortypes.ErrInvalid,
				"Cannot open period {0}: period {1} has not been opened yet. Open periods in order.",
				entity.PeriodNumber,
				p.PeriodNumber,
			)
			break
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func addClosedYearError(
	multiErr *errortypes.MultiError,
	year *fiscalyear.FiscalYear,
	period *fiscalperiod.FiscalPeriod,
) {
	if year == nil || !year.Status.IsClosed() {
		return
	}

	if year.Status == fiscalyear.StatusPermanentlyClosed {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Period {0} belongs to fiscal year {1}, which is permanently closed.",
			period.PeriodNumber,
			year.Name,
		)
		return
	}

	multiErr.Add(
		"status",
		errortypes.ErrInvalid,
		"Period {0} belongs to fiscal year {1}, which is closed. Reopen the fiscal year first.",
		period.PeriodNumber,
		year.Name,
	)
}

func validateEditable(
	original *fiscalperiod.FiscalPeriod,
	updated *fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if original.Status == fiscalperiod.StatusPermanentlyClosed {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Permanently closed fiscal periods cannot be changed",
		)
		return multiErr
	}

	if original.Status == fiscalperiod.StatusInactive {
		return nil
	}

	if updated.PeriodNumber != original.PeriodNumber {
		addStructureLocked(multiErr, "periodNumber")
	}
	if updated.PeriodType != original.PeriodType {
		addStructureLocked(multiErr, "periodType")
	}
	if updated.IsAdjusting != original.IsAdjusting {
		addStructureLocked(multiErr, "isAdjusting")
	}
	if updated.StartDate != original.StartDate {
		addStructureLocked(multiErr, "startDate")
	}
	if updated.EndDate != original.EndDate {
		addStructureLocked(multiErr, "endDate")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func addStructureLocked(multiErr *errortypes.MultiError, field string) {
	multiErr.Add(
		field,
		errortypes.ErrInvalid,
		"This field can only be changed before the period is opened",
	)
}

func preserveLifecycle(original, updated *fiscalperiod.FiscalPeriod) {
	updated.FiscalYearID = original.FiscalYearID
	updated.Status = original.Status
	updated.LockedAt = original.LockedAt
	updated.LockedByID = original.LockedByID
	updated.ClosedAt = original.ClosedAt
	updated.ClosedByID = original.ClosedByID
	updated.ReopenedAt = original.ReopenedAt
	updated.ReopenedByID = original.ReopenedByID
	updated.ReopenReason = original.ReopenReason
	updated.CreatedAt = original.CreatedAt
}
