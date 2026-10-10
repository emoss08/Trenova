package permissionports

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const PrincipalTypeUser kernelports.PrincipalType = "session_user"

const PrincipalTypeAPIKey kernelports.PrincipalType = "api_key"

const PrincipalTypeSystem kernelports.PrincipalType = "system"

const PrincipalTypeAgent kernelports.PrincipalType = "agent"

type RequestActor struct {
	PrincipalType  kernelports.PrincipalType `json:"principalType"`
	PrincipalID    pulid.ID                  `json:"principalId"`
	UserID         pulid.ID                  `json:"userId,omitempty"`
	APIKeyID       pulid.ID                  `json:"apiKeyId,omitempty"`
	BusinessUnitID pulid.ID                  `json:"businessUnitId"`
	OrganizationID pulid.ID                  `json:"organizationId"`
}

func (a *RequestActor) IsAgent() bool {
	if a == nil {
		return false
	}
	return a.PrincipalType == PrincipalTypeAgent
}

func (a *RequestActor) AuditActorOrSystem() kernelports.AuditActor {
	auditActor := a.AuditActor()
	if auditActor.PrincipalType == "" {
		return SystemAuditActor()
	}
	return auditActor
}

// TenantInfo is the actor's tenant scope. Deriving it here rather than at each
// call site keeps a caller from scoping a query to a tenant the actor does not
// belong to.
func (a *RequestActor) TenantInfo() pagination.TenantInfo {
	if a == nil {
		return pagination.TenantInfo{}
	}

	return pagination.TenantInfo{
		OrgID:  a.OrganizationID,
		BuID:   a.BusinessUnitID,
		UserID: a.UserID,
	}
}

func (a *RequestActor) ExecutorUserID() pulid.ID {
	if a == nil {
		return pulid.Nil
	}

	switch a.PrincipalType {
	case PrincipalTypeUser, PrincipalTypeAgent:
		return a.UserID
	case "", PrincipalTypeAPIKey, PrincipalTypeSystem:
	}

	return pulid.Nil
}

func (a *RequestActor) IsUser() bool {
	if a == nil {
		return false
	}
	return a.PrincipalType == PrincipalTypeUser
}

func (a *RequestActor) UserIDOrNil() pulid.ID {
	if a == nil {
		return pulid.Nil
	}
	return a.UserID
}

func (a *RequestActor) AuditActor() kernelports.AuditActor {
	if a == nil {
		return kernelports.AuditActor{}
	}

	auditActor := kernelports.AuditActor{
		PrincipalType: a.PrincipalType,
		PrincipalID:   a.PrincipalID,
		UserID:        a.UserID,
		APIKeyID:      a.APIKeyID,
	}

	switch auditActor.PrincipalType {
	case "":
		switch {
		case auditActor.APIKeyID.IsNotNil():
			auditActor.PrincipalType = PrincipalTypeAPIKey
		case auditActor.UserID.IsNotNil():
			auditActor.PrincipalType = PrincipalTypeUser
		}
	case PrincipalTypeUser:
		auditActor.APIKeyID = pulid.Nil
	case PrincipalTypeAPIKey:
		auditActor.UserID = pulid.Nil
	case PrincipalTypeSystem:
		auditActor.UserID = pulid.Nil
		auditActor.APIKeyID = pulid.Nil
	case PrincipalTypeAgent:
		auditActor.APIKeyID = pulid.Nil
	}

	if auditActor.PrincipalID.IsNil() {
		switch auditActor.PrincipalType {
		case PrincipalTypeAPIKey:
			auditActor.PrincipalID = auditActor.APIKeyID
		case PrincipalTypeUser:
			auditActor.PrincipalID = auditActor.UserID
		case PrincipalTypeSystem:
			auditActor.PrincipalID = SystemPrincipalID
		case PrincipalTypeAgent:
			auditActor.PrincipalID = AgentPrincipalID
		}
	}

	if auditActor.PrincipalType == PrincipalTypeAPIKey && auditActor.APIKeyID.IsNil() {
		auditActor.APIKeyID = auditActor.PrincipalID
	}

	if auditActor.PrincipalType == PrincipalTypeUser {
		auditActor.APIKeyID = pulid.Nil
		if auditActor.UserID.IsNil() {
			auditActor.UserID = auditActor.PrincipalID
		}
	}

	if auditActor.PrincipalType == PrincipalTypeSystem {
		auditActor.UserID = pulid.Nil
		auditActor.APIKeyID = pulid.Nil
		if auditActor.PrincipalID.IsNil() {
			auditActor.PrincipalID = SystemPrincipalID
		}
	}

	if auditActor.PrincipalType == PrincipalTypeAgent {
		auditActor.APIKeyID = pulid.Nil
		if auditActor.PrincipalID.IsNil() {
			auditActor.PrincipalID = AgentPrincipalID
		}
		if auditActor.UserID == auditActor.PrincipalID {
			auditActor.UserID = pulid.Nil
		}
	}

	return auditActor
}

// PermissionCheck is the check of the actor holding the operation on the
// resource.
func (a *RequestActor) PermissionCheck(
	resource permission.Resource,
	operation permission.Operation,
) *PermissionCheckRequest {
	if a == nil {
		return &PermissionCheckRequest{Resource: resource.String(), Operation: operation}
	}

	return &PermissionCheckRequest{
		PrincipalType:  a.PrincipalType,
		PrincipalID:    a.PrincipalID,
		UserID:         a.UserID,
		APIKeyID:       a.APIKeyID,
		BusinessUnitID: a.BusinessUnitID,
		OrganizationID: a.OrganizationID,
		Resource:       resource.String(),
		Operation:      operation,
	}
}

func (a *RequestActor) DBTenant() dbscope.Tenant {
	tenant := a.TenantInfo().DBTenant()
	if a.IsAgent() {
		tenant.UserID = pulid.Nil
	}

	return tenant
}

func (a *RequestActor) IsAPIKey() bool {
	if a == nil {
		return false
	}
	return a.PrincipalType == PrincipalTypeAPIKey
}

func (a *RequestActor) PersonUserID() pulid.ID {
	if a == nil {
		return pulid.Nil
	}

	switch a.PrincipalType {
	case PrincipalTypeUser:
		return a.UserID
	case "":
		if a.APIKeyID.IsNil() {
			return a.UserID
		}
	case PrincipalTypeAPIKey, PrincipalTypeSystem, PrincipalTypeAgent:
	}

	return pulid.Nil
}
