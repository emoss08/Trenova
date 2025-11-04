package fiscalperiod

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/fiscalperiodvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger           *zap.Logger
	Repo             repositories.FiscalPeriodRepository
	AuditService     services.AuditService
	PermissionEngine ports.PermissionEngine
	Validator        *fiscalperiodvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.FiscalPeriodRepository
	pe   ports.PermissionEngine
	as   services.AuditService
	v    *fiscalperiodvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.fiscalperiod"),
		repo: p.Repo,
		pe:   p.PermissionEngine,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListFiscalPeriodRequest,
) (*pagination.ListResult[*accounting.FiscalPeriod], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetFiscalPeriodByIDRequest,
) (*accounting.FiscalPeriod, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByNumber(
	ctx context.Context,
	req *repositories.GetFiscalPeriodByNumberRequest,
) (*accounting.FiscalPeriod, error) {
	return s.repo.GetByNumber(ctx, req)
}

func (s *Service) GetByFiscalYear(
	ctx context.Context,
	req *repositories.GetFiscalPeriodsByYearRequest,
) ([]*accounting.FiscalPeriod, error) {
	return s.repo.GetByFiscalYear(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *accounting.FiscalPeriod,
	userID pulid.ID,
) (*accounting.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Fiscal period created"),
	)
	if err != nil {
		log.Error("failed to log fiscal period creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *accounting.FiscalPeriod,
	userID pulid.ID,
) (*accounting.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetFiscalPeriodByIDRequest{
		FiscalPeriodID: entity.ID,
		OrgID:          entity.OrganizationID,
		BuID:           entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update fiscal period", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Fiscal period updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal period update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *repositories.DeleteFiscalPeriodRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalPeriodByIDRequest{
		FiscalPeriodID: req.FiscalPeriodID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal period", zap.Error(err))
		return err
	}

	if err = s.validateDelete(existing); err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete fiscal period", zap.Error(err))
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpDelete,
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			UserID:         req.UserID,
		},
		audit.WithComment("Fiscal period deleted"),
		audit.WithDiff(existing, nil),
	)
	if err != nil {
		log.Error("failed to log fiscal period deletion", zap.Error(err))
	}

	return nil
}

func (s *Service) validateDelete(entity *accounting.FiscalPeriod) error {
	me := errortypes.NewMultiError()

	if entity.Status != accounting.PeriodStatusOpen {
		me.Add("status", errortypes.ErrInvalid, "Only Open fiscal periods can be deleted")
	}

	// TODO: Add check for transactions
	// if hasTransactions(fp.ID) {
	//     me.Add("transactions", errortypes.ErrInvalid,
	//         "Cannot delete fiscal period with existing transactions")
	// }

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) Close(
	ctx context.Context,
	req *repositories.CloseFiscalPeriodRequest,
) (*accounting.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Close"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalPeriodByIDRequest{
		FiscalPeriodID: req.FiscalPeriodID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal period", zap.Error(err))
		return nil, err
	}

	if err = s.validateClose(existing); err != nil {
		return nil, err
	}

	closedEntity, err := s.repo.Close(ctx, req)
	if err != nil {
		log.Error("failed to close fiscal period", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     closedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.ClosedByID,
			PreviousState:  jsonutils.MustToJSON(existing),
			CurrentState:   jsonutils.MustToJSON(closedEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Fiscal period closed"),
		audit.WithDiff(existing, closedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal period close", zap.Error(err))
	}

	return closedEntity, nil
}

func (s *Service) validateClose(fp *accounting.FiscalPeriod) error {
	me := errortypes.NewMultiError()

	if fp.Status != accounting.PeriodStatusOpen {
		me.Add("status", errortypes.ErrInvalid,
			fmt.Sprintf("Only Open fiscal periods can be closed. Current status: %s", fp.Status))
	}

	// TODO: Period-end checklist validations
	// - All transactions posted
	// - All reconciliations complete
	// - No pending adjustments

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) Reopen(
	ctx context.Context,
	req *repositories.ReopenFiscalPeriodRequest,
) (*accounting.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Reopen"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalPeriodByIDRequest{
		FiscalPeriodID: req.FiscalPeriodID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal period", zap.Error(err))
		return nil, err
	}

	if err = s.validateReopen(existing); err != nil {
		return nil, err
	}

	reopenedEntity, err := s.repo.Reopen(ctx, req)
	if err != nil {
		log.Error("failed to reopen fiscal period", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     reopenedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(existing),
			CurrentState:   jsonutils.MustToJSON(reopenedEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Fiscal period reopened"),
		audit.WithDiff(existing, reopenedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal period reopen", zap.Error(err))
	}

	return reopenedEntity, nil
}

func (s *Service) validateReopen(fp *accounting.FiscalPeriod) error {
	me := errortypes.NewMultiError()

	if fp.Status != accounting.PeriodStatusClosed {
		me.Add(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Only Closed fiscal periods can be reopened. Current status: %s",
				fp.Status,
			),
		)
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) Lock(
	ctx context.Context,
	req *repositories.LockFiscalPeriodRequest,
) (*accounting.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Lock"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalPeriodByIDRequest{
		FiscalPeriodID: req.FiscalPeriodID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal period", zap.Error(err))
		return nil, err
	}

	if err = s.validateLock(existing); err != nil {
		return nil, err
	}

	lockedEntity, err := s.repo.Lock(ctx, req)
	if err != nil {
		log.Error("failed to lock fiscal period", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     lockedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(existing),
			CurrentState:   jsonutils.MustToJSON(lockedEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Fiscal period locked"),
		audit.WithDiff(existing, lockedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal period lock", zap.Error(err))
	}

	return lockedEntity, nil
}

func (s *Service) validateLock(fp *accounting.FiscalPeriod) error {
	me := errortypes.NewMultiError()

	if fp.Status != accounting.PeriodStatusClosed {
		me.Add("status", errortypes.ErrInvalid,
			fmt.Sprintf("Only Closed fiscal periods can be locked. Current status: %s", fp.Status))
	}

	// TODO: Additional validations
	// - No pending adjusting entries
	// - Management approval obtained

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) Unlock(
	ctx context.Context,
	req *repositories.UnlockFiscalPeriodRequest,
) (*accounting.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "Unlock"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalPeriodByIDRequest{
		FiscalPeriodID: req.FiscalPeriodID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal period", zap.Error(err))
		return nil, err
	}

	result, err := s.pe.Check(ctx, &ports.PermissionCheckRequest{
		UserID:         req.UserID,
		OrganizationID: req.OrgID,
		ResourceType:   permission.ResourceFiscalPeriod.String(),
		Action:         permission.OpUnlock.String(),
	})
	if err != nil {
		return nil, err
	}

	if !result.Allowed {
		return nil, errortypes.NewAuthorizationError("unlock fiscal period")
	}

	if err = s.validateUnlock(existing); err != nil {
		return nil, err
	}

	unlockedEntity, err := s.repo.Unlock(ctx, req)
	if err != nil {
		log.Error("failed to unlock fiscal period", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalPeriod,
			ResourceID:     unlockedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(existing),
			CurrentState:   jsonutils.MustToJSON(unlockedEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Fiscal period unlocked"),
		audit.WithDiff(existing, unlockedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal period unlock", zap.Error(err))
	}

	return unlockedEntity, nil
}

func (s *Service) validateUnlock(fp *accounting.FiscalPeriod) error {
	me := errortypes.NewMultiError()

	if fp.Status != accounting.PeriodStatusLocked {
		me.Add(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Only Locked fiscal periods can be unlocked. Current status: %s",
				fp.Status,
			),
		)
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) GeneratePeriodsForFiscalYear(
	ctx context.Context,
	fiscalYear *accounting.FiscalYear,
) ([]*accounting.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "GeneratePeriodsForFiscalYear"),
		zap.String("fiscalYearId", fiscalYear.ID.String()),
		zap.Int("year", fiscalYear.Year),
	)

	periods := make([]*accounting.FiscalPeriod, 0, 12)

	startTime := time.Unix(fiscalYear.StartDate, 0)
	endTime := time.Unix(fiscalYear.EndDate, 0)

	totalDays := endTime.Sub(startTime).Hours() / 24

	currentStart := startTime
	for i := 1; i <= 12; i++ {
		var periodEnd time.Time

		if i == 12 {
			periodEnd = endTime
		} else {
			daysInPeriod := totalDays / 12
			periodEnd = currentStart.Add(time.Duration(daysInPeriod*24) * time.Hour)

			periodEnd = time.Date(
				periodEnd.Year(),
				periodEnd.Month(),
				periodEnd.Day(),
				23, 59, 59, 0,
				periodEnd.Location(),
			)
		}

		periodName := fmt.Sprintf("Period %d - %s", i, currentStart.Format("January 2006"))

		period := &accounting.FiscalPeriod{
			FiscalYearID:   fiscalYear.ID,
			OrganizationID: fiscalYear.OrganizationID,
			BusinessUnitID: fiscalYear.BusinessUnitID,
			PeriodNumber:   i,
			PeriodType:     accounting.PeriodTypeMonth,
			Name:           periodName,
			StartDate:      currentStart.Unix(),
			EndDate:        periodEnd.Unix(),
			Status:         accounting.PeriodStatusOpen,
		}

		periods = append(periods, period)

		currentStart = periodEnd.Add(time.Second)
	}

	err := s.repo.BulkCreate(ctx, &repositories.BulkCreateFiscalPeriodsRequest{
		Periods: periods,
		OrgID:   fiscalYear.OrganizationID,
		BuID:    fiscalYear.BusinessUnitID,
	})
	if err != nil {
		log.Error("failed to bulk create fiscal periods", zap.Error(err))
		return nil, err
	}

	log.Info("successfully generated fiscal periods",
		zap.Int("count", len(periods)),
		zap.String("fiscalYearId", fiscalYear.ID.String()),
	)

	return periods, nil
}

func (s *Service) GenerateQuarterlyPeriods(
	ctx context.Context,
	fiscalYear *accounting.FiscalYear,
) ([]*accounting.FiscalPeriod, error) {
	log := s.l.With(
		zap.String("operation", "GenerateQuarterlyPeriods"),
		zap.String("fiscalYearId", fiscalYear.ID.String()),
		zap.Int("year", fiscalYear.Year),
	)

	periods := make([]*accounting.FiscalPeriod, 0, 4)

	startTime := time.Unix(fiscalYear.StartDate, 0)
	endTime := time.Unix(fiscalYear.EndDate, 0)

	totalDays := endTime.Sub(startTime).Hours() / 24

	currentStart := startTime
	for i := 1; i <= 4; i++ {
		var periodEnd time.Time

		if i == 4 {
			periodEnd = endTime
		} else {
			daysInQuarter := totalDays / 4
			periodEnd = currentStart.Add(time.Duration(daysInQuarter*24) * time.Hour)

			periodEnd = time.Date(
				periodEnd.Year(),
				periodEnd.Month(),
				periodEnd.Day(),
				23, 59, 59, 0,
				periodEnd.Location(),
			)
		}

		periodName := fmt.Sprintf("Q%d %d", i, fiscalYear.Year)

		period := &accounting.FiscalPeriod{
			FiscalYearID:   fiscalYear.ID,
			OrganizationID: fiscalYear.OrganizationID,
			BusinessUnitID: fiscalYear.BusinessUnitID,
			PeriodNumber:   i,
			PeriodType:     accounting.PeriodTypeQuarter,
			Name:           periodName,
			StartDate:      currentStart.Unix(),
			EndDate:        periodEnd.Unix(),
			Status:         accounting.PeriodStatusOpen,
		}

		periods = append(periods, period)

		currentStart = periodEnd.Add(time.Second)
	}

	err := s.repo.BulkCreate(ctx, &repositories.BulkCreateFiscalPeriodsRequest{
		Periods: periods,
		OrgID:   fiscalYear.OrganizationID,
		BuID:    fiscalYear.BusinessUnitID,
	})
	if err != nil {
		log.Error("failed to bulk create quarterly periods", zap.Error(err))
		return nil, err
	}

	log.Info("successfully generated quarterly periods",
		zap.Int("count", len(periods)),
		zap.String("fiscalYearId", fiscalYear.ID.String()),
	)

	return periods, nil
}
