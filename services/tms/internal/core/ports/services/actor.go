package services

import "github.com/emoss08/trenova/shared/pulid"

type AuditActor struct {
	PrincipalType PrincipalType
	PrincipalID   pulid.ID
	UserID        pulid.ID
	APIKeyID      pulid.ID
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
	}

	if auditActor.PrincipalID.IsNil() {
		switch auditActor.PrincipalType {
		case PrincipalTypeAPIKey:
			auditActor.PrincipalID = auditActor.APIKeyID
		case PrincipalTypeUser:
			auditActor.PrincipalID = auditActor.UserID
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

	return auditActor
}
