package planservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Admission struct {
	Plans    platformplanports.PlanService
	Features platformcatalogports.FeatureGate
	Standing managedfeaturesports.StandingReader
}

func (a *Admission) Admit(ctx context.Context, req *AdmissionRequest) error {
	if err := RefuseSuspended(ctx, a.Standing, req.TenantInfo); err != nil {
		return err
	}
	return Admit(ctx, a.Plans, a.Features, req)
}

func RefuseSuspended(
	ctx context.Context,
	standing managedfeaturesports.StandingReader,
	tenantInfo pagination.TenantInfo,
) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return nil
	}
	held, err := standing.Standing(ctx, dbscope.Tenant{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	})
	if err != nil {
		return err
	}
	if held.Suspended {
		return errortypes.NewSuspendedError(held.ManageURL)
	}
	return nil
}
