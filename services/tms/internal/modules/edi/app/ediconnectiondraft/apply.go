package ediconnectiondraft

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	writeCreated = "Created"
	writeUpdated = "Updated"
)

func (s *Service) ApplyConnectionDraft(
	ctx context.Context,
	req *ediports.ApplyConnectionDraftRequest,
) (*ediports.AppliedConnectionDraft, error) {
	if req == nil || req.Draft == nil {
		return nil, errortypes.NewValidationError("", errortypes.ErrRequired, "A draft is required")
	}
	if req.Actor == nil || req.Actor.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"A person must approve the connection before it is written",
		)
	}
	if req.Actor.OrganizationID != req.Draft.TenantInfo.OrgID ||
		req.Actor.BusinessUnitID != req.Draft.TenantInfo.BuID {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrInvalidOperation,
			"The approver belongs to another organization",
		)
	}

	applied := &ediports.AppliedConnectionDraft{}
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		p, err := s.plan(ctx, req.Draft)
		if err != nil {
			return err
		}
		if p.draft.DocumentDigest != req.ExpectedDocumentDigest {
			return errortypes.NewValidationError(
				"documentDigest",
				errortypes.ErrInvalidOperation,
				"The spec document was read differently since this draft was approved; "+
					"draft it again",
			)
		}
		if !p.draft.Ready {
			return errortypes.NewValidationError(
				"draft",
				errortypes.ErrInvalidOperation,
				"The draft is not ready: "+strings.Join(p.draft.Blockers, "; "),
			)
		}

		applied.Draft = p.draft
		return s.write(ctx, p, req, applied)
	})
	if err != nil {
		return nil, err
	}
	return applied, nil
}

func (s *Service) write(
	ctx context.Context,
	p *planner,
	req *ediports.ApplyConnectionDraftRequest,
	applied *ediports.AppliedConnectionDraft,
) error {
	writes := p.writes
	actor := req.Actor
	tenant := req.Draft.TenantInfo

	partner := writes.partner
	switch p.draft.Partner.Change {
	case ediports.ChangeKindAdd:
		created, err := s.setup.CreatePartner(ctx, partner, actor)
		if err != nil {
			return err
		}
		partner = created
		applied.Created = true
		applied.Writes = append(applied.Writes, record(draftPartnerLabel, created.ID, writeCreated, created.Name))
	case ediports.ChangeKindChange:
		updated, err := s.setup.UpdatePartner(ctx, partner, actor)
		if err != nil {
			return err
		}
		partner = updated
		applied.Writes = append(applied.Writes, record(draftPartnerLabel, updated.ID, writeUpdated, updated.Name))
	}
	applied.PartnerID = partner.ID

	if transport := writes.transport; transport != nil && writes.transportChange.Writes() {
		transport.EDIPartnerID = partner.ID
		if transport.ProfileID.IsNil() {
			transport.Name = transportName(partner.Name, transport.Method)
			created, err := s.setup.CreateCommunicationProfile(ctx, transport, actor)
			if err != nil {
				return err
			}
			applied.Writes = append(applied.Writes, record(transportLabel, created.ID, writeCreated, created.Name))
		} else {
			updated, err := s.setup.UpdateCommunicationProfile(ctx, transport, actor)
			if err != nil {
				return err
			}
			applied.Writes = append(applied.Writes, record(transportLabel, updated.ID, writeUpdated, updated.Name))
		}
	}

	documentTypes := make(map[string]pulid.ID, len(writes.profiles))
	for _, profile := range writes.profiles {
		documentTypes[profileKey(profile.set, profile.direction)] = p.docTypes[profileKey(profile.set, profile.direction)]
		if !profile.change.Writes() {
			continue
		}
		request := profile.request
		request.EDIPartnerID = partner.ID
		if request.ProfileID.IsNil() {
			request.Name = profileName(partner.Name, profile.set, profile.direction)
		}
		if profile.starter {
			template, version, err := s.templates.EnsureStarterTemplate(
				ctx,
				tenant,
				profile.set,
				profile.direction,
			)
			if err != nil {
				return err
			}
			request.TemplateID = template.ID
			request.TemplateVersionID = version.ID
		}
		saved, err := s.setup.UpsertPartnerDocumentProfile(ctx, request, actor)
		if err != nil {
			return err
		}
		action := writeUpdated
		if request.ProfileID.IsNil() {
			action = writeCreated
		}
		applied.Writes = append(applied.Writes, record("Document profile", saved.ID, action, saved.Name))
	}

	for _, control := range writes.controls {
		documentTypeID := documentTypes[profileKey(control.set, control.direction)]
		sequence, err := s.setup.ResetControlNumber(ctx, &ediports.ResetEDIControlNumberRequest{
			TenantInfo:     tenant,
			PartnerID:      partner.ID,
			DocumentTypeID: documentTypeID,
			Kind:           control.kind,
			NextValue:      control.next,
		}, actor)
		if err != nil {
			return err
		}
		applied.Writes = append(applied.Writes, record(
			"Control number",
			sequence.ID,
			writeUpdated,
			fmt.Sprintf("%s %s %s starts at %d", control.set, strings.ToLower(string(control.direction)), strings.ToLower(string(control.kind)), control.next),
		))
	}

	if len(writes.mappings) > 0 {
		items, err := s.setup.SaveMappingProfile(ctx, &ediports.SaveMappingItemsRequest{
			PartnerID:  partner.ID,
			TenantInfo: tenant,
			ActorID:    actor.UserID,
			Items:      writes.mappings,
		})
		if err != nil {
			return err
		}
		for _, item := range items {
			applied.Writes = append(applied.Writes, record(
				mappingRecordLabel,
				item.ID,
				writeUpdated,
				fmt.Sprintf("%s %s → %s", item.EntityType, item.SourceID, firstNonEmpty(item.TargetLabel, item.TargetID.String())),
			))
		}
	}
	if applied.Writes == nil {
		applied.Writes = []ediports.DraftWrite{}
	}
	return nil
}

func record(kind string, id pulid.ID, action, label string) ediports.DraftWrite {
	return ediports.DraftWrite{Record: kind, ID: id, Action: action, Label: label}
}
