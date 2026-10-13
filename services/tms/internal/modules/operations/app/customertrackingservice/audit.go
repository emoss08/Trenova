package customertrackingservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type auditParams struct {
	tenant     pagination.TenantInfo
	userID     pulid.ID
	resource   permission.Resource
	resourceID string
	operation  permission.Operation
	comment    string
	previous   map[string]any
	current    map[string]any
}

func (s *Service) audit(p *auditParams) {
	if s.auditService == nil {
		return
	}
	params := &auditports.LogActionParams{
		Resource:       p.resource,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   p.current,
		PreviousState:  p.previous,
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
	}
	if p.userID.IsNil() {
		params.PrincipalType = permissionports.PrincipalTypeSystem
		params.PrincipalID = permissionports.SystemPrincipalID
	}
	comment := p.comment
	if err := s.auditService.LogAction(params, func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}); err != nil {
		s.l.Error("failed to log customer tracking audit", zap.Error(err))
	}
}
