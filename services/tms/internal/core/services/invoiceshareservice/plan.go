package invoiceshareservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type sharePlan struct {
	invoice    *invoice.Invoice
	sharer     *tenant.User
	recipients []*tenant.User
	shares     []*invoice.InvoiceShare
	input      shareInput
}

func (s *Service) planShare(
	ctx context.Context,
	req *servicesports.ShareInvoiceRequest,
	actor *servicesports.RequestActor,
) (*sharePlan, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if !actor.IsUser() || actor.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrInvalidOperation,
			"Only a signed-in user can share an invoice",
		)
	}

	plan := &sharePlan{input: shareInput{
		userIDs: sliceutils.Dedupe(req.UserIDs),
		note:    strings.TrimSpace(req.Note),
		tab:     req.Tab,
	}}
	if plan.input.tab == "" {
		plan.input.tab = invoice.ShareTabOverview
	}
	if multiErr := validateShareInput(&plan.input, actor.UserID); multiErr != nil {
		return nil, multiErr
	}

	var err error
	if plan.invoice, err = s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}
	if plan.sharer, err = s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   req.TenantInfo,
		LookupUserID: actor.UserID,
	}); err != nil {
		return nil, err
	}
	if plan.recipients, err = s.resolveRecipients(ctx, req.TenantInfo, plan.input.userIDs); err != nil {
		return nil, err
	}
	if plan.shares, err = buildShares(
		plan.invoice,
		plan.sharer.ID,
		plan.recipients,
		&plan.input,
	); err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *Service) PreviewShare(
	ctx context.Context,
	req *servicesports.ShareInvoiceRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceSharePreview, error) {
	plan, err := s.planShare(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  plan.invoice.OrganizationID,
		BuID:   plan.invoice.BusinessUnitID,
		UserID: plan.sharer.ID,
	}
	baseURL := s.cfg.App.GetWebBaseURL()
	canEmail := baseURL != "" && s.emailService != nil && s.templates != nil
	companyName := ""
	if canEmail {
		companyName = s.companyName(ctx, tenantInfo)
	}
	delivery := &deliveryParams{
		invoice:    plan.invoice,
		sharer:     plan.sharer,
		recipients: plan.recipients,
		note:       plan.input.note,
		tab:        plan.input.tab,
	}
	path := invoiceSharePath(plan.invoice.ID.String(), plan.input.tab)
	sentAt := timeutils.NowUnix()

	preview := &servicesports.InvoiceSharePreview{
		Invoice:         plan.invoice,
		SharedByName:    plan.sharer.Name,
		Note:            plan.input.note,
		Tab:             plan.input.tab,
		EmailConfigured: canEmail,
		Recipients:      make([]servicesports.InvoiceShareRecipientPreview, 0, len(plan.recipients)),
	}
	for _, recipient := range plan.recipients {
		entry := servicesports.InvoiceShareRecipientPreview{
			UserID:       recipient.ID,
			Name:         recipient.Name,
			EmailAddress: strings.TrimSpace(recipient.EmailAddress),
		}
		if canEmail && entry.EmailAddress != "" {
			rendered, renderErr := s.renderShareEmail(ctx, &emailParams{
				tenantInfo:  tenantInfo,
				delivery:    delivery,
				recipient:   recipient,
				invoiceURL:  baseURL + path,
				companyName: companyName,
				sentAt:      sentAt,
			})
			if renderErr != nil {
				return nil, renderErr
			}
			entry.Emailed = true
			entry.Subject = rendered.Subject
			entry.Body = rendered.Text
		}
		preview.Recipients = append(preview.Recipients, entry)
	}

	return preview, nil
}
