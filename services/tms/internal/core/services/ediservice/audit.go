package ediservice

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) logAction(
	entity interface {
		GetID() pulid.ID
		GetOrganizationID() pulid.ID
		GetBusinessUnitID() pulid.ID
	},
	actor *services.RequestActor,
	operation permission.Operation,
	previous any,
	current any,
	comment string,
) {
	if s.auditService == nil || actor == nil || entity == nil {
		return
	}

	auditActor := actor.AuditActor()
	params := &services.LogActionParams{
		Resource:       permission.ResourceEDI,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		CurrentState:   jsonutils.MustToJSON(current),
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	if err := s.auditService.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Warn("failed to log EDI audit action", zap.Error(err))
	}
}
