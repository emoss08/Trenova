package ediconnectiondraft

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	writeCreated = "Created"
	writeUpdated = "Updated"
)

func (s *Service) ApplyConnectionDraft(
	ctx context.Context,
	req *ediconnectiondraftports.ApplyConnectionDraftRequest,
) (*ediconnectiondraftports.AppliedConnectionDraft, error) {
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

	applied := &ediconnectiondraftports.AppliedConnectionDraft{}
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
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

type writer struct {
	s       *Service
	p       *planner
	actor   *permissionports.RequestActor
	tenant  pagination.TenantInfo
	applied *ediconnectiondraftports.AppliedConnectionDraft
	partner *edi.EDIPartner
}

func (s *Service) write(
	ctx context.Context,
	p *planner,
	req *ediconnectiondraftports.ApplyConnectionDraftRequest,
	applied *ediconnectiondraftports.AppliedConnectionDraft,
) error {
	w := &writer{
		s:       s,
		p:       p,
		actor:   req.Actor,
		tenant:  req.Draft.TenantInfo,
		applied: applied,
		partner: p.writes.partner,
	}
	steps := []func(context.Context) error{
		w.writePartner,
		w.writeTransport,
		w.writeProject,
		w.writeControlNumbers,
		w.writeMappings,
	}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return err
		}
	}
	if applied.Writes == nil {
		applied.Writes = []ediconnectiondraftports.DraftWrite{}
	}
	return nil
}

func (w *writer) note(kind string, id pulid.ID, action, label string) {
	w.applied.Writes = append(w.applied.Writes, record(kind, id, action, label))
}

func (w *writer) writePartner(ctx context.Context) error {
	switch w.p.draft.Partner.Change {
	case ediconnectiondraftports.ChangeKindAdd:
		created, err := w.s.setup.CreatePartner(ctx, w.partner, w.actor)
		if err != nil {
			return err
		}
		w.partner = created
		w.applied.Created = true
		w.note(draftPartnerLabel, created.ID, writeCreated, created.Name)
	case ediconnectiondraftports.ChangeKindChange:
		updated, err := w.s.setup.UpdatePartner(ctx, w.partner, w.actor)
		if err != nil {
			return err
		}
		w.partner = updated
		w.note(draftPartnerLabel, updated.ID, writeUpdated, updated.Name)
	case ediconnectiondraftports.ChangeKindSame, ediconnectiondraftports.ChangeKindNone:
	}
	w.applied.PartnerID = w.partner.ID
	return nil
}

func (w *writer) writeTransport(ctx context.Context) error {
	transport := w.p.writes.transport
	if transport == nil || !w.p.writes.transportChange.Writes() {
		return nil
	}
	transport.EDIPartnerID = w.partner.ID
	if transport.ProfileID.IsNotNil() {
		updated, err := w.s.setup.UpdateCommunicationProfile(ctx, transport, w.actor)
		if err != nil {
			return err
		}
		w.note(transportLabel, updated.ID, writeUpdated, updated.Name)
		return nil
	}
	transport.Name = transportName(w.partner.Name, transport.Method)
	created, err := w.s.setup.CreateCommunicationProfile(ctx, transport, w.actor)
	if err != nil {
		return err
	}
	w.note(transportLabel, created.ID, writeCreated, created.Name)
	return nil
}

func (w *writer) writeProject(ctx context.Context) error {
	project := w.p.writes.project
	if project == nil || !project.change.Writes() {
		return nil
	}
	version, err := w.s.projects.Import(ctx, &edilading.ImportRequest{
		TenantInfo: w.tenant,
		PartnerID:  w.partner.ID,
		Files:      project.files,
		Samples:    project.samples,
		Notes:      project.notes,
	}, w.actor)
	if err != nil {
		return err
	}
	action := writeUpdated
	if project.change == ediconnectiondraftports.ChangeKindAdd {
		action = writeCreated
	}
	w.note(projectRecord, version.ProjectID, action, fmt.Sprintf("Draft %d", version.Number))
	return nil
}

func (w *writer) writeControlNumbers(ctx context.Context) error {
	for _, control := range w.p.writes.controls {
		sequence, err := w.s.setup.StartControlNumber(
			ctx,
			&edipartnerports.ResetEDIControlNumberRequest{
				TenantInfo:     w.tenant,
				PartnerID:      w.partner.ID,
				DocumentTypeID: w.p.docTypes[setKey(control.set, control.direction)],
				Kind:           control.kind,
				NextValue:      control.next,
			},
			w.actor,
		)
		if err != nil {
			return err
		}
		w.note("Control number", sequence.ID, writeUpdated, fmt.Sprintf(
			"%s %s %s starts at %d",
			control.set,
			strings.ToLower(string(control.direction)),
			strings.ToLower(string(control.kind)),
			control.next,
		))
	}
	return nil
}

func (w *writer) writeMappings(ctx context.Context) error {
	if len(w.p.writes.mappings) == 0 {
		return nil
	}
	items, err := w.s.setup.SaveMappingProfile(ctx, &edipartnerports.SaveMappingItemsRequest{
		PartnerID:  w.partner.ID,
		TenantInfo: w.tenant,
		ActorID:    w.actor.UserID,
		Items:      w.p.writes.mappings,
	})
	if err != nil {
		return err
	}
	for _, item := range items {
		w.note(mappingRecordLabel, item.ID, writeUpdated, fmt.Sprintf(
			"%s %s → %s",
			item.EntityType,
			item.SourceID,
			firstNonEmpty(item.TargetLabel, item.TargetID.String()),
		))
	}
	return nil
}

func record(kind string, id pulid.ID, action, label string) ediconnectiondraftports.DraftWrite {
	return ediconnectiondraftports.DraftWrite{Record: kind, ID: id, Action: action, Label: label}
}
