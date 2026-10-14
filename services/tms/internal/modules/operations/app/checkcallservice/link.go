package checkcallservice

import (
	"context"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const trackingLinkPath = "/carrier-tracking/"

func (s *Service) IssueLink(
	ctx context.Context,
	req *checkcallports.IssueTrackingLinkRequest,
) (*checkcallports.IssuedTrackingLink, error) {
	if multiErr := validateIssueRequest(req); multiErr != nil {
		return nil, multiErr
	}
	base := s.cfg.PublicLinkBaseURL()
	if base == "" {
		return nil, errortypes.NewBusinessError(
			"Carrier tracking links need a public base URL; set app.webBaseUrl",
		)
	}
	if req.RecipientEmail != "" && (s.templates == nil || s.emailService == nil) {
		return nil, errortypes.NewBusinessError(
			"Email delivery is not configured for this environment",
		)
	}

	move, err := s.loadMove(ctx, req.TenantInfo, req.MoveID)
	if err != nil {
		return nil, err
	}
	assignment, err := s.trackableAssignment(ctx, req.TenantInfo, move)
	if err != nil {
		return nil, err
	}

	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}
	now := s.now()
	link := &checkcall.TrackingLink{
		ID:             checkcall.NewTrackingLinkID(),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ShipmentID:     move.ShipmentID,
		ShipmentMoveID: move.ID,
		CarrierID:      assignment.CarrierID,
		TokenHash:      tokenHash,
		RecipientEmail: req.RecipientEmail,
		ExpiresAt:      now + checkcall.DefaultLinkTTLSeconds,
		CreatedByID:    req.IssuedByID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, rErr := s.links.RevokeForMove(txCtx, &checkcallports.RevokeTrackingLinksRequest{
			TenantInfo: req.TenantInfo,
			MoveID:     move.ID,
			At:         now,
		}); rErr != nil {
			return rErr
		}
		return s.links.Create(txCtx, link)
	})
	if err != nil {
		return nil, err
	}

	issued := &checkcallports.IssuedTrackingLink{
		Link: link,
		URL:  base + trackingLinkPath + url.PathEscape(token),
	}
	s.logLinkAudit(&linkAuditParams{
		tenant:    req.TenantInfo,
		userID:    req.IssuedByID,
		link:      link,
		operation: permission.OpCreate,
		comment:   "Issued a carrier tracking link",
	})

	if req.RecipientEmail != "" {
		if sendErr := s.sendLinkEmail(ctx, req.TenantInfo, move, link, issued.URL); sendErr != nil {
			s.l.Warn("failed to email carrier tracking link",
				zap.String("linkId", link.ID.String()),
				zap.Error(sendErr))
			issued.EmailError = sendErr.Error()
		} else {
			issued.Emailed = true
		}
	}
	return issued, nil
}

func (s *Service) RevokeLink(
	ctx context.Context,
	req *checkcallports.RevokeTrackingLinkRequest,
) error {
	if req == nil || req.MoveID.IsNil() {
		return errortypes.NewValidationError(
			"moveId",
			errortypes.ErrRequired,
			"Move ID is required",
		)
	}
	now := s.now()
	active, err := s.links.GetActiveForMove(ctx, req.TenantInfo, req.MoveID, now)
	if err != nil {
		return err
	}
	if active == nil {
		return errortypes.NewNotFoundError("This load has no active carrier tracking link")
	}
	if _, err = s.links.RevokeForMove(ctx, &checkcallports.RevokeTrackingLinksRequest{
		TenantInfo: req.TenantInfo,
		MoveID:     req.MoveID,
		At:         now,
	}); err != nil {
		return err
	}
	active.RevokedAt = &now
	s.logLinkAudit(&linkAuditParams{
		tenant:    req.TenantInfo,
		userID:    req.TenantInfo.UserID,
		link:      active,
		operation: permission.OpUpdate,
		comment:   "Revoked the carrier tracking link",
	})
	return nil
}

func (s *Service) ActiveLink(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*checkcall.TrackingLink, error) {
	if moveID.IsNil() {
		return nil, errortypes.NewValidationError(
			"moveId",
			errortypes.ErrRequired,
			"Move ID is required",
		)
	}
	return s.links.GetActiveForMove(ctx, tenantInfo, moveID, s.now())
}

func (s *Service) trackableAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
) (*shipment.CarrierAssignment, error) {
	if moveIsClosed(move) {
		return nil, errortypes.NewBusinessError(
			"This load is finished, so there is nothing left for a carrier to track",
		)
	}
	if !move.IsCarrierCovered() {
		return nil, errortypes.NewBusinessError(
			"Only a load covered by an outside carrier gets a carrier tracking link",
		)
	}
	assignment, err := s.carrierAssignments.GetActiveByMoveID(ctx, tenantInfo, move.ID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewBusinessError(
				"Assign a carrier to this load before sending a tracking link",
			)
		}
		return nil, err
	}
	if assignment == nil || assignment.Status == shipment.CarrierAssignmentStatusCanceled {
		return nil, errortypes.NewBusinessError(
			"Assign a carrier to this load before sending a tracking link",
		)
	}
	return assignment, nil
}

func (s *Service) sendLinkEmail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	link *checkcall.TrackingLink,
	trackingURL string,
) error {
	data := documenttemplate.CarrierTrackingLinkEmailContext{
		CarrierName: s.carrierName(ctx, tenantInfo, link.CarrierID),
		CompanyName: s.companyName(ctx, tenantInfo),
		//nolint:gosec // G203: an operator-configured base URL and a server-minted, path-escaped token
		TrackingURL: template.URL(trackingURL),
		ExpiresAt:   timeutils.WindowLabelUTC(link.ExpiresAt, nil),
	}
	if sh, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         move.ShipmentID,
		TenantInfo: tenantInfo,
	}); err == nil && sh != nil {
		data.ShipmentProNumber = sh.ProNumber
	}
	for _, stop := range move.ActiveStopsBySequence() {
		summary := stopPlace(stop)
		window := timeutils.WindowLabelUTC(stop.ScheduledWindowStart, stop.ScheduledWindowEnd)
		if stop.IsOriginStop() && data.OriginSummary == "" {
			data.OriginSummary, data.PickupWindow = summary, window
		}
		if stop.IsDestinationStop() {
			data.DestinationSummary, data.DeliveryWindow = summary, window
		}
	}

	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:        tenantInfo,
		Kind:              documenttemplate.KindCarrierTrackingLinkEmail,
		Data:              data,
		ReferenceID:       link.ID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		return fmt.Errorf("render carrier tracking link email: %w", err)
	}
	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeGeneral,
		To:             []string{link.RecipientEmail},
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: "carrier-tracking-link-" + link.ID.String(),
	}); err != nil {
		return fmt.Errorf("send carrier tracking link email: %w", err)
	}
	return nil
}

func (s *Service) carrierName(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) string {
	if s.carrierRepo == nil || carrierID.IsNil() {
		return ""
	}
	entity, err := s.carrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
		ID:         carrierID,
		TenantInfo: tenantInfo,
	})
	if err != nil || entity == nil {
		return ""
	}
	return entity.Name
}

func (s *Service) companyName(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	if s.orgRepo == nil {
		return ""
	}
	org, err := s.orgRepo.GetByID(
		ctx,
		tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo},
	)
	if err != nil || org == nil {
		return ""
	}
	return org.Name
}

func stopPlace(stop *shipment.Stop) string {
	if stop.Location == nil {
		return ""
	}
	place := stop.Location.City
	if stop.Location.State != nil && stop.Location.State.Abbreviation != "" {
		place += ", " + stop.Location.State.Abbreviation
	}
	return place
}

func validateIssueRequest(req *checkcallports.IssueTrackingLinkRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("", errortypes.ErrInvalid, "Request is required")
		return multiErr
	}
	addTenant(multiErr, req.TenantInfo)
	if req.MoveID.IsNil() {
		multiErr.Add("moveId", errortypes.ErrRequired, "Move ID is required")
	}
	req.RecipientEmail = strings.TrimSpace(req.RecipientEmail)
	if req.RecipientEmail != "" {
		if err := validation.Validate(req.RecipientEmail,
			validation.RuneLength(0, checkcall.MaxRecipientLength),
			is.EmailFormat,
		); err != nil {
			multiErr.Add(
				"recipientEmail",
				errortypes.ErrInvalid,
				"Recipient email must be a valid email address",
			)
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type linkAuditParams struct {
	tenant    pagination.TenantInfo
	userID    pulid.ID
	link      *checkcall.TrackingLink
	operation permission.Operation
	comment   string
}

func (s *Service) logLinkAudit(p *linkAuditParams) {
	if s.auditService == nil || p.link == nil {
		return
	}
	params := &auditports.LogActionParams{
		Resource:   permission.ResourceShipmentMove,
		ResourceID: p.link.ShipmentMoveID.String(),
		Operation:  p.operation,
		UserID:     p.userID,
		CurrentState: map[string]any{
			"trackingLinkId": p.link.ID.String(),
			"carrierId":      p.link.CarrierID.String(),
			"recipientEmail": p.link.RecipientEmail,
			"expiresAt":      p.link.ExpiresAt,
			"revokedAt":      p.link.RevokedAt,
		},
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
	}
	if p.userID.IsNil() {
		params.PrincipalType = permissionports.PrincipalTypeSystem
		params.PrincipalID = permissionports.SystemPrincipalID
	}
	comment := p.comment
	if err := s.auditService.LogAction(params, func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}); err != nil {
		s.l.Error("failed to log carrier tracking link audit", zap.Error(err))
	}
}
