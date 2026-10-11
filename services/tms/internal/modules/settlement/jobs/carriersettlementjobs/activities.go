package carriersettlementjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	SettlementControl tenantports.CarrierSettlementControlRepository
	SettlementService *carriersettlementservice.Service
	Logger            *zap.Logger
}

type Activities struct {
	settlementControl tenantports.CarrierSettlementControlRepository
	settlementService *carriersettlementservice.Service
	logger            *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		settlementControl: p.SettlementControl,
		settlementService: p.SettlementService,
		logger:            p.Logger.Named("carrier-settlement-activities"),
	}
}

func systemActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeSystem,
		PrincipalID:   permissionports.SystemPrincipalID,
		UserID:        permissionports.SystemPrincipalID,
	}
}

func (a *Activities) GenerateCarrierSettlementBatchesActivity(
	ctx context.Context,
) (*GenerateCarrierSettlementBatchesResult, error) {
	result := &GenerateCarrierSettlementBatchesResult{CompletedAt: timeutils.NowUnix()}

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
		bounds := carriersettlementservice.ResolveCurrentPeriod(control, now)

		batch, genErr := a.settlementService.GenerateBatch(
			tenantCtx,
			&carriersettlementservice.GenerateBatchRequest{
				TenantInfo:  tenantInfo,
				PeriodStart: bounds.PeriodStart,
				PeriodEnd:   bounds.PeriodEnd,
			},
			systemActor(),
		)
		if genErr != nil {
			if errors.Is(genErr, settlementshared.ErrPeriodBatchClosed) {
				result.Skipped++
				continue
			}
			result.Failed++
			a.logger.Error("failed to auto-generate carrier settlement batch",
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
