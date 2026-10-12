package rateconfirmationservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type rateConfirmationAuditParams struct {
	TenantInfo pagination.TenantInfo
	Operation  permission.Operation
	UserID     pulid.ID
	Comment    string
	Current    *rateconfirmation.RateConfirmation
	Previous   *rateconfirmation.RateConfirmation
}

func (s *Service) logRateConfirmationAudit(p *rateConfirmationAuditParams) {
	if s.auditService == nil || p.Current == nil {
		return
	}

	params := &auditports.LogActionParams{
		Resource:       permission.ResourceRateConfirmation,
		ResourceID:     p.Current.ID.String(),
		Operation:      p.Operation,
		UserID:         p.UserID,
		CurrentState:   jsonutils.MustToJSON(p.Current),
		OrganizationID: p.TenantInfo.OrgID,
		BusinessUnitID: p.TenantInfo.BuID,
	}
	if p.UserID.IsNil() {
		params.PrincipalType = permissionports.PrincipalTypeSystem
		params.PrincipalID = permissionports.SystemPrincipalID
	}

	options := []auditports.LogOption{auditservice.WithComment(p.Comment)}
	if p.Previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.Previous)
		options = append(options, auditservice.WithDiff(p.Previous, p.Current))
	}

	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log rate confirmation audit action", zap.Error(err))
	}
}
