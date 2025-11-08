package glaccount

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/glaccountvalidator"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger           *zap.Logger
	DB               *postgres.Connection
	Repo             repositories.GLAccountRepository
	JournalEntryRepo repositories.JournalEntryRepository
	AuditService     services.AuditService
	PermissionEngine ports.PermissionEngine
	Validator        *glaccountvalidator.Validator
}

type Service struct {
	l      *zap.Logger
	db     *postgres.Connection
	repo   repositories.GLAccountRepository
	jeRepo repositories.JournalEntryRepository
	pe     ports.PermissionEngine
	as     services.AuditService
	v      *glaccountvalidator.Validator
}

//nolint:gocritic // This is a constructor
func NewService(p ServiceParams) *Service {
	return &Service{
		l:      p.Logger.Named("service.glaccount"),
		db:     p.DB,
		repo:   p.Repo,
		jeRepo: p.JournalEntryRepo,
		pe:     p.PermissionEngine,
		as:     p.AuditService,
		v:      p.Validator,
	}
}

func (s *Service) GetOption(
	ctx context.Context,
	req repositories.GetGLAccountByIDRequest,
) (*accounting.GLAccount, error) {
	return s.repo.GetOption(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req repositories.GLAccountSelectOptionsRequest,
) ([]*repositories.GLAccountSelectOptionResponse, error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListGLAccountRequest,
) (*pagination.ListResult[*accounting.GLAccount], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetGLAccountByIDRequest,
) (*accounting.GLAccount, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByCode(
	ctx context.Context,
	req *repositories.GetGLAccountByCodeRequest,
) (*accounting.GLAccount, error) {
	return s.repo.GetByCode(ctx, req)
}

func (s *Service) GetByType(
	ctx context.Context,
	req *repositories.GetGLAccountsByTypeRequest,
) ([]*accounting.GLAccount, error) {
	return s.repo.GetByType(ctx, req)
}

func (s *Service) GetByParent(
	ctx context.Context,
	req *repositories.GetGLAccountsByParentRequest,
) ([]*accounting.GLAccount, error) {
	return s.repo.GetByParent(ctx, req)
}

func (s *Service) GetHierarchy(
	ctx context.Context,
	req *repositories.GetGLAccountHierarchyRequest,
) ([]*accounting.GLAccount, error) {
	return s.repo.GetHierarchy(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *accounting.GLAccount,
	userID pulid.ID,
) (*accounting.GLAccount, error) {
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
			Resource:       permission.ResourceGlAccount,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("GL Account created"),
	)
	if err != nil {
		log.Error("failed to log gl account creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *accounting.GLAccount,
	userID pulid.ID,
) (*accounting.GLAccount, error) {
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

	original, err := s.repo.GetByID(ctx, &repositories.GetGLAccountByIDRequest{
		GLAccountID: entity.ID,
		OrgID:       entity.OrganizationID,
		BuID:        entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update gl account", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceGlAccount,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("GL Account updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log gl account update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *repositories.DeleteGLAccountRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetGLAccountByIDRequest{
		GLAccountID: req.GLAccountID,
		OrgID:       req.OrgID,
		BuID:        req.BuID,
	})
	if err != nil {
		log.Error("failed to get gl account", zap.Error(err))
		return err
	}

	if err = s.validateDelete(ctx, existing); err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete gl account", zap.Error(err))
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceGlAccount,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpDelete,
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			UserID:         req.UserID,
		},
		audit.WithComment("GL Account deleted"),
		audit.WithDiff(existing, nil),
	)
	if err != nil {
		log.Error("failed to log gl account deletion", zap.Error(err))
	}

	return nil
}

func (s *Service) validateDelete(ctx context.Context, entity *accounting.GLAccount) error {
	me := errortypes.NewMultiError()

	if entity.IsSystem {
		me.Add("isSystem", errortypes.ErrInvalid, "System accounts cannot be deleted")
	}

	if entity.CurrentBalance != 0 {
		me.Add(
			"currentBalance",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Cannot delete account with non-zero balance ($%.2f). Please transfer or clear the balance first.",
				float64(entity.CurrentBalance)/100.0,
			),
		)
	}

	children, err := s.repo.GetByParent(ctx, &repositories.GetGLAccountsByParentRequest{
		ParentID: entity.ID,
		OrgID:    entity.OrganizationID,
		BuID:     entity.BusinessUnitID,
	})
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check for child accounts")
	} else if len(children) > 0 {
		me.Add(
			"children",
			errortypes.ErrInvalid,
			fmt.Sprintf("Cannot delete account with %d child accounts", len(children)),
		)
	}

	// Check if account has any journal entry lines
	// Note: We check journal entry lines directly since they reference the GL account
	hasTransactions, transactionCount, err := s.checkForJournalEntries(ctx, entity)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check for journal entries")
	} else if hasTransactions {
		me.Add(
			"transactions",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Cannot delete account with %d journal entry transactions. Consider deactivating instead.",
				transactionCount,
			),
		)
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) BulkCreate(
	ctx context.Context,
	accounts []*accounting.GLAccount,
	userID pulid.ID,
	orgID, buID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "BulkCreate"),
		zap.Int("count", len(accounts)),
		zap.String("orgID", orgID.String()),
		zap.String("buID", buID.String()),
	)

	for i, account := range accounts {
		valCtx := &validator.ValidationContext{
			IsCreate: true,
			IsUpdate: false,
		}

		if err := s.v.Validate(ctx, valCtx, account); err != nil {
			return fmt.Errorf(
				"validation failed for account %d (%s): %w",
				i+1,
				account.AccountCode,
				err,
			)
		}
	}

	err := s.repo.BulkCreate(ctx, &repositories.BulkCreateGLAccountsRequest{
		Accounts: accounts,
		OrgID:    orgID,
		BuID:     buID,
	})
	if err != nil {
		log.Error("failed to bulk create gl accounts", zap.Error(err))
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceGlAccount,
			ResourceID:     fmt.Sprintf("bulk_%d_accounts", len(accounts)),
			Operation:      permission.OpImport,
			UserID:         userID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment(fmt.Sprintf("Bulk created %d GL accounts", len(accounts))),
	)
	if err != nil {
		log.Error("failed to log bulk gl account creation", zap.Error(err))
	}

	return nil
}

func (s *Service) UpdateBalance(
	ctx context.Context,
	req *repositories.UpdateGLAccountBalanceRequest,
	userID pulid.ID,
) (*accounting.GLAccount, error) {
	log := s.l.With(
		zap.String("operation", "UpdateBalance"),
		zap.String("glAccountId", req.GLAccountID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetGLAccountByIDRequest{
		GLAccountID: req.GLAccountID,
		OrgID:       req.OrgID,
		BuID:        req.BuID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.UpdateBalance(ctx, req)
	if err != nil {
		log.Error("failed to update gl account balance", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceGlAccount,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("GL Account balance updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log gl account balance update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) checkForJournalEntries(
	ctx context.Context,
	entity *accounting.GLAccount,
) (bool, int, error) {
	db, err := s.db.DB(ctx)
	if err != nil {
		return false, 0, err
	}

	count, err := db.NewSelect().
		Model((*accounting.JournalEntryLine)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("jel.gl_account_id = ?", entity.ID).
				Where("jel.organization_id = ?", entity.OrganizationID).
				Where("jel.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Count(ctx)
	if err != nil {
		return false, 0, err
	}

	return count > 0, count, nil
}
