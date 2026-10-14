package factoringservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	RecipientSpecified   = "Specified"
	RecipientDriver      = "Driver"
	RecipientCarrier     = "Carrier"
	RecipientCustomer    = "Customer"
	maxRequestMessageLen = 4000
	requestKeyPrefix     = "facreq-"
	requestKeyHashLength = 16
)

type documentRequest struct {
	inv       *invoice.Invoice
	needs     []factoring.Check
	recipient []string
	kind      string
}

func (s *ReadinessService) RequestMissingDocuments(
	ctx context.Context,
	req *factoringports.RequestMissingDocumentsRequest,
	actor *permissionports.RequestActor,
) (*factoringports.RequestMissingDocumentsResult, error) {
	if err := requireActor(actor, "Requesting documents for factoring"); err != nil {
		return nil, err
	}
	plan, request, err := s.planDocuments(ctx, req, actor.UserID)
	if err != nil {
		return nil, err
	}
	if _, err = s.email.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     req.TenantInfo,
		Purpose:        email.PurposeBilling,
		To:             plan.Message.To,
		Subject:        plan.Message.Subject,
		HTML:           plan.Message.HTML,
		Text:           plan.Message.Text,
		IdempotencyKey: requestKey(request, s.now()),
	}); err != nil {
		return nil, err
	}

	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceInvoice,
		entity:   request.inv,
		op:       permission.OpUpdate,
		comment: "Asked " + strings.Join(request.recipient, ", ") + " for " +
			strings.Join(plan.DocumentTypes, ", ") + " so the invoice can be factored",
	})
	return &factoringports.RequestMissingDocumentsResult{
		Recipient:     strings.Join(request.recipient, ", "),
		RecipientKind: request.kind,
		DocumentTypes: plan.DocumentTypes,
		ShipmentIDs:   plan.ShipmentIDs,
		Sent:          true,
	}, nil
}

func (s *ReadinessService) PlanDocumentRequest(
	ctx context.Context,
	req *factoringports.RequestMissingDocumentsRequest,
	userID pulid.ID,
) (*factoringports.DocumentRequestPlan, error) {
	plan, _, err := s.planDocuments(ctx, req, userID)
	return plan, err
}

func (s *ReadinessService) planDocuments(
	ctx context.Context,
	req *factoringports.RequestMissingDocumentsRequest,
	userID pulid.ID,
) (*factoringports.DocumentRequestPlan, *documentRequest, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, nil, err
	}
	message := strings.TrimSpace(req.Message)
	if len(message) > maxRequestMessageLen {
		return nil, nil, errortypes.NewValidationError(
			"message",
			errortypes.ErrInvalid,
			"Keep the message under 4,000 characters",
		)
	}
	request, err := s.planDocumentRequest(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	rendered, err := s.renderDocumentRequest(ctx, req.TenantInfo, request, message, userID)
	if err != nil {
		return nil, nil, err
	}
	return &factoringports.DocumentRequestPlan{
		Invoice:       request.inv,
		RecipientKind: request.kind,
		DocumentTypes: neededNames(request.needs),
		ShipmentIDs:   neededShipments(request.needs),
		Message: factoringports.MessagePlan{
			To:        request.recipient,
			Subject:   rendered.Subject,
			HTML:      rendered.HTML,
			Text:      rendered.Text,
			VersionID: rendered.VersionID,
		},
	}, request, nil
}

func (s *ReadinessService) planDocumentRequest(
	ctx context.Context,
	req *factoringports.RequestMissingDocumentsRequest,
) (*documentRequest, error) {
	inv, err := s.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	readiness, err := s.assess(ctx, req.TenantInfo, inv, s.now())
	if err != nil {
		return nil, err
	}

	needs := requestableChecks(readiness.Checks)
	if len(needs) == 0 {
		return nil, errortypes.NewBusinessError(
			"Invoice {0} is not missing any document the factor needs",
			inv.Number,
		)
	}
	if len(req.DocumentTypeIDs) > 0 {
		wanted := make(map[pulid.ID]struct{}, len(req.DocumentTypeIDs))
		for _, id := range req.DocumentTypeIDs {
			wanted[id] = struct{}{}
		}
		filtered := make([]factoring.Check, 0, len(needs))
		for idx := range needs {
			if _, ok := wanted[needs[idx].DocumentTypeID]; ok {
				filtered = append(filtered, needs[idx])
				delete(wanted, needs[idx].DocumentTypeID)
			}
		}
		if len(wanted) > 0 {
			return nil, errortypes.NewValidationError(
				"documentTypeIds",
				errortypes.ErrInvalid,
				"Ask only for documents the invoice is missing",
			)
		}
		needs = filtered
	}

	recipient, kind, err := s.resolveRecipient(ctx, req, inv, needs)
	if err != nil {
		return nil, err
	}
	return &documentRequest{inv: inv, needs: needs, recipient: recipient, kind: kind}, nil
}

func (s *ReadinessService) resolveRecipient(
	ctx context.Context,
	req *factoringports.RequestMissingDocumentsRequest,
	inv *invoice.Invoice,
	needs []factoring.Check,
) ([]string, string, error) {
	if explicit := strings.TrimSpace(req.Recipient); explicit != "" {
		to := stringutils.SplitEmailList(explicit)
		if len(to) == 0 {
			return nil, "", errortypes.NewValidationError(
				"recipient",
				errortypes.ErrInvalid,
				"Enter a valid email address",
			)
		}
		return to, RecipientSpecified, nil
	}

	shipmentIDs := neededShipments(needs)
	if len(shipmentIDs) == 0 {
		shipmentIDs = inv.LegShipmentIDs()
	}
	contacts, err := s.states.ListShipmentContacts(ctx, &factoringports.ListShipmentContactsRequest{
		TenantInfo:  req.TenantInfo,
		ShipmentIDs: shipmentIDs,
	})
	if err != nil {
		return nil, "", err
	}
	for _, id := range shipmentIDs {
		if c := contacts[id]; c != nil && c.DriverEmail != "" {
			return []string{c.DriverEmail}, RecipientDriver, nil
		}
	}
	for _, id := range shipmentIDs {
		if c := contacts[id]; c != nil && c.CarrierEmail != "" {
			return []string{c.CarrierEmail}, RecipientCarrier, nil
		}
	}

	found, err := s.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  req.TenantInfo,
		CustomerIDs: []pulid.ID{inv.CustomerID},
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeEmailProfile: true,
		},
	})
	if err != nil {
		return nil, "", err
	}
	for _, cus := range found {
		if cus == nil || cus.ID != inv.CustomerID {
			continue
		}
		if to := cus.EmailProfile.Recipients().To; len(to) > 0 {
			return to, RecipientCustomer, nil
		}
	}
	return nil, "", errortypes.NewBusinessError(
		"No driver, carrier or customer billing email is on file for invoice {0}; enter a recipient",
		inv.Number,
	)
}

func (s *ReadinessService) renderDocumentRequest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	plan *documentRequest,
	message string,
	userID pulid.ID,
) (*documenttemplateports.RenderedMessage, error) {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	logo, err := documenttemplateports.ResolveLogoDataURI(ctx, s.inliner, org.LogoURL)
	if err != nil {
		s.l.Warn("could not inline the organization logo on a document request", zap.Error(err))
		logo = ""
	}

	names := neededNames(plan.needs)
	body := message
	if body == "" {
		body = "To get paid for invoice " + plan.inv.Number +
			" we need the documents listed below. Please reply with them attached " +
			"or upload them to the shipment."
	}
	data := &documenttemplate.AgentEmailContext{
		AgentSubject:       "Documents needed for invoice " + plan.inv.Number,
		AgentBody:          body,
		CompanyName:        org.Name,
		CustomerName:       plan.inv.BillToName,
		ShipmentProNumber:  singlePro(plan.needs, plan.inv),
		RequestedDocuments: names,
		LogoDataURI:        logo,
	}
	return s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindAgentRequestMissingDocsEmail,
		CustomerID:  &plan.inv.CustomerID,
		Data:        data,
		ReferenceID: plan.inv.ID,
		UserID:      userID,
	})
}

func requestableChecks(checks []factoring.Check) []factoring.Check {
	out := make([]factoring.Check, 0, len(checks))
	for idx := range checks {
		c := checks[idx]
		if !c.Passed && c.Fix == factoring.FixRequestDocument && !c.DocumentTypeID.IsNil() {
			out = append(out, c)
		}
	}
	return out
}

func neededNames(needs []factoring.Check) []string {
	multi := len(neededShipments(needs)) > 1
	names := make([]string, 0, len(needs))
	for idx := range needs {
		name := needs[idx].DocumentTypeName
		if multi && needs[idx].ShipmentPro != "" {
			name += " (" + needs[idx].ShipmentPro + ")"
		}
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

func neededShipments(needs []factoring.Check) []pulid.ID {
	ids := make([]pulid.ID, 0, len(needs))
	for idx := range needs {
		if id := needs[idx].ShipmentID; !id.IsNil() && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids
}

func singlePro(needs []factoring.Check, inv *invoice.Invoice) string {
	if len(neededShipments(needs)) > 1 {
		return ""
	}
	for idx := range needs {
		if needs[idx].ShipmentPro != "" {
			return needs[idx].ShipmentPro
		}
	}
	return inv.ShipmentProNumber
}

func requestKey(plan *documentRequest, now int64) string {
	parts := make([]string, 0, len(plan.needs)+len(plan.recipient))
	for idx := range plan.needs {
		parts = append(parts, plan.needs[idx].DocumentTypeID.String()+"@"+
			plan.needs[idx].ShipmentID.String())
	}
	slices.Sort(parts)
	recipients := slices.Clone(plan.recipient)
	slices.Sort(recipients)
	parts = append(parts, recipients...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	day := now / timeutils.SecondsPerDay
	return requestKeyPrefix + plan.inv.ID.String() + "-" +
		hex.EncodeToString(sum[:])[:requestKeyHashLength] + "-" + strconv.FormatInt(day, 10)
}
