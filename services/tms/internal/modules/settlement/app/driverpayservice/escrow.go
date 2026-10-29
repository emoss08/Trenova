package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) ListEscrowAccounts(
	ctx context.Context,
	req *driverpayports.ListEscrowAccountsRequest,
) (*pagination.ListResult[*payescrow.EscrowAccount], error) {
	return s.escrowRepo.List(ctx, req)
}

func (s *Service) ListEscrowAccountsConnection(
	ctx context.Context,
	req *driverpayports.ListEscrowAccountConnectionRequest,
) (*pagination.CursorListResult[*payescrow.EscrowAccount], error) {
	return s.escrowRepo.ListConnection(ctx, req)
}

func (s *Service) GetEscrowAccount(
	ctx context.Context,
	req driverpayports.GetEscrowAccountByIDRequest,
) (*payescrow.EscrowAccount, error) {
	return s.escrowRepo.GetByID(ctx, req)
}

func (s *Service) OpenEscrowAccount(
	ctx context.Context,
	entity *payescrow.EscrowAccount,
	actor *permissionports.RequestActor,
) (*payescrow.EscrowAccount, error) {
	if err := requireActor(actor, "Escrow account opening"); err != nil {
		return nil, err
	}
	if err := s.PlanOpenEscrowAccount(ctx, entity, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	created, err := s.escrowRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logEscrowAudit(created, nil, actor.UserID, permission.OpCreate, "Escrow account opened")
	return created, nil
}

func (s *Service) UpdateEscrowAccount(
	ctx context.Context,
	entity *payescrow.EscrowAccount,
	actor *permissionports.RequestActor,
) (*payescrow.EscrowAccount, error) {
	if err := requireActor(actor, "Escrow account update"); err != nil {
		return nil, err
	}
	previous, err := s.PlanUpdateEscrowAccount(ctx, entity)
	if err != nil {
		return nil, err
	}

	updated, err := s.escrowRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logEscrowAudit(updated, previous, actor.UserID, permission.OpUpdate,
		"Escrow account updated")
	return updated, nil
}

type EscrowAdjustmentRequest struct {
	TenantInfo   pagination.TenantInfo
	AccountID    pulid.ID
	AmountMinor  int64
	Description  string
	OccurredDate int64
}

func (s *Service) AdjustEscrowAccount(
	ctx context.Context,
	req *EscrowAdjustmentRequest,
	actor *permissionports.RequestActor,
) (*payescrow.EscrowAccount, error) {
	if err := requireActor(actor, "Escrow adjustment"); err != nil {
		return nil, err
	}
	if err := CheckEscrowAdjustment(req); err != nil {
		return nil, err
	}
	if req.OccurredDate == 0 {
		req.OccurredDate = timeutils.NowUnix()
	}

	var updated *payescrow.EscrowAccount
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		account, txErr := s.escrowRepo.GetByID(txCtx, driverpayports.GetEscrowAccountByIDRequest{
			ID:         req.AccountID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		transaction, txErr := PlanEscrowAdjustment(account, req, actor.UserID)
		if txErr != nil {
			return txErr
		}
		updated, txErr = s.applyEscrowTransaction(txCtx, account, transaction)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logEscrowAudit(updated, nil, actor.UserID, permission.OpUpdate,
		"Escrow adjustment: "+req.Description)
	return updated, nil
}

func (s *Service) CloseEscrowAccount(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	accountID pulid.ID,
	actor *permissionports.RequestActor,
) (*payescrow.EscrowAccount, error) {
	if err := requireActor(actor, "Escrow account closure"); err != nil {
		return nil, err
	}
	var updated *payescrow.EscrowAccount
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		account, txErr := s.escrowRepo.GetByID(txCtx, driverpayports.GetEscrowAccountByIDRequest{
			ID:         accountID,
			TenantInfo: tenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		now := timeutils.NowUnix()
		refund, txErr := PlanCloseEscrowAccount(account, actor.UserID, now)
		if txErr != nil {
			return txErr
		}
		if refund != nil {
			account, txErr = s.applyEscrowTransaction(txCtx, account, refund)
			if txErr != nil {
				return txErr
			}
		}

		CloseEscrowAccount(account, now)
		updated, txErr = s.escrowRepo.Update(txCtx, account)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logEscrowAudit(updated, nil, actor.UserID, permission.OpClose,
		"Escrow account closed; remaining balance refunded per 49 CFR 376.12(k)")
	return updated, nil
}

func (s *Service) AccrueEscrowInterest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	accountID pulid.ID,
) (*payescrow.EscrowAccount, error) {
	var updated *payescrow.EscrowAccount
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		account, txErr := s.escrowRepo.GetByID(txCtx, driverpayports.GetEscrowAccountByIDRequest{
			ID:         accountID,
			TenantInfo: tenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if account.Status != payescrow.EscrowAccountStatusActive ||
			account.BalanceMinor <= 0 ||
			account.AnnualInterestRate.IsZero() {
			updated = account
			return nil
		}

		now := timeutils.NowUnix()
		accrualStart := account.OpenedDate
		if account.LastInterestAccrualDate != nil {
			accrualStart = *account.LastInterestAccrualDate
		}
		elapsedSeconds := now - accrualStart
		if elapsedSeconds <= 0 {
			updated = account
			return nil
		}

		yearFraction := decimal.NewFromInt(elapsedSeconds).
			Div(decimal.NewFromInt(365 * 24 * 3600))
		interest := decimal.NewFromInt(account.BalanceMinor).
			Mul(account.AnnualInterestRate).
			Div(decimal.NewFromInt(100)).
			Mul(yearFraction).
			Round(0).
			IntPart()
		if interest <= 0 {
			account.LastInterestAccrualDate = &now
			updated, txErr = s.escrowRepo.Update(txCtx, account)
			return txErr
		}

		account, txErr = s.applyEscrowTransaction(txCtx, account, &payescrow.EscrowTransaction{
			Type:         payescrow.EscrowTransactionTypeInterestAccrual,
			AmountMinor:  interest,
			OccurredDate: now,
			Description:  "Quarterly escrow interest accrual per 49 CFR 376.12(k)",
		})
		if txErr != nil {
			return txErr
		}
		account.LastInterestAccrualDate = &now
		updated, txErr = s.escrowRepo.Update(txCtx, account)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) applyEscrowTransaction(
	ctx context.Context,
	account *payescrow.EscrowAccount,
	tx *payescrow.EscrowTransaction,
) (*payescrow.EscrowAccount, error) {
	StampEscrowTransaction(account, tx)

	multiErr := errortypes.NewMultiError()
	tx.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if _, err := s.escrowRepo.AppendTransaction(ctx, tx); err != nil {
		return nil, err
	}
	account.BalanceMinor = tx.BalanceAfterMinor
	return s.escrowRepo.Update(ctx, account)
}

func (s *Service) logEscrowAudit(
	current, previous *payescrow.EscrowAccount,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceEscrowAccount,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []auditports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log escrow account audit action", zap.Error(err))
	}
}
