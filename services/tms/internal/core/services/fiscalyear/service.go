package fiscalyear

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/services/fiscalperiod"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/fiscalyearvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger              *zap.Logger
	Repo                repositories.FiscalYearRepository
	AuditService        services.AuditService
	FiscalPeriodService *fiscalperiod.Service
	PermissionEngine    ports.PermissionEngine
	Validator           *fiscalyearvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.FiscalYearRepository
	fps  *fiscalperiod.Service
	pe   ports.PermissionEngine
	as   services.AuditService
	v    *fiscalyearvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.fiscalyear"),
		repo: p.Repo,
		fps:  p.FiscalPeriodService,
		pe:   p.PermissionEngine,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListFiscalYearRequest,
) (*pagination.ListResult[*accounting.FiscalYear], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetFiscalYearByIDRequest,
) (*accounting.FiscalYear, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByYear(
	ctx context.Context,
	req *repositories.GetFiscalYearByYearRequest,
) (*accounting.FiscalYear, error) {
	return s.repo.GetByYear(ctx, req)
}

func (s *Service) GetCurrent(
	ctx context.Context,
	req *repositories.GetCurrentFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	return s.repo.GetCurrent(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *accounting.FiscalYear,
	userID pulid.ID,
) (*accounting.FiscalYear, error) {
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

	_, err = s.fps.GeneratePeriodsForFiscalYear(ctx, createdEntity)
	if err != nil {
		log.Error("failed to auto-generate fiscal periods", zap.Error(err))
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalYear,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Fiscal year created"),
	)
	if err != nil {
		log.Error("failed to log fiscal year creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *accounting.FiscalYear,
	userID pulid.ID,
) (*accounting.FiscalYear, error) {
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

	original, err := s.repo.GetByID(ctx, &repositories.GetFiscalYearByIDRequest{
		FiscalYearID: entity.ID,
		OrgID:        entity.OrganizationID,
		BuID:         entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update fiscal year", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalYear,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Fiscal year updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal year update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *repositories.DeleteFiscalYearRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalYearByIDRequest{
		FiscalYearID: req.FiscalYearID,
		OrgID:        req.OrgID,
		BuID:         req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal year", zap.Error(err))
		return err
	}

	if err = s.validateDelete(existing); err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete fiscal year", zap.Error(err))
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalYear,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpDelete,
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			UserID:         req.UserID,
		},
		audit.WithComment("Fiscal year deleted"),
		audit.WithDiff(existing, nil),
	)
	if err != nil {
		log.Error("failed to log fiscal year deletion", zap.Error(err))
	}

	return nil
}

func (s *Service) validateDelete(entity *accounting.FiscalYear) error {
	me := errortypes.NewMultiError()

	if entity.Status != accounting.FiscalYearStatusDraft {
		me.Add("status", errortypes.ErrInvalid, "Only Draft fiscal years can be deleted")
	}

	if entity.IsCurrent {
		me.Add("isCurrent", errortypes.ErrInvalid, "Current fiscal year cannot be deleted")
	}

	// TODO: Add check for transactions
	// if hasTransactions(fy.ID) {
	//     me.Add("transactions", errortypes.ErrInvalid,
	//         "Cannot delete fiscal year with existing transactions")
	// }

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) Close(
	ctx context.Context,
	req *repositories.CloseFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Close"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalYearByIDRequest{
		FiscalYearID: req.FiscalYearID,
		OrgID:        req.OrgID,
		BuID:         req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal year", zap.Error(err))
		return nil, err
	}

	if err = s.validateClose(existing); err != nil {
		return nil, err
	}

	closedEntity, err := s.repo.Close(ctx, req)
	if err != nil {
		log.Error("failed to close fiscal year", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalYear,
			ResourceID:     closedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.ClosedByID,
			PreviousState:  jsonutils.MustToJSON(existing),
			CurrentState:   jsonutils.MustToJSON(closedEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Fiscal year closed"),
		audit.WithDiff(existing, closedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal year close", zap.Error(err))
	}

	return closedEntity, nil
}

func (s *Service) validateClose(fy *accounting.FiscalYear) error {
	me := errortypes.NewMultiError()

	if fy.Status != accounting.FiscalYearStatusOpen {
		me.Add("status", errortypes.ErrInvalid,
			fmt.Sprintf("Only Open fiscal years can be closed. Current status: %s", fy.Status))
	}

	// TODO: Year-end checklist validations
	// - All shipments billed
	// - All AP entries recorded
	// - Depreciation posted
	// - Trial balance verified (debits = credits)
	// - Bank reconciliation complete

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) Lock(
	ctx context.Context,
	req *repositories.LockFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Lock"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalYearByIDRequest{
		FiscalYearID: req.FiscalYearID,
		OrgID:        req.OrgID,
		BuID:         req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal year", zap.Error(err))
		return nil, err
	}

	if err = s.validateLock(existing); err != nil {
		return nil, err
	}

	lockedEntity, err := s.repo.Lock(ctx, req)
	if err != nil {
		log.Error("failed to lock fiscal year", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalYear,
			ResourceID:     lockedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.LockedByID,
			PreviousState:  jsonutils.MustToJSON(existing),
			CurrentState:   jsonutils.MustToJSON(lockedEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Fiscal year locked"),
		audit.WithDiff(existing, lockedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal year lock", zap.Error(err))
	}

	return lockedEntity, nil
}

func (s *Service) validateLock(fy *accounting.FiscalYear) error {
	me := errortypes.NewMultiError()

	if fy.Status != accounting.FiscalYearStatusClosed {
		me.Add("status", errortypes.ErrInvalid,
			fmt.Sprintf("Only Closed fiscal years can be locked. Current status: %s", fy.Status))
	}

	// TODO: Additional validations
	// - No pending adjusting entries
	// - Adjustment period has expired
	// - Management approval obtained

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) Unlock(
	ctx context.Context,
	req *repositories.UnlockFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Unlock"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalYearByIDRequest{
		FiscalYearID: req.FiscalYearID,
		OrgID:        req.OrgID,
		BuID:         req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal year", zap.Error(err))
		return nil, err
	}

	result, err := s.pe.Check(ctx, &ports.PermissionCheckRequest{
		UserID:         req.UserID,
		OrganizationID: req.OrgID,
		ResourceType:   permission.ResourceFiscalYear.String(),
		Action:         permission.OpUpdate.String(),
	})
	if err != nil {
		return nil, err
	}

	isAdmin, err := s.pe.HasAdminRole(ctx, req.UserID, req.OrgID)
	if err != nil {
		return nil, err
	}

	// If the user not admin or not allowed to update the fiscal year, return an error
	if !isAdmin || !result.Allowed {
		return nil, errortypes.NewAuthorizationError(
			"User does not have permission to unlock fiscal year",
		)
	}

	unlockedEntity, err := s.repo.Unlock(ctx, req)
	if err != nil {
		log.Error("failed to unlock fiscal year", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalYear,
			ResourceID:     unlockedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(existing),
			CurrentState:   jsonutils.MustToJSON(unlockedEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Fiscal year unlocked"),
		audit.WithDiff(existing, unlockedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal year unlock", zap.Error(err))
	}

	return unlockedEntity, nil
}

func (s *Service) Activate(
	ctx context.Context,
	req *repositories.ActivateFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := s.l.With(
		zap.String("operation", "Activate"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetFiscalYearByIDRequest{
		FiscalYearID: req.FiscalYearID,
		OrgID:        req.OrgID,
		BuID:         req.BuID,
	})
	if err != nil {
		log.Error("failed to get fiscal year", zap.Error(err))
		return nil, err
	}

	if err = s.validateActivate(ctx, req, existing); err != nil {
		return nil, err
	}

	activatedEntity, err := s.repo.Activate(ctx, req)
	if err != nil {
		log.Error("failed to activate fiscal year", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFiscalYear,
			ResourceID:     activatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(existing),
			CurrentState:   jsonutils.MustToJSON(activatedEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Fiscal year activated"),
		audit.WithDiff(existing, activatedEntity),
	)
	if err != nil {
		log.Error("failed to log fiscal year activate", zap.Error(err))
	}

	return activatedEntity, nil
}

func (s *Service) validateActivate(
	ctx context.Context,
	req *repositories.ActivateFiscalYearRequest,
	fy *accounting.FiscalYear,
) error {
	me := errortypes.NewMultiError()

	if fy.Status == accounting.FiscalYearStatusLocked {
		me.Add("status", errortypes.ErrInvalid,
			"Cannot activate a locked fiscal year")
	}

	currentFY, err := s.repo.GetCurrent(ctx, &repositories.GetCurrentFiscalYearRequest{
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})

	// If there's a current year, it should be closed before activating next
	if err == nil && currentFY != nil {
		if currentFY.Status == accounting.FiscalYearStatusOpen {
			me.Add("previousYear", errortypes.ErrInvalid,
				fmt.Sprintf("Current fiscal year %d must be closed before activating %d",
					currentFY.Year, fy.Year))
		}

		// Verify this is the next sequential year
		if fy.Year != currentFY.Year+1 {
			me.Add("year", errortypes.ErrInvalid,
				fmt.Sprintf("Fiscal year %d should be activated after %d",
					currentFY.Year+1, currentFY.Year))
		}
	}

	currentYear := utils.GetCurrentYear()
	if fy.Year > currentYear+2 {
		me.Add(
			"year",
			errortypes.ErrInvalid,
			"Cannot activate fiscal year more than 2 years in advance")
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) CheckOverlappingFiscalYears(
	ctx context.Context,
	req *repositories.CheckOverlappingFiscalYearsRequest,
) ([]*repositories.OverlappingFiscalYearResponse, error) {
	return s.repo.CheckOverlappingFiscalYears(ctx, req)
}
