package settlementjobs

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driverpayservice"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	SettlementControl repositories.SettlementControlRepository
	EscrowRepo        repositories.EscrowAccountRepository
	SettlementService *driversettlementservice.Service
	PayService        *driverpayservice.Service
	Logger            *zap.Logger
}

type Activities struct {
	settlementControl repositories.SettlementControlRepository
	escrowRepo        repositories.EscrowAccountRepository
	settlementService *driversettlementservice.Service
	payService        *driverpayservice.Service
	logger            *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		settlementControl: p.SettlementControl,
		escrowRepo:        p.EscrowRepo,
		settlementService: p.SettlementService,
		payService:        p.PayService,
		logger:            p.Logger.Named("settlement-activities"),
	}
}

func systemActor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType: services.PrincipalTypeSystem,
		PrincipalID:   services.SystemPrincipalID,
		UserID:        services.SystemPrincipalID,
	}
}

func (a *Activities) GenerateSettlementBatchesActivity(
	ctx context.Context,
) (*GenerateSettlementBatchesResult, error) {
	result := &GenerateSettlementBatchesResult{CompletedAt: timeutils.NowUnix()}

	controls, err := a.settlementControl.ListAutoGenerate(ctx)
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()

	for _, control := range controls {
		result.OrganizationsChecked++
		tenantInfo := pagination.TenantInfo{
			OrgID: control.OrganizationID,
			BuID:  control.BusinessUnitID,
		}
		bounds := driversettlementservice.ResolveCurrentPeriod(control, now)

		batch, genErr := a.settlementService.GenerateBatch(
			ctx,
			&driversettlementservice.GenerateBatchRequest{
				TenantInfo:  tenantInfo,
				PeriodStart: bounds.PeriodStart,
				PeriodEnd:   bounds.PeriodEnd,
			},
			systemActor(),
		)
		if genErr != nil {
			if isDuplicatePeriodError(genErr) {
				continue
			}
			result.Failed++
			a.logger.Error("failed to auto-generate settlement batch",
				zap.Error(genErr),
				zap.String("orgId", control.OrganizationID.String()))
			continue
		}
		if batch != nil {
			result.BatchesGenerated++
		}
	}
	return result, nil
}

func (a *Activities) AccrueEscrowInterestActivity(
	ctx context.Context,
) (*AccrueEscrowInterestResult, error) {
	result := &AccrueEscrowInterestResult{CompletedAt: timeutils.NowUnix()}

	controls, err := a.settlementControl.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Unix(timeutils.NowUnix(), 0).UTC()

	for _, control := range controls {
		tenantInfo := pagination.TenantInfo{
			OrgID: control.OrganizationID,
			BuID:  control.BusinessUnitID,
		}
		cutoff := now.AddDate(0, -control.EscrowInterestFrequencyMonths, 0).Unix()
		accounts, listErr := a.escrowRepo.ListDueForInterest(
			ctx,
			repositories.ListEscrowAccountsForInterestRequest{
				TenantInfo:       tenantInfo,
				AccrueOnOrBefore: cutoff,
			},
		)
		if listErr != nil {
			result.Failed++
			a.logger.Error("failed to list escrow accounts due for interest",
				zap.Error(listErr),
				zap.String("orgId", control.OrganizationID.String()))
			continue
		}

		for _, account := range accounts {
			if _, accrueErr := a.payService.AccrueEscrowInterest(
				ctx,
				tenantInfo,
				account.ID,
			); accrueErr != nil {
				result.Failed++
				a.logger.Error("failed to accrue escrow interest",
					zap.Error(accrueErr),
					zap.String("accountId", account.ID.String()))
				continue
			}
			result.AccountsAccrued++
		}
	}
	return result, nil
}

func isDuplicatePeriodError(err error) bool {
	var validationErr *errortypes.Error
	return errors.As(err, &validationErr) && validationErr.Code == errortypes.ErrDuplicate
}
