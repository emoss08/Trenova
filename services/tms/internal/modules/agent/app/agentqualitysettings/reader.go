package agentqualitysettings

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Controls      agentqualityports.AgentQualityControlRepository
	Organizations tenantports.OrganizationRepository
}

type Reader struct {
	l             *zap.Logger
	controls      agentqualityports.AgentQualityControlRepository
	organizations tenantports.OrganizationRepository
}

func New(p Params) *Reader {
	return &Reader{
		l:             p.Logger.Named("service.agentquality"),
		controls:      p.Controls,
		organizations: p.Organizations,
	}
}

func (r *Reader) Control(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*agentqualitycontrol.Control, error) {
	control, err := r.controls.Get(ctx, tenant)
	if err == nil {
		return control, nil
	}
	if !dberror.IsNotFoundError(err) && !errortypes.IsNotFoundError(err) {
		return nil, fmt.Errorf("read quality controls: %w", err)
	}

	return agentqualitycontrol.DefaultControl(tenant.OrgID, tenant.BuID), nil
}

func (r *Reader) Timezone(
	ctx context.Context,
	control *agentqualitycontrol.Control,
	tenant pagination.TenantInfo,
) string {
	organizationTimezone := ""
	if r.organizations != nil {
		organization, err := r.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
			TenantInfo: tenant,
		})
		if err != nil {
			r.l.Warn("could not read the organization's timezone; reading windows in UTC",
				zap.String("organization", tenant.OrgID.String()),
				zap.Error(err),
			)
		} else {
			organizationTimezone = organization.Timezone
		}
	}

	return control.EffectiveTimezone(organizationTimezone)
}

func Windows(now int64, timezone string) (dayStart, monthStart int64, err error) {
	if dayStart, err = timeutils.DayStartUnix(now, timezone); err != nil {
		return 0, 0, fmt.Errorf("start of day: %w", err)
	}
	if monthStart, err = timeutils.MonthStartUnix(now, timezone); err != nil {
		return 0, 0, fmt.Errorf("start of month: %w", err)
	}

	return dayStart, monthStart, nil
}
