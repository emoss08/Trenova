package rateconfirmationservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const supersededRevisionReason = "Superseded by a new revision"

// GeneratePreview is the revision Generate would create, unrendered and
// unsaved, and the standing revision it would void.
type GeneratePreview struct {
	Created          *rateconfirmation.RateConfirmation
	SupersededBefore *rateconfirmation.RateConfirmation
	SupersededAfter  *rateconfirmation.RateConfirmation
	Assignment       *shipment.CarrierAssignment
	Carrier          *carrier.Carrier
	Shipment         *shipment.Shipment
}

// SendPreview is the email Send would send, rendered by the template Send
// renders it with, and the revision as the send would leave it. The sign
// link is minted only when the email goes, so the preview says whether one
// travels rather than showing it.
type SendPreview struct {
	Before     *rateconfirmation.RateConfirmation
	After      *rateconfirmation.RateConfirmation
	Recipients []string
	Subject    string
	Body       string
	Attachment string
	SignLink   bool
}

// ChangePreview is a revision before and after a void or a confirmation, and
// its carrier assignment when the change moves it too.
type ChangePreview struct {
	Before           *rateconfirmation.RateConfirmation
	After            *rateconfirmation.RateConfirmation
	AssignmentBefore *shipment.CarrierAssignment
	AssignmentAfter  *shipment.CarrierAssignment
}

// generatePlan is everything generate reads before it renders.
type generatePlan struct {
	assignment *shipment.CarrierAssignment
	carrier    *carrier.Carrier
	shipment   *shipment.Shipment
	move       *shipment.ShipmentMove
}

// sendPlan is everything Send decides before it mints a link or sends.
type sendPlan struct {
	entity          *rateconfirmation.RateConfirmation
	templateContext *documenttemplate.RateConfirmationContext
	recipients      []string
}

// PreviewGenerate is what Generate would do, from the same checks, without
// rendering, filing or saving anything.
func (s *Service) PreviewGenerate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	actor *services.RequestActor,
) (*GeneratePreview, error) {
	plan, err := s.planGenerate(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}

	maxRevision, err := s.repo.MaxRevisionForAssignment(ctx, tenantInfo, plan.assignment.ID)
	if err != nil {
		return nil, err
	}

	created := newRevision(tenantInfo, plan, maxRevision+1, actor, nil)
	multiErr := errortypes.NewMultiError()
	created.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	out := &GeneratePreview{
		Created:    created,
		Assignment: plan.assignment,
		Carrier:    plan.carrier,
		Shipment:   plan.shipment,
	}

	active, err := s.repo.GetActiveByAssignmentID(ctx, tenantInfo, plan.assignment.ID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		out.SupersededBefore = active
		out.SupersededAfter, err = copyRateCon(active, func(rc *rateconfirmation.RateConfirmation) {
			rc.Void(timeutils.NowUnix(), supersededRevisionReason)
		})
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

// PreviewSend is what Send would send and record, from the same checks and
// the same message template, without minting a sign link or sending.
func (s *Service) PreviewSend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
) (*SendPreview, error) {
	plan, err := s.planSend(ctx, tenantInfo, rateConfirmationID)
	if err != nil {
		return nil, err
	}

	message, err := s.templates.RenderMessage(ctx, &services.RenderMessageRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindRateConfirmationEmail,
		Data:        plan.templateContext,
		ReferenceID: plan.entity.CarrierAssignmentID,
	})
	if err != nil {
		return nil, err
	}

	after, err := copyRateCon(plan.entity, func(rc *rateconfirmation.RateConfirmation) {
		markSent(rc, plan.recipients, timeutils.NowUnix())
	})
	if err != nil {
		return nil, err
	}

	return &SendPreview{
		Before:     plan.entity,
		After:      after,
		Recipients: plan.recipients,
		Subject:    message.Subject,
		Body:       message.Text,
		Attachment: fileName(plan.entity),
		SignLink: plan.entity.Status != rateconfirmation.StatusConfirmed &&
			s.signLinkConfigured(),
	}, nil
}

// PreviewVoid is what Void would do, from the same checks, without writing.
// A revision already voided comes back unchanged, as Void returns it.
func (s *Service) PreviewVoid(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	reason string,
) (*ChangePreview, error) {
	entity, err := s.planVoid(ctx, tenantInfo, rateConfirmationID, reason)
	if err != nil {
		return nil, err
	}

	out := &ChangePreview{Before: entity, After: entity}
	if entity.Status == rateconfirmation.StatusVoided {
		return out, nil
	}

	out.After, err = copyRateCon(entity, func(rc *rateconfirmation.RateConfirmation) {
		rc.Void(timeutils.NowUnix(), reason)
	})
	if err != nil {
		return nil, err
	}
	if entity.Status != rateconfirmation.StatusConfirmed {
		return out, nil
	}

	return out, s.previewAssignment(ctx, tenantInfo, out, (*shipment.CarrierAssignment).RevertConfirmation)
}

// PreviewMarkConfirmed is what MarkConfirmed would do, from the same checks,
// without writing.
func (s *Service) PreviewMarkConfirmed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	confirmedByName string,
) (*ChangePreview, error) {
	entity, err := s.planConfirm(ctx, tenantInfo, rateConfirmationID, confirmedByName)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	out := &ChangePreview{Before: entity}
	out.After, err = copyRateCon(entity, func(rc *rateconfirmation.RateConfirmation) {
		rc.Confirm(dispatcherConfirmation(confirmedByName, now))
	})
	if err != nil {
		return nil, err
	}

	return out, s.previewAssignment(ctx, tenantInfo, out,
		func(assignment *shipment.CarrierAssignment) bool {
			return assignment.Confirm(now)
		},
	)
}

func (s *Service) planGenerate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*generatePlan, error) {
	if s.templates == nil {
		return nil, errortypes.NewBusinessError(
			"Document templates are not configured; the rate confirmation cannot be rendered",
		)
	}

	assignment, err := s.carrierAssignmentRepo.GetActiveByMoveID(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return nil, errortypes.NewBusinessError(
			"Shipment move has no active carrier assignment to confirm",
		).WithParam("shipmentMoveId", moveID.String())
	}

	carrierEntity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         assignment.CarrierID,
		TenantInfo: tenantInfo,
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeContacts: true,
		},
	})
	if err != nil {
		return nil, err
	}

	shipmentEntity, moveEntity, err := s.loadShipmentAndMove(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}

	return &generatePlan{
		assignment: assignment,
		carrier:    carrierEntity,
		shipment:   shipmentEntity,
		move:       moveEntity,
	}, nil
}

func (s *Service) planSend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
) (*sendPlan, error) {
	if s.templates == nil {
		return nil, errortypes.NewBusinessError(
			"Document templates are not configured; the rate confirmation cannot be rendered",
		)
	}

	entity, err := s.repo.GetByID(ctx, &repositories.GetRateConfirmationByIDRequest{
		TenantInfo:         tenantInfo,
		RateConfirmationID: rateConfirmationID,
	})
	if err != nil {
		return nil, err
	}
	if !entity.CanSend() {
		return nil, errortypes.NewBusinessError(
			"A {0} rate confirmation cannot be sent", entity.Status,
		)
	}
	if s.emailService == nil {
		return nil, errortypes.NewBusinessError(
			"No email service is configured. Download the rate confirmation and deliver it manually",
		)
	}
	if len(entity.PayloadSnapshot) == 0 {
		return nil, errortypes.NewBusinessError(
			"The rate confirmation has no payload snapshot to render from. Regenerate it first",
		)
	}

	// The frozen snapshot re-hydrates into the typed template context; feeding
	// the raw JSON map to html/template rejects trusted values like the logo
	// data URI (ZgotmplZ).
	templateContext, err := contextFromSnapshot(entity.PayloadSnapshot)
	if err != nil {
		return nil, err
	}

	recipients, err := s.recipients(ctx, tenantInfo, entity)
	if err != nil {
		return nil, err
	}

	return &sendPlan{entity: entity, templateContext: templateContext, recipients: recipients}, nil
}

func (s *Service) planVoid(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	reason string,
) (*rateconfirmation.RateConfirmation, error) {
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason", errortypes.ErrRequired, "A void reason is required")
	}

	return s.repo.GetByID(ctx, &repositories.GetRateConfirmationByIDRequest{
		TenantInfo:         tenantInfo,
		RateConfirmationID: rateConfirmationID,
	})
}

func (s *Service) planConfirm(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	confirmedByName string,
) (*rateconfirmation.RateConfirmation, error) {
	entity, err := s.repo.GetByID(ctx, &repositories.GetRateConfirmationByIDRequest{
		TenantInfo:         tenantInfo,
		RateConfirmationID: rateConfirmationID,
	})
	if err != nil {
		return nil, err
	}
	if !entity.CanConfirm() {
		return nil, errortypes.NewBusinessError(
			"A {0} rate confirmation cannot be confirmed", entity.Status,
		)
	}
	if confirmedByName == "" {
		return nil, errortypes.NewValidationError(
			"confirmedByName", errortypes.ErrRequired,
			"The confirming party's name is required")
	}

	return entity, nil
}

// previewAssignment reads the revision's carrier assignment and applies
// change to a copy, keeping it only when change moved it.
func (s *Service) previewAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *ChangePreview,
	change func(*shipment.CarrierAssignment) bool,
) error {
	assignment, err := s.carrierAssignmentRepo.GetByID(
		ctx,
		&repositories.GetCarrierAssignmentByIDRequest{
			TenantInfo:          tenantInfo,
			CarrierAssignmentID: out.Before.CarrierAssignmentID,
		},
	)
	if err != nil {
		return err
	}

	after := new(shipment.CarrierAssignment)
	if err = jsonutils.Convert(assignment, after); err != nil {
		return err
	}
	if !change(after) {
		return nil
	}

	out.AssignmentBefore = assignment
	out.AssignmentAfter = after

	return nil
}

// newRevision is the next revision of the assignment's agreement, before its
// rendered snapshot is attached.
func newRevision(
	tenantInfo pagination.TenantInfo,
	plan *generatePlan,
	revision int64,
	actor *services.RequestActor,
	opts *generateOptions,
) *rateconfirmation.RateConfirmation {
	entity := &rateconfirmation.RateConfirmation{
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		CarrierAssignmentID: plan.assignment.ID,
		CarrierID:           plan.assignment.CarrierID,
		ShipmentID:          plan.shipment.ID,
		ShipmentMoveID:      plan.move.ID,
		Revision:            revision,
		Status:              rateconfirmation.StatusGenerated,
	}
	if actor != nil && !actor.UserID.IsNil() {
		userID := actor.UserID
		entity.GeneratedByID = &userID
	}
	if opts != nil {
		entity.GeneratedVia = opts.Via
		entity.SourceTenderOfferID = opts.SourceTenderOfferID
	}

	return entity
}

// markSent is the bookkeeping a delivered email leaves. Emailing the executed
// copy must not demote a Confirmed agreement back to Sent; only the delivery
// bookkeeping refreshes.
func markSent(entity *rateconfirmation.RateConfirmation, recipients []string, now int64) {
	if entity.Status != rateconfirmation.StatusConfirmed {
		entity.Status = rateconfirmation.StatusSent
	}
	entity.SentAt = &now
	entity.SentToEmails = strings.Join(recipients, ", ")
}

func dispatcherConfirmation(name string, at int64) rateconfirmation.Confirmation {
	return rateconfirmation.Confirmation{Name: name, Via: rateconfirmation.ViaDispatcher, At: at}
}

func (s *Service) signLinkConfigured() bool {
	return s.cfg != nil && s.cfg.Tendering.GetPublicBaseURL() != ""
}

func copyRateCon(
	entity *rateconfirmation.RateConfirmation,
	change func(*rateconfirmation.RateConfirmation),
) (*rateconfirmation.RateConfirmation, error) {
	out := new(rateconfirmation.RateConfirmation)
	if err := jsonutils.Convert(entity, out); err != nil {
		return nil, err
	}
	change(out)

	return out, nil
}
