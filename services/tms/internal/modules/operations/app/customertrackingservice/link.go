package customertrackingservice

import (
	"context"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"go.uber.org/zap"
)

const trackingPagePath = "/shipment-tracking/"

type mintedLink struct {
	link *customertracking.Link
	url  string
}

// mintLink creates a live link for the shipment and returns its URL. The raw
// token exists only in that URL; the table keeps its hash.
func (s *Service) mintLink(
	ctx context.Context,
	sh *shipment.Shipment,
	profile *customertracking.Profile,
	purpose customertracking.LinkPurpose,
	recipient string,
	createdByID pulid.ID,
) (*mintedLink, error) {
	base := s.cfg.Tendering.GetPublicBaseURL()
	if base == "" {
		return nil, errortypes.NewBusinessError(
			"Customer tracking links need a public base URL; set tendering.publicBaseUrl",
		)
	}
	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}
	now := s.now()
	link := &customertracking.Link{
		ID:             customertracking.NewLinkID(),
		OrganizationID: sh.OrganizationID,
		BusinessUnitID: sh.BusinessUnitID,
		ShipmentID:     sh.ID,
		CustomerID:     sh.CustomerID,
		TokenHash:      tokenHash,
		Purpose:        purpose,
		RecipientEmail: recipient,
		ExpiresAt:      now + profile.LinkTTLSeconds(),
		CreatedByID:    createdByID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err = s.links.Create(ctx, link); err != nil {
		return nil, err
	}
	return &mintedLink{link: link, url: base + trackingPagePath + url.PathEscape(token)}, nil
}

func (s *Service) IssueLink(
	ctx context.Context,
	req *customertrackingports.IssueLinkRequest,
) (*customertrackingports.IssuedLink, error) {
	if multiErr := validateIssueRequest(req); multiErr != nil {
		return nil, multiErr
	}
	if req.RecipientEmail != "" && !s.canEmail() {
		return nil, errortypes.NewBusinessError(
			"Email delivery is not configured for this environment",
		)
	}

	sh, err := s.loadShipment(ctx, req.TenantInfo, req.ShipmentID)
	if err != nil {
		return nil, err
	}
	if sh.IsCanceled() {
		return nil, errortypes.NewBusinessError(
			"This shipment is canceled, so there is nothing to track",
		)
	}
	profile, err := s.profileOrDefault(ctx, req.TenantInfo, sh.CustomerID)
	if err != nil {
		return nil, err
	}
	if !profile.LinkEnabled {
		return nil, errortypes.NewBusinessError(
			"Tracking links are turned off for this customer; turn them on in the customer's tracking settings",
		)
	}

	minted, err := s.mintLink(
		ctx,
		sh,
		profile,
		customertracking.LinkPurposeShared,
		req.RecipientEmail,
		req.IssuedByID,
	)
	if err != nil {
		return nil, err
	}
	issued := &customertrackingports.IssuedLink{Link: minted.link, URL: minted.url}
	s.audit(&auditParams{
		tenant:     req.TenantInfo,
		userID:     req.IssuedByID,
		resource:   permission.ResourceShipment,
		resourceID: sh.ID.String(),
		operation:  permission.OpUpdate,
		comment:    "Shared a customer tracking link",
		current:    linkState(minted.link),
	})

	if req.RecipientEmail != "" {
		if sendErr := s.sendLinkEmail(ctx, req.TenantInfo, sh, minted); sendErr != nil {
			s.l.Warn("failed to email customer tracking link",
				zap.String("linkId", minted.link.ID.String()),
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
	req *customertrackingports.RevokeLinkRequest,
) (int, error) {
	if req == nil || req.ShipmentID.IsNil() {
		return 0, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required",
		)
	}
	now := s.now()
	revoked, err := s.links.Revoke(ctx, &customertrackingports.RevokeLinksRequest{
		TenantInfo: req.TenantInfo,
		ShipmentID: req.ShipmentID,
		LinkID:     req.LinkID,
		At:         now,
	})
	if err != nil {
		return 0, err
	}
	if revoked == 0 {
		return 0, errortypes.NewNotFoundError("This shipment has no live customer tracking link")
	}
	state := map[string]any{"revokedAt": now, "revokedLinks": revoked}
	if !req.LinkID.IsNil() {
		state["trackingLinkId"] = req.LinkID.String()
	}
	s.audit(&auditParams{
		tenant:     req.TenantInfo,
		userID:     req.RevokedByID,
		resource:   permission.ResourceShipment,
		resourceID: req.ShipmentID.String(),
		operation:  permission.OpUpdate,
		comment:    "Revoked customer tracking links",
		current:    state,
	})
	return revoked, nil
}

func (s *Service) ListLinks(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*customertracking.Link, error) {
	if shipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required",
		)
	}
	return s.links.ListForShipment(ctx, tenantInfo, shipmentID)
}

func (s *Service) ListNotifications(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*customertracking.Notification, error) {
	if shipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required",
		)
	}
	return s.notifications.ListForShipment(ctx, tenantInfo, shipmentID)
}

func (s *Service) sendLinkEmail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sh *shipment.Shipment,
	minted *mintedLink,
) error {
	stops := route(sh)
	origin, destination := originAndDestination(stops)
	data := documenttemplate.CustomerTrackingLinkEmailContext{
		CompanyName:       s.companyName(ctx, tenantInfo),
		ShipmentProNumber: sh.ProNumber,
		BOL:               sh.BOL,
		//nolint:gosec // G203: an operator-configured base URL and a server-minted, path-escaped token
		TrackingURL: template.URL(minted.url),
		ExpiresAt:   timeutils.WindowLabelUTC(minted.link.ExpiresAt, nil),
	}
	if sh.Customer != nil {
		data.CustomerName = sh.Customer.Name
	}
	if origin != nil {
		data.OriginSummary = placeSummary(origin)
		data.PickupWindow = stopWindowLabel(origin)
	}
	if destination != nil {
		data.DestinationSummary = placeSummary(destination)
		data.DeliveryWindow = stopWindowLabel(destination)
	}

	customerID := sh.CustomerID
	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:        tenantInfo,
		Kind:              documenttemplate.KindCustomerTrackingLinkEmail,
		CustomerID:        &customerID,
		Data:              data,
		ReferenceID:       minted.link.ID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		return fmt.Errorf("render customer tracking link email: %w", err)
	}
	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeOperations,
		To:             []string{minted.link.RecipientEmail},
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: "customer-tracking-link-" + minted.link.ID.String(),
	}); err != nil {
		return fmt.Errorf("send customer tracking link email: %w", err)
	}
	return nil
}

func linkState(link *customertracking.Link) map[string]any {
	return map[string]any{
		"trackingLinkId": link.ID.String(),
		"purpose":        link.Purpose,
		"recipientEmail": link.RecipientEmail,
		"expiresAt":      link.ExpiresAt,
	}
}

func validateIssueRequest(req *customertrackingports.IssueLinkRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("", errortypes.ErrInvalid, "Request is required")
		return multiErr
	}
	if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		multiErr.Add(
			"tenantInfo",
			errortypes.ErrRequired,
			"Organization and business unit are required",
		)
	}
	if req.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	req.RecipientEmail = strings.TrimSpace(req.RecipientEmail)
	if req.RecipientEmail != "" {
		if err := validation.Validate(req.RecipientEmail,
			validation.RuneLength(0, customertracking.MaxRecipientEmailLength),
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
