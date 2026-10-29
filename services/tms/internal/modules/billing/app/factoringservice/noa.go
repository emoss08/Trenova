package factoringservice

import (
	"context"
	"html/template"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/factoringtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	pdfContentType       = "application/pdf"
	maxPacketBytes       = 256 << 20
	noaFileNameFallback  = "Notice-of-Assignment.pdf"
	maxNoaFileNameLength = 120
	noaSendKeyPrefix     = "facnoa-"
)

type NoaParams struct {
	fx.In

	Repo          factoringports.NoaRepository
	Customers     customerports.CustomerRepository
	Factors       factoringcompanyports.FactoringCompanyRepository
	Documents     documentports.DocumentRepository
	Organizations tenantports.OrganizationRepository
	Templates     documenttemplateports.DocumentTemplateResolver
	Inliner       documenttemplateports.AssetInliner `optional:"true"`
	Email         emailports.EmailService
	Requester     factoringports.EvaluationRequester `optional:"true"`
	Audit         auditports.AuditService
	Realtime      realtimeports.RealtimeService `optional:"true"`
	Logger        *zap.Logger
}

type NoaService struct {
	repo          factoringports.NoaRepository
	customers     customerports.CustomerRepository
	factors       factoringcompanyports.FactoringCompanyRepository
	documents     documentports.DocumentRepository
	organizations tenantports.OrganizationRepository
	templates     documenttemplateports.DocumentTemplateResolver
	inliner       documenttemplateports.AssetInliner
	email         emailports.EmailService
	requester     factoringports.EvaluationRequester
	rec           recorder
	l             *zap.Logger
	now           func() int64
}

var _ factoringports.NoaService = (*NoaService)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewNoaService(p NoaParams) factoringports.NoaService {
	l := p.Logger.Named("service.factoring-noa")
	return &NoaService{
		repo:          p.Repo,
		customers:     p.Customers,
		factors:       p.Factors,
		documents:     p.Documents,
		organizations: p.Organizations,
		templates:     p.Templates,
		inliner:       p.Inliner,
		email:         p.Email,
		requester:     p.Requester,
		rec:           recorder{audit: p.Audit, realtime: p.Realtime, l: l},
		l:             l,
		now:           timeutils.NowUnix,
	}
}

func (s *NoaService) List(
	ctx context.Context,
	req *factoringports.ListNoaConnectionRequest,
) (*pagination.CursorListResult[*factoring.Noa], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *NoaService) Get(
	ctx context.Context,
	req factoringports.GetNoaRequest,
) (*factoring.Noa, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *NoaService) Draft(
	ctx context.Context,
	req *factoringports.DraftNoaRequest,
	actor *permissionports.RequestActor,
) (*factoring.Noa, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Drafting a notice of assignment"); err != nil {
		return nil, err
	}
	if _, err := s.loadCustomer(ctx, req.TenantInfo, req.CustomerID); err != nil {
		return nil, err
	}
	factor, err := s.loadFactor(ctx, req.TenantInfo, req.FactoringCompanyID)
	if err != nil {
		return nil, err
	}
	if !factor.IsActive() {
		return nil, errortypes.NewValidationError(
			"factoringCompanyId",
			errortypes.ErrInvalid,
			"The factoring company is inactive",
		)
	}

	entity := &factoring.Noa{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		CustomerID:         req.CustomerID,
		FactoringCompanyID: req.FactoringCompanyID,
		Status:             factoring.NoaStatusDraft,
		Notes:              strings.TrimSpace(req.Notes),
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if err = validationOf(multiErr); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceFactoringNoa,
		entity:   created,
		op:       permission.OpCreate,
		comment:  "Notice of assignment drafted",
	})
	s.rec.publish(ctx, permission.ResourceFactoringNoa, created, actor, "created")
	return created, nil
}

func (s *NoaService) RenderLetter(
	ctx context.Context,
	req factoringports.GetNoaRequest,
) (*factoringports.RenderedLetter, error) {
	noa, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	letter, err := s.prepare(ctx, req.TenantInfo, noa, req.TenantInfo.UserID)
	if err != nil {
		return nil, err
	}
	return &factoringports.RenderedLetter{FileName: letter.fileName, PDF: letter.document.PDF}, nil
}

func (s *NoaService) Send(
	ctx context.Context,
	req *factoringports.SendNoaRequest,
	actor *permissionports.RequestActor,
) (*factoring.Noa, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Sending a notice of assignment"); err != nil {
		return nil, err
	}

	plan, letter, err := s.planSend(ctx, req, actor.UserID)
	if err != nil {
		return nil, err
	}
	if err = s.send(ctx, req.TenantInfo, letter, &plan.Message); err != nil {
		return nil, err
	}

	current := plan.Noa
	previous := *current
	if err = current.MarkSent(s.now(), plan.Message.To); err != nil {
		return nil, err
	}
	updated, err := s.save(ctx, current)
	if err != nil {
		return nil, err
	}

	s.recordLetter(ctx, req.TenantInfo, letter, updated, actor.UserID)
	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceFactoringNoa,
		entity:   updated,
		previous: &previous,
		op:       permission.OpUpdate,
		comment:  "Notice of assignment sent to " + updated.SentTo,
	})
	s.rec.publish(ctx, permission.ResourceFactoringNoa, updated, actor, "updated")
	s.requestCustomer(ctx, req.TenantInfo, updated.CustomerID)
	return updated, nil
}

func (s *NoaService) Acknowledge(
	ctx context.Context,
	req *factoringports.AcknowledgeNoaRequest,
	actor *permissionports.RequestActor,
) (*factoring.Noa, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Recording a notice acknowledgement"); err != nil {
		return nil, err
	}

	current, err := s.repo.GetByID(ctx, factoringports.GetNoaRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = s.checkSignedCopy(ctx, req.TenantInfo, current.ID, req.SignedDocumentID); err != nil {
		return nil, err
	}

	previous := *current
	current.Version = req.Version
	if err = current.Acknowledge(
		s.now(),
		req.AcknowledgedAt,
		req.SignedDocumentID,
		req.ExpiresAt,
	); err != nil {
		return nil, err
	}
	updated, err := s.save(ctx, current)
	if err != nil {
		return nil, err
	}

	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceFactoringNoa,
		entity:   updated,
		previous: &previous,
		op:       permission.OpUpdate,
		comment:  "Notice of assignment acknowledged by the customer",
	})
	s.rec.publish(ctx, permission.ResourceFactoringNoa, updated, actor, "updated")
	s.requestCustomer(ctx, req.TenantInfo, updated.CustomerID)
	return updated, nil
}

func (s *NoaService) Release(
	ctx context.Context,
	req *factoringports.ReleaseNoaRequest,
	actor *permissionports.RequestActor,
) (*factoring.Noa, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Releasing a notice of assignment"); err != nil {
		return nil, err
	}
	current, err := s.repo.GetByID(ctx, factoringports.GetNoaRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	previous := *current
	current.Version = req.Version
	if err = current.Release(s.now(), req.Reason); err != nil {
		return nil, err
	}
	updated, err := s.save(ctx, current)
	if err != nil {
		return nil, err
	}

	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceFactoringNoa,
		entity:   updated,
		previous: &previous,
		op:       permission.OpUpdate,
		comment:  "Notice of assignment released: " + updated.ReleaseReason,
	})
	s.rec.publish(ctx, permission.ResourceFactoringNoa, updated, actor, "updated")
	s.requestCustomer(ctx, req.TenantInfo, updated.CustomerID)
	return updated, nil
}

func (s *NoaService) save(ctx context.Context, entity *factoring.Noa) (*factoring.Noa, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if err := validationOf(multiErr); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, entity)
}

func (s *NoaService) checkSignedCopy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	noaID pulid.ID,
	documentID pulid.ID,
) error {
	if documentID.IsNil() {
		return errortypes.NewValidationError(
			"signedDocumentId",
			errortypes.ErrRequired,
			"Attach the customer's signed copy of the notice",
		)
	}
	doc, err := s.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if doc.ResourceType != factoring.NoaDocumentResourceType || doc.ResourceID != noaID.String() {
		return errortypes.NewValidationError(
			"signedDocumentId",
			errortypes.ErrInvalid,
			"Upload the signed copy to this notice of assignment",
		)
	}
	return nil
}

type preparedLetter struct {
	org      *tenant.Organization
	customer *customer.Customer
	factor   *factoringcompany.FactoringCompany
	noa      *factoring.Noa
	logo     template.URL
	fileName string
	document *documenttemplateports.RenderedDocument
}

func (s *NoaService) prepare(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	noa *factoring.Noa,
	userID pulid.ID,
) (*preparedLetter, error) {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
	if err != nil {
		return nil, err
	}
	cus, err := s.loadCustomer(ctx, tenantInfo, noa.CustomerID)
	if err != nil {
		return nil, err
	}
	factor, err := s.loadFactor(ctx, tenantInfo, noa.FactoringCompanyID)
	if err != nil {
		return nil, err
	}
	logo, err := documenttemplateports.ResolveLogoDataURI(ctx, s.inliner, org.LogoURL)
	if err != nil {
		s.l.Warn("could not inline the organization logo on a notice of assignment", zap.Error(err))
		logo = ""
	}

	out := &preparedLetter{org: org, customer: cus, factor: factor, noa: noa, logo: logo}
	out.fileName = fileutils.SanitizeDisplayFilename(
		"Notice-of-Assignment-"+customerLabel(cus)+".pdf",
		noaFileNameFallback,
		maxNoaFileNameLength,
	)
	out.document, err = s.templates.RenderDocument(
		ctx,
		&documenttemplateports.RenderDocumentRequest{
			TenantInfo:  tenantInfo,
			Kind:        documenttemplate.KindFactoringNoaPDF,
			CustomerID:  &cus.ID,
			Data:        letterContext(out, s.now()),
			ReferenceID: noa.ID,
			UserID:      userID,
			Title:       strings.TrimSuffix(out.fileName, ".pdf"),
		},
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *NoaService) PlanSend(
	ctx context.Context,
	req *factoringports.SendNoaRequest,
	userID pulid.ID,
) (*factoringports.NoaSendPlan, error) {
	plan, _, err := s.planSend(ctx, req, userID)
	return plan, err
}

func (s *NoaService) planSend(
	ctx context.Context,
	req *factoringports.SendNoaRequest,
	userID pulid.ID,
) (*factoringports.NoaSendPlan, *preparedLetter, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, nil, err
	}
	current, err := s.repo.GetByID(ctx, factoringports.GetNoaRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if !current.CanSend() {
		return nil, nil, errortypes.NewBusinessError(
			"Only a drafted or sent notice can be sent; draft a new notice instead",
		)
	}
	if current.Version != req.Version {
		return nil, nil, errortypes.NewConflictError(
			"The notice of assignment changed since you opened it; reload and try again",
		)
	}

	letter, err := s.prepare(ctx, req.TenantInfo, current, userID)
	if err != nil {
		return nil, nil, err
	}
	recipients, err := recipientsFor(letter.customer, req.Recipient)
	if err != nil {
		return nil, nil, err
	}
	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:  req.TenantInfo,
		Kind:        documenttemplate.KindFactoringNoaEmail,
		CustomerID:  &letter.customer.ID,
		Data:        letterEmailContext(letter, s.now()),
		ReferenceID: current.ID,
		UserID:      userID,
	})
	if err != nil {
		return nil, nil, err
	}

	return &factoringports.NoaSendPlan{
		Noa: current,
		Message: factoringports.MessagePlan{
			To:             recipients.To,
			CC:             recipients.CC,
			BCC:            recipients.BCC,
			Subject:        rendered.Subject,
			HTML:           rendered.HTML,
			Text:           rendered.Text,
			AttachmentName: letter.fileName,
			VersionID:      rendered.VersionID,
		},
	}, letter, nil
}

func (s *NoaService) send(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	letter *preparedLetter,
	message *factoringports.MessagePlan,
) error {
	_, err := s.email.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo: tenantInfo,
		Purpose:    email.PurposeBilling,
		To:         message.To,
		CC:         message.CC,
		BCC:        message.BCC,
		Subject:    message.Subject,
		HTML:       message.HTML,
		Text:       message.Text,
		Attachments: []emailports.EmailAttachment{{
			FileName:    letter.fileName,
			ContentType: pdfContentType,
			Content:     letter.document.PDF,
			SizeBytes:   int64(len(letter.document.PDF)),
		}},
		IdempotencyKey: noaSendKey(letter.noa),
	})
	return err
}

func (s *NoaService) recordLetter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	letter *preparedLetter,
	noa *factoring.Noa,
	userID pulid.ID,
) {
	if _, err := s.templates.RecordGeneratedDocument(
		ctx,
		&documenttemplateports.RecordGeneratedDocumentRequest{
			TenantInfo:    tenantInfo,
			Kind:          documenttemplate.KindFactoringNoaPDF,
			Rendered:      letter.document,
			ReferenceType: factoring.NoaDocumentResourceType,
			ReferenceID:   noa.ID,
			FileName:      letter.fileName,
			FileSize:      int64(len(letter.document.PDF)),
			UserID:        userID,
		},
	); err != nil {
		s.l.Warn("could not record the generated notice of assignment",
			zap.String("noaId", noa.ID.String()), zap.Error(err))
	}
}

func (s *NoaService) loadCustomer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*customer.Customer, error) {
	if id.IsNil() {
		return nil, errortypes.NewValidationError(
			"customerId",
			errortypes.ErrRequired,
			"Customer is required",
		)
	}
	found, err := s.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: []pulid.ID{id},
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeState:          true,
			IncludeBillingProfile: true,
			IncludeEmailProfile:   true,
		},
	})
	if err != nil {
		return nil, err
	}
	for _, cus := range found {
		if cus != nil && cus.ID == id {
			return cus, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Customer not found")
}

func (s *NoaService) loadFactor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*factoringcompany.FactoringCompany, error) {
	if id.IsNil() {
		return nil, errortypes.NewValidationError(
			"factoringCompanyId",
			errortypes.ErrRequired,
			"Factoring company is required",
		)
	}
	return s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:           id,
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
}

func (s *NoaService) requestCustomer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) {
	if s.requester == nil {
		return
	}
	s.requester.RequestEvaluation(ctx, tenantInfo, &factoringports.EvaluationTargets{
		CustomerIDs: []pulid.ID{customerID},
	})
}

func recipientsFor(cus *customer.Customer, override string) (customer.EmailRecipients, error) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		to := stringutils.SplitEmailList(trimmed)
		if len(to) == 0 {
			return customer.EmailRecipients{}, errortypes.NewValidationError(
				"recipient",
				errortypes.ErrInvalid,
				"Enter a valid email address",
			)
		}
		return customer.EmailRecipients{To: to}, nil
	}
	recipients := cus.EmailProfile.Recipients()
	if len(recipients.To) == 0 {
		return recipients, errortypes.NewBusinessError(
			"{0} has no invoice email recipients. Add one to the customer's email profile or enter a recipient.",
			cus.DisplayName(),
		)
	}
	return recipients, nil
}

func noaSendKey(noa *factoring.Noa) string {
	return noaSendKeyPrefix + noa.ID.String() + "-send-" + strconv.FormatInt(noa.Version, 10)
}

func customerLabel(cus *customer.Customer) string {
	if code := strings.TrimSpace(cus.Code); code != "" {
		return code
	}
	return cus.DisplayName()
}

func letterContext(p *preparedLetter, now int64) *factoringtemplate.FactoringNoaContext {
	tz := p.org.Timezone
	effective := now
	if p.noa.SentAt != nil {
		effective = *p.noa.SentAt
	}
	return &factoringtemplate.FactoringNoaContext{
		LetterDate:    timeutils.FormatUnixDateIn(now, tz),
		EffectiveDate: timeutils.FormatUnixDateIn(effective, tz),
		Reference:     p.noa.ID.String(),
		Organization: templatecatalog.AddressBlock{
			Name:  p.org.Name,
			Lines: p.org.MailingAddress().Lines(),
		},
		Customer: templatecatalog.AddressBlock{
			Name:  p.customer.DisplayName(),
			Lines: p.customer.MailingAddress().Lines(),
		},
		Factor: templatecatalog.AddressBlock{
			Name:  p.factor.Name,
			Lines: p.factor.MailingAddress().Lines(),
		},
		RemitTo: templatecatalog.AddressBlock{
			Name:  p.factor.Name,
			Lines: p.factor.RemitLines(),
		},
		LogoDataURI: p.logo,
		Note:        p.noa.Notes,
	}
}

func letterEmailContext(p *preparedLetter, now int64) *factoringtemplate.FactoringNoaEmailContext {
	effective := now
	if p.noa.SentAt != nil {
		effective = *p.noa.SentAt
	}
	return &factoringtemplate.FactoringNoaEmailContext{
		CustomerName: p.customer.DisplayName(),
		CompanyName:  p.org.Name,
		FactorName:   p.factor.Name,
		RemitTo: templatecatalog.AddressBlock{
			Name:  p.factor.Name,
			Lines: p.factor.RemitLines(),
		},
		EffectiveDate:  timeutils.FormatUnixDateIn(effective, p.org.Timezone),
		Note:           p.noa.Notes,
		AttachmentName: p.fileName,
	}
}
