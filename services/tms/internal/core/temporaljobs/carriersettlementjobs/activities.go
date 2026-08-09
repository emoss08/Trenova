package carriersettlementjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	SettlementControl repositories.CarrierSettlementControlRepository
	SettlementService *carriersettlementservice.Service
	Logger            *zap.Logger
}

type Activities struct {
	settlementControl repositories.CarrierSettlementControlRepository
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

func systemActor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType: services.PrincipalTypeSystem,
		PrincipalID:   services.SystemPrincipalID,
		UserID:        services.SystemPrincipalID,
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
		bounds := carriersettlementservice.ResolveCurrentPeriod(control, now)

		batch, genErr := a.settlementService.GenerateBatch(
			ctx,
			&carriersettlementservice.GenerateBatchRequest{
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

func isDuplicatePeriodError(err error) bool {
	var validationErr *errortypes.Error
	return errors.As(err, &validationErr) && validationErr.Code == errortypes.ErrDuplicate
}
