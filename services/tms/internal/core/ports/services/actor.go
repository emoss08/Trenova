package services

import "github.com/emoss08/trenova/shared/pulid"

const (
	SystemPrincipalID = pulid.ID("system")
	AgentPrincipalID  = pulid.ID("agent")
)

type AuditActor struct {
	PrincipalType PrincipalType
	PrincipalID   pulid.ID
	UserID        pulid.ID
	APIKeyID      pulid.ID
}

func SystemAuditActor() AuditActor {
	return AuditActor{
		PrincipalType: PrincipalTypeSystem,
		PrincipalID:   SystemPrincipalID,
	}
}

func AgentAuditActor() AuditActor {
	return AuditActor{
		PrincipalType: PrincipalTypeAgent,
		PrincipalID:   AgentPrincipalID,
	}
}

func (a *RequestActor) UserIDOrNil() pulid.ID {
	if a == nil {
		return pulid.Nil
	}
	return a.UserID
}

func (a *RequestActor) AuditActorOrSystem() AuditActor {
	auditActor := a.AuditActor()
	if auditActor.PrincipalType == "" {
		return SystemAuditActor()
	}
	return auditActor
}

func (a *RequestActor) AuditActor() AuditActor {
	if a == nil {
		return AuditActor{}
	}

	auditActor := AuditActor{
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
	case PrincipalTypeSystem, PrincipalTypeAgent:
		auditActor.UserID = pulid.Nil
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
		auditActor.UserID = pulid.Nil
		auditActor.APIKeyID = pulid.Nil
		if auditActor.PrincipalID.IsNil() {
			auditActor.PrincipalID = AgentPrincipalID
		}
	}

	return auditActor
}
