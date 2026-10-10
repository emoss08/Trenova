package permissionports

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const SystemPrincipalID = pulid.ID("system")

const AgentPrincipalID = pulid.ID("agent")

func SystemAuditActor() kernelports.AuditActor {
	return kernelports.AuditActor{
		PrincipalType: PrincipalTypeSystem,
		PrincipalID:   SystemPrincipalID,
	}
}

func AgentAuditActor() kernelports.AuditActor {
	return kernelports.AuditActor{
		PrincipalType: PrincipalTypeAgent,
		PrincipalID:   AgentPrincipalID,
	}
}

// UserActor is the person a tenant scope was read for, as an actor: the
// user named by its UserID, in its organization and business unit. It is
// for code that holds only the scope, such as a request's loaders or a
// conversation read under its owner's id.
func UserActor(tenant pagination.TenantInfo) *RequestActor {
	return &RequestActor{
		PrincipalType:  PrincipalTypeUser,
		PrincipalID:    tenant.UserID,
		UserID:         tenant.UserID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
}
