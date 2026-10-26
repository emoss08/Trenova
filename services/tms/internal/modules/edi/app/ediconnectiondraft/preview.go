package ediconnectiondraft

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

func (s *Service) PreviewConnectionDraft(
	ctx context.Context,
	req *ediports.ConnectionDraftRequest,
) (*ediports.ConnectionDraftPlan, error) {
	p, err := s.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	plan := &ediports.ConnectionDraftPlan{
		Draft: p.draft,
		Partner: ediports.PlannedPartner{
			After:  p.writes.partner,
			Change: p.draft.Partner.Change,
		},
		Mappings:       make([]ediports.PlannedMapping, 0, len(p.writes.mappings)),
		ControlNumbers: make([]ediports.PlannedControlNumber, 0, len(p.writes.controls)),
	}
	if p.match.setup != nil {
		plan.Partner.Before = p.match.setup.Partner
	}
	if transport := p.plannedTransport(); transport != nil {
		plan.Transport = transport
	}
	if project := p.writes.project; project != nil {
		plan.Project = &ediports.PlannedProject{
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
		change := ediports.ChangeKindAdd
		if before != nil {
			change = ediports.ChangeKindChange
		}
		plan.Mappings = append(plan.Mappings, ediports.PlannedMapping{
			Before: before,
			After:  item,
			Change: change,
		})
	}
	for _, control := range p.writes.controls {
		documentTypeID := p.docTypes[setKey(control.set, control.direction)]
		before := p.current.sequence(documentTypeID, control.kind)
		change := ediports.ChangeKindAdd
		if before != nil {
			change = ediports.ChangeKindChange
		}
		plan.ControlNumbers = append(plan.ControlNumbers, ediports.PlannedControlNumber{
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
		})
	}
	return plan, nil
}

func (p *planner) plannedTransport() *ediports.PlannedTransport {
	request := p.writes.transport
	if request == nil {
		return nil
	}
	planned := &ediports.PlannedTransport{
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
