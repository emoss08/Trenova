package settlementjobs

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	SettlementControl tenantports.SettlementControlRepository
	EscrowRepo        driverpayports.EscrowAccountRepository
	SettlementService *driversettlementservice.Service
	PayService        *driverpayservice.Service
	Logger            *zap.Logger
}

type Activities struct {
	settlementControl tenantports.SettlementControlRepository
	escrowRepo        driverpayports.EscrowAccountRepository
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

func systemActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeSystem,
		PrincipalID:   permissionports.SystemPrincipalID,
		UserID:        permissionports.SystemPrincipalID,
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
		tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
		bounds := driversettlementservice.ResolveCurrentPeriod(control, now)

		batch, genErr := a.settlementService.GenerateBatch(
			tenantCtx,
			&driversettlementservice.GenerateBatchRequest{
				TenantInfo:  tenantInfo,
				PeriodStart: bounds.PeriodStart,
				PeriodEnd:   bounds.PeriodEnd,
			},
			systemActor(),
		)
		if genErr != nil {
			if isDuplicatePeriodError(genErr) ||
				errors.Is(genErr, settlementshared.ErrPeriodBatchClosed) {
				result.Skipped++
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
		tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
		cutoff := now.AddDate(0, -control.EscrowInterestFrequencyMonths, 0).Unix()
		accounts, listErr := a.escrowRepo.ListDueForInterest(
			tenantCtx,
			driverpayports.ListEscrowAccountsForInterestRequest{
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
				tenantCtx,
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
