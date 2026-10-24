package organizationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/zap"
)

var capabilityLocks = map[tenantports.CapabilityLock]keel.LockStrength{
	tenantports.CapabilityLockShare:  keel.ForShare,
	tenantports.CapabilityLockUpdate: keel.ForUpdate,
}

func (r *repository) GetCapabilities(
	ctx context.Context,
	req tenantports.GetOrganizationCapabilitiesRequest,
) (*tenantports.OrganizationCapabilities, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*tenantports.OrganizationCapabilities, error) {
			o := tenantdb.Organizations
			q := o.Pick(o.BrokerageEnabled, o.AssetOperationsEnabled).Where(
				o.ID.Eq(req.TenantInfo.OrgID),
				o.BusinessUnitID.Eq(req.TenantInfo.BuID),
			)
			if strength, ok := capabilityLocks[req.Lock]; ok {
				q = q.Lock(strength)
			}

			org, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to read organization capabilities",
					zap.String("orgID", req.TenantInfo.OrgID.String()),
					zap.String("lock", string(req.Lock)),
					zap.Error(err),
				)
				return nil, organizationErrors.Map(err, req.TenantInfo.OrgID.String())
			}

			return &tenantports.OrganizationCapabilities{
				BrokerageEnabled:       org.BrokerageEnabled,
				AssetOperationsEnabled: org.AssetOperationsEnabled,
			}, nil
		},
	)
}
