package shipmentimportassistantservice

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type toolGrant struct {
	resource  permission.Resource
	operation permission.Operation
}

var toolGrants = map[string]toolGrant{
	"search_customers": {
		resource:  permission.ResourceCustomer,
		operation: permission.OpRead,
	},
	"get_customer_requirements": {
		resource:  permission.ResourceCustomer,
		operation: permission.OpRead,
	},
	"search_locations": {
		resource:  permission.ResourceLocation,
		operation: permission.OpRead,
	},
	"add_location": {
		resource:  permission.ResourceLocation,
		operation: permission.OpCreate,
	},
}

func actingUser(tenantInfo pagination.TenantInfo) *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		UserID:         tenantInfo.UserID,
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
	}
}

func (s *Service) authorizeTool(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	name string,
) (string, bool) {
	grant, gated := toolGrants[name]
	if !gated {
		return "", true
	}

	if s.permissions == nil || tenantInfo.UserID.IsNil() {
		s.logger.Error("import assistant tool has no one to authorize it",
			zap.String("tool", name),
			zap.Bool("hasPermissionEngine", s.permissions != nil),
		)

		return toolRefusal(fmt.Sprintf(
			"Tool %q could not be authorized, so it was not run. "+
				"Tell the person it did not happen.",
			name,
		)), false
	}

	result, err := s.permissions.Check(
		ctx,
		actingUser(tenantInfo).PermissionCheck(grant.resource, grant.operation),
	)
	if err != nil {
		s.logger.Error("import assistant tool authorization failed",
			zap.String("tool", name),
			zap.String("userId", tenantInfo.UserID.String()),
			zap.Error(err),
		)

		return toolRefusal(fmt.Sprintf(
			"Tool %q could not be authorized just now, so it was not run. "+
				"Tell the person it did not happen and that they can try again.",
			name,
		)), false
	}

	if !result.Allowed {
		s.logger.Info("import assistant tool call denied",
			zap.String("tool", name),
			zap.String("userId", tenantInfo.UserID.String()),
			zap.String("resource", grant.resource.String()),
			zap.String("operation", string(grant.operation)),
			zap.String("reason", result.Reason),
		)

		return toolRefusal(fmt.Sprintf(
			"Tool %q is not permitted: the person you are working for does not have "+
				"%s access to %s, so it was not run. Tell them it did not happen and that "+
				"an administrator can grant that access. Do not call it again this turn.",
			name,
			grant.operation,
			grant.resource.String(),
		)), false
	}

	return "", true
}

func toolRefusal(message string) string {
	encoded, err := sonic.MarshalString(map[string]string{"error": message})
	if err != nil {
		return `{"error":"the tool was not run"}`
	}

	return encoded
}
