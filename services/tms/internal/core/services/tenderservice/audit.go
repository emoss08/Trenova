package tenderservice

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type tenderAuditParams struct {
	TenantInfo pagination.TenantInfo
	TenderID   pulid.ID
	Operation  permission.Operation
	UserID     pulid.ID
	Comment    string
	Current    *tender.Tender
	Previous   *tender.Tender
}

func (s *Service) logTenderAudit(p *tenderAuditParams) {
	if s.auditService == nil {
		return
	}

	params := &portservices.LogActionParams{
		Resource:       permission.ResourceTender,
		ResourceID:     p.TenderID.String(),
		Operation:      p.Operation,
		UserID:         p.UserID,
		OrganizationID: p.TenantInfo.OrgID,
		BusinessUnitID: p.TenantInfo.BuID,
	}
	if p.Current != nil {
		params.CurrentState = jsonutils.MustToJSON(p.Current)
	}
	if p.UserID.IsNil() {
		params.PrincipalType = portservices.PrincipalTypeSystem
		params.PrincipalID = portservices.SystemPrincipalID
	}

	options := []portservices.LogOption{auditservice.WithComment(p.Comment)}
	if p.Previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.Previous)
		if p.Current != nil {
			options = append(options, auditservice.WithDiff(p.Previous, p.Current))
		}
	}

	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log tender audit action", zap.Error(err))
	}
}
