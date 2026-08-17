package organizationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/zap"
)

func (r *repository) GetCapabilities(
	ctx context.Context,
	req repositories.GetOrganizationCapabilitiesRequest,
) (*repositories.OrganizationCapabilities, error) {
	log := r.l.With(
		zap.String("operation", "GetCapabilities"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
		zap.String("lock", string(req.Lock)),
	)

	org := buncolgen.OrganizationColumns
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.Organization)(nil)).
		Column(
			org.BrokerageEnabled.String(),
			org.AssetOperationsEnabled.String(),
		).
		Where(org.ID.Eq(), req.TenantInfo.OrgID).
		Where(org.BusinessUnitID.Eq(), req.TenantInfo.BuID)

	if req.Lock != repositories.CapabilityLockNone {
		query = query.For(string(req.Lock))
	}

	capabilities := new(repositories.OrganizationCapabilities)
	if err := query.Scan(
		ctx,
		&capabilities.BrokerageEnabled,
		&capabilities.AssetOperationsEnabled,
	); err != nil {
		log.Error("failed to read organization capabilities", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Organization")
	}

	return capabilities, nil
}
