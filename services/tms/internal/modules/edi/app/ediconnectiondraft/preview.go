package ediconnectiondraft

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

func (s *Service) PreviewConnectionDraft(
	ctx context.Context,
	req *ediconnectiondraftports.ConnectionDraftRequest,
) (*ediconnectiondraftports.ConnectionDraftPlan, error) {
	p, err := s.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	plan := &ediconnectiondraftports.ConnectionDraftPlan{
		Draft: p.draft,
		Partner: ediconnectiondraftports.PlannedPartner{
			After:  p.writes.partner,
			Change: p.draft.Partner.Change,
		},
		Mappings: make([]ediconnectiondraftports.PlannedMapping, 0, len(p.writes.mappings)),
		ControlNumbers: make(
			[]ediconnectiondraftports.PlannedControlNumber,
			0,
			len(p.writes.controls),
		),
	}
	if p.match.setup != nil {
		plan.Partner.Before = p.match.setup.Partner
	}
	if transport := p.plannedTransport(); transport != nil {
		plan.Transport = transport
	}
	if project := p.writes.project; project != nil {
		plan.Project = &ediconnectiondraftports.PlannedProject{
			Files:   project.files,
			Samples: project.samples,
			Change:  project.change,
		}
		if p.match.setup != nil {
			plan.Project.Project = p.match.setup.Project
		}
		plan.Project.Before = p.current.version
	}
	for _, item := range p.writes.mappings {
		before := p.current.mapping(item.EntityType, item.SourceID)
		change := ediconnectiondraftports.ChangeKindAdd
		if before != nil {
			change = ediconnectiondraftports.ChangeKindChange
		}
		plan.Mappings = append(plan.Mappings, ediconnectiondraftports.PlannedMapping{
			Before: before,
			After:  item,
			Change: change,
		})
	}
	for _, control := range p.writes.controls {
		documentTypeID := p.docTypes[setKey(control.set, control.direction)]
		before := p.current.sequence(documentTypeID, control.kind)
		change := ediconnectiondraftports.ChangeKindAdd
		if before != nil {
			change = ediconnectiondraftports.ChangeKindChange
		}
		plan.ControlNumbers = append(
			plan.ControlNumbers,
			ediconnectiondraftports.PlannedControlNumber{
				Before: before,
				After: &edi.EDIControlNumberSequence{
					BusinessUnitID: req.TenantInfo.BuID,
					OrganizationID: req.TenantInfo.OrgID,
					EDIPartnerID:   p.match.partnerID(),
					DocumentTypeID: documentTypeID,
					Kind:           control.kind,
					NextValue:      control.next,
				},
				Change: change,
			},
		)
	}
	return plan, nil
}

func (p *planner) plannedTransport() *ediconnectiondraftports.PlannedTransport {
	request := p.writes.transport
	if request == nil {
		return nil
	}
	planned := &ediconnectiondraftports.PlannedTransport{
		After: &edi.EDICommunicationProfile{
			ID:             request.ProfileID,
			BusinessUnitID: p.req.TenantInfo.BuID,
			OrganizationID: p.req.TenantInfo.OrgID,
			EDIPartnerID:   request.EDIPartnerID,
			Method:         request.Method,
			Status:         domaintypes.Status(request.Status),
			Name:           request.Name,
			Description:    request.Description,
			Config:         request.Config,
			Version:        request.Version,
		},
		Change: p.writes.transportChange,
	}
	if request.ProfileID.IsNotNil() {
		planned.Before = p.current.transport
	}
	return planned
}
