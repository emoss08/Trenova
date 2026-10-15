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
		DocumentProfiles: make([]ediports.PlannedDocumentProfile, 0, len(p.writes.profiles)),
		Mappings:         make([]ediports.PlannedMapping, 0, len(p.writes.mappings)),
		ControlNumbers:   make([]ediports.PlannedControlNumber, 0, len(p.writes.controls)),
	}
	if p.match.setup != nil {
		plan.Partner.Before = p.match.setup.Partner
	}
	if transport := p.plannedTransport(); transport != nil {
		plan.Transport = transport
	}
	for _, profile := range p.writes.profiles {
		plan.DocumentProfiles = append(plan.DocumentProfiles, ediports.PlannedDocumentProfile{
			Before: p.current.profiles[profileKey(profile.set, profile.direction)],
			After:  plannedProfile(p, profile),
			Change: profile.change,
		})
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
		documentTypeID := p.docTypes[profileKey(control.set, control.direction)]
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

func plannedProfile(p *planner, profile *profileWrite) *edi.EDIPartnerDocumentProfile {
	request := profile.request
	return &edi.EDIPartnerDocumentProfile{
		ID:                           request.ProfileID,
		BusinessUnitID:               p.req.TenantInfo.BuID,
		OrganizationID:               p.req.TenantInfo.OrgID,
		EDIPartnerID:                 request.EDIPartnerID,
		DocumentTypeID:               p.docTypes[profileKey(profile.set, profile.direction)],
		TemplateID:                   request.TemplateID,
		TemplateVersionID:            request.TemplateVersionID,
		Name:                         request.Name,
		Status:                       request.Status,
		Direction:                    profile.direction,
		Standard:                     edi.EDIStandardX12,
		TransactionSet:               profile.set,
		X12VersionOverride:           request.X12VersionOverride,
		FunctionalGroupID:            request.FunctionalGroupID,
		Envelope:                     request.Envelope,
		Acknowledgment:               request.Acknowledgment,
		ValidationMode:               request.ValidationMode,
		PartnerSettings:              request.PartnerSettings,
		PartnerSettingsSchemaID:      request.PartnerSettingsSchemaID,
		PartnerSettingsSchemaVersion: request.PartnerSettingsSchemaVersion,
		Version:                      request.Version,
	}
}
