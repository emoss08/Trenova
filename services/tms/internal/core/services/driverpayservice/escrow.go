package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (s *Service) ListEscrowAccounts(
	ctx context.Context,
	req *repositories.ListEscrowAccountsRequest,
) (*pagination.ListResult[*driverpay.EscrowAccount], error) {
	return s.escrowRepo.List(ctx, req)
}

func (s *Service) ListEscrowAccountsConnection(
	ctx context.Context,
	req *repositories.ListEscrowAccountConnectionRequest,
) (*pagination.CursorListResult[*driverpay.EscrowAccount], error) {
	return s.escrowRepo.ListConnection(ctx, req)
}

func (s *Service) GetEscrowAccount(
	ctx context.Context,
	req repositories.GetEscrowAccountByIDRequest,
) (*driverpay.EscrowAccount, error) {
	return s.escrowRepo.GetByID(ctx, req)
}

func (s *Service) OpenEscrowAccount(
	ctx context.Context,
	entity *driverpay.EscrowAccount,
	actor *serviceports.RequestActor,
) (*driverpay.EscrowAccount, error) {
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
	entity *driverpay.EscrowAccount,
	actor *serviceports.RequestActor,
) (*driverpay.EscrowAccount, error) {
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
	actor *serviceports.RequestActor,
) (*driverpay.EscrowAccount, error) {
	if err := requireActor(actor, "Escrow adjustment"); err != nil {
		return nil, err
	}
	if err := CheckEscrowAdjustment(req); err != nil {
		return nil, err
	}
	if req.OccurredDate == 0 {
		req.OccurredDate = timeutils.NowUnix()
	}

	var updated *driverpay.EscrowAccount
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		account, txErr := s.escrowRepo.GetByID(txCtx, repositories.GetEscrowAccountByIDRequest{
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
	actor *serviceports.RequestActor,
) (*driverpay.EscrowAccount, error) {
	if err := requireActor(actor, "Escrow account closure"); err != nil {
		return nil, err
	}
	var updated *driverpay.EscrowAccount
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		account, txErr := s.escrowRepo.GetByID(txCtx, repositories.GetEscrowAccountByIDRequest{
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
) (*driverpay.EscrowAccount, error) {
	var updated *driverpay.EscrowAccount
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		account, txErr := s.escrowRepo.GetByID(txCtx, repositories.GetEscrowAccountByIDRequest{
			ID:         accountID,
			TenantInfo: tenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if account.Status != driverpay.EscrowAccountStatusActive ||
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

		account, txErr = s.applyEscrowTransaction(txCtx, account, &driverpay.EscrowTransaction{
			Type:         driverpay.EscrowTransactionTypeInterestAccrual,
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
	account *driverpay.EscrowAccount,
	tx *driverpay.EscrowTransaction,
) (*driverpay.EscrowAccount, error) {
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
	current, previous *driverpay.EscrowAccount,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceEscrowAccount,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log escrow account audit action", zap.Error(err))
	}
}
