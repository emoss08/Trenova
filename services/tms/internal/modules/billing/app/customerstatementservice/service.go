// Package customerstatementservice renders a customer's statement of account
// as a PDF and emails it: by hand from the statement page, or each month for
// the month just closed when billing control has monthly statements on.
package customerstatementservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/app/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerstatement"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxNoteLength = 2000
	// futureSlack lets a statement be dated to the end of today wherever the
	// person drawing it up is.
	futureSlack       = 36 * 60 * 60
	referenceType     = "CustomerStatement"
	pdfContentType    = "application/pdf"
	maxFileNameLength = 120
	fileNameFallback  = "Statement.pdf"
)

// statementSource is the account statement the PDF prints.
type statementSource interface {
	GetCustomerStatement(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
		startDate, asOfDate int64,
	) (*customerledgerports.ARCustomerStatement, error)
}

type Params struct {
	fx.In

	DB              kernelports.Transactor
	Repo            customerstatementports.Repository
	Receivables     *accountsreceivableservice.Service
	Customers       customerports.CustomerRepository
	Collections     collectionports.Repository
	Organizations   tenantports.OrganizationRepository
	Users           tenantports.UserRepository
	BillingControls tenantports.BillingControlRepository
	EmailService    emailports.EmailService
	Templates       documenttemplateports.DocumentTemplateResolver
	Inliner         documenttemplateports.AssetInliner
	Factors         factoringcompanyports.FactoringCompanyRepository
	Logger          *zap.Logger
}

type Service struct {
	db              kernelports.Transactor
	repo            customerstatementports.Repository
	statements      statementSource
	customers       customerports.CustomerRepository
	collections     collectionports.Repository
	organizations   tenantports.OrganizationRepository
	users           tenantports.UserRepository
	billingControls tenantports.BillingControlRepository
	emailService    emailports.EmailService
	templates       documenttemplateports.DocumentTemplateResolver
	inliner         documenttemplateports.AssetInliner
	factors         factoringcompanyports.FactoringCompanyRepository
	l               *zap.Logger
}

var _ customerstatementports.Service = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		db:              p.DB,
		repo:            p.Repo,
		statements:      p.Receivables,
		customers:       p.Customers,
		collections:     p.Collections,
		organizations:   p.Organizations,
		users:           p.Users,
		billingControls: p.BillingControls,
		emailService:    p.EmailService,
		templates:       p.Templates,
		inliner:         p.Inliner,
		factors:         p.Factors,
		l:               p.Logger.Named("service.customer-statement"),
	}
}

// issuer is what every statement an organization sends shares.
type issuer struct {
	tenant     pagination.TenantInfo
	org        *tenant.Organization
	control    *billingcontrol.BillingControl
	logo       template.URL
	activityBy pulid.ID
}

// prepared is one customer's statement, drawn up and rendered.
type prepared struct {
	customer  *customer.Customer
	statement *customerledgerports.ARCustomerStatement
	period    customerstatement.Period
	currency  string
	pastDue   int64
	fileName  string
	document  *documenttemplateports.RenderedDocument
	// factor is who the customer's invoices are assigned to, nil when they
	// are not factored.
	factor *factoringcompany.FactoringCompany
}

// Render draws up a customer's statement as a PDF.
func (s *Service) Render(
	ctx context.Context,
	req *customerstatementports.StatementRequest,
) (*customerstatementports.RenderedStatement, error) {
	period, err := periodOf(req, timeutils.NowUnix())
	if err != nil {
		return nil, err
	}
	cus, err := s.loadCustomer(ctx, req.TenantInfo, req.CustomerID)
	if err != nil {
		return nil, err
	}
	from, err := s.issuerFor(ctx, req.TenantInfo, pulid.Nil)
	if err != nil {
		return nil, err
	}
	statement, err := s.prepare(ctx, from, cus, period, "")
	if err != nil {
		return nil, err
	}
	return &customerstatementports.RenderedStatement{
		FileName: statement.fileName,
		PDF:      statement.document.PDF,
	}, nil
}

// Send emails a customer its statement with the PDF attached, whether or not
// the customer takes the monthly statement: a person asked for this one.
func (s *Service) Send(
	ctx context.Context,
	req *customerstatementports.SendRequest,
	actor *permissionports.RequestActor,
) (*customerstatementports.SendResult, error) {
	manual, err := s.prepareSend(ctx, req, actor, true)
	if err != nil {
		return nil, err
	}
	statement, err := s.prepare(ctx, manual.from, manual.customer, manual.period, manual.note)
	if err != nil {
		return nil, err
	}
	if err = s.deliver(ctx, manual.from, statement, manual.delivery()); err != nil {
		return nil, err
	}
	return &customerstatementports.SendResult{
		SentTo: manual.recipients.To,
		EndingBalance: money.FormatMinor(
			statement.statement.EndingBalanceMinor,
			statement.currency,
		),
		FileName: statement.fileName,
	}, nil
}

func (s *Service) PreviewSend(
	ctx context.Context,
	req *customerstatementports.SendRequest,
	actor *permissionports.RequestActor,
) (*customerstatementports.SendPreview, error) {
	manual, err := s.prepareSend(ctx, req, actor, false)
	if err != nil {
		return nil, err
	}
	statement, err := s.draft(ctx, manual.from, manual.customer, manual.period)
	if err != nil {
		return nil, err
	}
	rendered, err := s.renderEmail(ctx, manual.from, statement, manual.delivery())
	if err != nil {
		return nil, err
	}
	return &customerstatementports.SendPreview{
		CustomerID:         manual.customer.ID,
		CustomerName:       manual.customer.DisplayName(),
		PeriodStart:        manual.period.Start,
		PeriodEnd:          manual.period.End,
		CurrencyCode:       statement.currency,
		EndingBalanceMinor: statement.statement.EndingBalanceMinor,
		PastDueMinor:       statement.pastDue,
		OpenItemCount:      len(statement.statement.OpenItems),
		FileName:           statement.fileName,
		Email: &emailports.PlannedEmail{
			To:                manual.recipients.To,
			CC:                manual.recipients.CC,
			BCC:               manual.recipients.BCC,
			Subject:           rendered.Subject,
			Text:              rendered.Text,
			Attachments:       []string{statement.fileName},
			TemplateVersionID: pulid.ConvertFromPtr(rendered.VersionID),
		},
	}, nil
}

type manualSend struct {
	from       *issuer
	customer   *customer.Customer
	period     customerstatement.Period
	recipients customer.EmailRecipients
	note       string
	sentBy     pulid.ID
	sentAt     int64
}

func (m *manualSend) delivery() *delivery {
	return &delivery{
		recipients: m.recipients,
		note:       m.note,
		sentBy:     m.sentBy,
		sentAt:     m.sentAt,
		key:        manualKey(m.customer.ID, m.period, m.sentBy, m.note, m.sentAt),
	}
}

func (s *Service) prepareSend(
	ctx context.Context,
	req *customerstatementports.SendRequest,
	actor *permissionports.RequestActor,
	inlineLogo bool,
) (*manualSend, error) {
	note := strings.TrimSpace(req.Note)
	if len(note) > maxNoteLength {
		return nil, errortypes.NewValidationError(
			"note", errortypes.ErrInvalid, "Keep the note under 2,000 characters",
		)
	}
	now := timeutils.NowUnix()
	period, err := periodOf(&req.StatementRequest, now)
	if err != nil {
		return nil, err
	}
	cus, err := s.loadCustomer(ctx, req.TenantInfo, req.CustomerID)
	if err != nil {
		return nil, err
	}
	recipients := cus.EmailProfile.Recipients()
	if len(recipients.To) == 0 {
		return nil, errortypes.NewBusinessError(
			"{0} has no invoice email recipients. Add one to the customer's email profile to send statements.",
			cus.DisplayName(),
		)
	}

	actorID := pulid.Nil
	if actor != nil {
		actorID = actor.UserID
	}
	from, err := s.issuer(ctx, req.TenantInfo, actorID, inlineLogo)
	if err != nil {
		return nil, err
	}
	return &manualSend{
		from:       from,
		customer:   cus,
		period:     period,
		recipients: recipients,
		note:       note,
		sentBy:     actorID,
		sentAt:     now,
	}, nil
}

// SendMonthly sends one organization the statements of the month just
// closed, once its statement day has come. A customer is skipped when it
// opted out, when it already has this month's statement, or when its
// account was settled and quiet all month. Customers with nobody to email
// are reported back so someone can add a recipient.
func (s *Service) SendMonthly(
	ctx context.Context,
	control *billingcontrol.BillingControl,
	now int64,
) (*customerstatementports.SweepResult, error) {
	result := &customerstatementports.SweepResult{NoRecipients: []string{}}
	if control == nil || !control.MonthlyStatementsEnabled {
		return result, nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: control.OrganizationID, BuID: control.BusinessUnitID}
	from, err := s.issuerFor(ctx, tenantInfo, pulid.Nil)
	if err != nil {
		return nil, err
	}
	period, due := customerstatement.MonthlyPeriod(
		now,
		timeutils.LoadLocation(from.org.Timezone),
		int(control.MonthlyStatementDay),
	)
	if !due {
		return result, nil
	}

	candidates, err := s.repo.ListStatementCustomers(
		ctx,
		customerstatementports.ListStatementCustomersRequest{
			TenantInfo: tenantInfo,
			Period:     period,
		},
	)
	if err != nil || len(candidates) == 0 {
		return result, err
	}
	sent, err := s.repo.ListMonthlySent(ctx, customerstatementports.ListMonthlySentRequest{
		TenantInfo:  tenantInfo,
		PeriodEnd:   period.End,
		CustomerIDs: candidates,
	})
	if err != nil {
		return nil, err
	}
	pending := make([]pulid.ID, 0, len(candidates))
	for _, id := range candidates {
		if sent[id] {
			result.AlreadySent++
			continue
		}
		pending = append(pending, id)
	}
	if len(pending) == 0 {
		return result, nil
	}
	customers, err := s.loadCustomers(ctx, tenantInfo, pending)
	if err != nil {
		return nil, err
	}

	for _, id := range pending {
		cus := customers[id]
		if cus == nil || (cus.BillingProfile != nil && !cus.BillingProfile.SendStatements) {
			continue
		}
		result.Customers++
		recipients := cus.EmailProfile.Recipients()
		if len(recipients.To) == 0 {
			result.NoRecipients = append(result.NoRecipients, cus.DisplayName())
			continue
		}
		if sendErr := s.sendMonthly(ctx, from, cus, period, recipients, now); sendErr != nil {
			result.Failed++
			s.l.Warn("failed to send a monthly statement",
				zap.String("customerId", id.String()), zap.Error(sendErr))
			continue
		}
		result.Sent++
	}
	return result, nil
}

func (s *Service) sendMonthly(
	ctx context.Context,
	from *issuer,
	cus *customer.Customer,
	period customerstatement.Period,
	recipients customer.EmailRecipients,
	now int64,
) error {
	statement, err := s.prepare(ctx, from, cus, period, "")
	if err != nil {
		return err
	}
	err = s.deliver(ctx, from, statement, &delivery{
		recipients: recipients,
		monthly:    true,
		sentAt:     now,
		key: "customer-statement-monthly-" + cus.ID.String() + "-" + strconv.FormatInt(
			period.End,
			10,
		),
	})
	return err
}

// ListSent is the statements a customer has been sent, newest first.
func (s *Service) ListSent(
	ctx context.Context,
	req customerstatementports.ListSentRequest,
) ([]*customerstatementports.SentStatement, error) {
	return s.repo.ListSent(ctx, req)
}

// prepare draws up the statement and renders its PDF.
func (s *Service) prepare(
	ctx context.Context,
	from *issuer,
	cus *customer.Customer,
	period customerstatement.Period,
	note string,
) (*prepared, error) {
	out, err := s.draft(ctx, from, cus, period)
	if err != nil {
		return nil, err
	}
	out.document, err = s.templates.RenderDocument(
		ctx,
		&documenttemplateports.RenderDocumentRequest{
			TenantInfo:  from.tenant,
			Kind:        documenttemplate.KindCustomerStatementPDF,
			CustomerID:  &cus.ID,
			Data:        pdfContext(from, out, note),
			ReferenceID: cus.ID,
			UserID:      from.tenant.UserID,
			Title:       strings.TrimSuffix(out.fileName, ".pdf"),
		},
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) draft(
	ctx context.Context,
	from *issuer,
	cus *customer.Customer,
	period customerstatement.Period,
) (*prepared, error) {
	statement, err := s.statements.GetCustomerStatement(
		ctx,
		from.tenant,
		cus.ID,
		period.Start,
		period.End,
	)
	if err != nil {
		return nil, err
	}
	out := &prepared{
		customer:  cus,
		statement: statement,
		period:    period,
		currency:  cus.BillingCurrencyCode(),
		pastDue:   pastDueOf(statement.Aging),
	}
	if out.factor, err = s.factorOf(ctx, from.tenant, cus); err != nil {
		return nil, err
	}
	statementDate := timeutils.FormatUnixDateIn(period.End, from.org.Timezone)
	out.fileName = fileutils.SanitizeDisplayFilename(
		"Statement-"+customerLabel(cus)+"-"+statementDate+".pdf",
		fileNameFallback,
		maxFileNameLength,
	)
	return out, nil
}

type delivery struct {
	recipients customer.EmailRecipients
	note       string
	monthly    bool
	sentBy     pulid.ID
	sentAt     int64
	key        string
}

// deliver emails the statement and records it. The idempotency key makes a
// retried send replay the queued message rather than send it again.
func (s *Service) deliver(
	ctx context.Context,
	from *issuer,
	statement *prepared,
	d *delivery,
) error {
	rendered, err := s.renderEmail(ctx, from, statement, d)
	if err != nil {
		return err
	}
	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo: from.tenant,
		Purpose:    email.PurposeBilling,
		To:         d.recipients.To,
		CC:         d.recipients.CC,
		BCC:        d.recipients.BCC,
		Subject:    rendered.Subject,
		HTML:       rendered.HTML,
		Text:       rendered.Text,
		Attachments: []emailports.EmailAttachment{{
			FileName:    statement.fileName,
			ContentType: pdfContentType,
			Content:     statement.document.PDF,
			SizeBytes:   int64(len(statement.document.PDF)),
		}},
		IdempotencyKey: d.key,
	}); err != nil {
		return err
	}

	sentTo := strings.Join(d.recipients.To, ", ")
	record := &customerstatement.Statement{
		OrganizationID:     from.tenant.OrgID,
		BusinessUnitID:     from.tenant.BuID,
		CustomerID:         statement.customer.ID,
		PeriodStart:        statement.period.Start,
		PeriodEnd:          statement.period.End,
		EndingBalanceMinor: statement.statement.EndingBalanceMinor,
		PastDueMinor:       statement.pastDue,
		CurrencyCode:       statement.currency,
		SentTo:             sentTo,
		Monthly:            d.monthly,
		SentByID:           d.sentBy,
		SentAt:             d.sentAt,
	}
	inserted := false
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		if inserted, txErr = s.repo.Insert(txCtx, record); txErr != nil || !inserted {
			return txErr
		}
		return s.logActivity(txCtx, from, statement, d, sentTo)
	})
	if err != nil {
		return err
	}
	if inserted {
		s.recordDocument(ctx, from, statement, record, d.sentBy)
	}
	return nil
}

func (s *Service) renderEmail(
	ctx context.Context,
	from *issuer,
	statement *prepared,
	d *delivery,
) (*documenttemplateports.RenderedMessage, error) {
	return s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:  from.tenant,
		Kind:        documenttemplate.KindCustomerStatementEmail,
		CustomerID:  &statement.customer.ID,
		Data:        emailContext(from, statement, d),
		ReferenceID: statement.customer.ID,
		UserID:      d.sentBy,
	})
}

// logActivity writes the statement into the customer's collections history,
// so a collector sees what the customer was sent.
func (s *Service) logActivity(
	ctx context.Context,
	from *issuer,
	statement *prepared,
	d *delivery,
	sentTo string,
) error {
	createdBy := d.sentBy
	if createdBy.IsNil() {
		createdBy = from.activityBy
	}
	if createdBy.IsNil() {
		return nil
	}
	label := "Statement"
	if d.monthly {
		label = "Monthly statement"
	}
	summary := label + " as of " +
		timeutils.FormatUnixDateIn(statement.period.End, from.org.Timezone) +
		" sent to " + sentTo + " (" +
		money.FormatMinor(statement.statement.EndingBalanceMinor, statement.currency) + " due"
	if statement.pastDue > 0 {
		summary += ", " + money.FormatMinor(statement.pastDue, statement.currency) + " past due"
	}
	summary += ")"
	if d.note != "" {
		summary += "\n" + d.note
	}
	return s.collections.InsertActivity(ctx, &collection.Activity{
		OrganizationID: from.tenant.OrgID,
		BusinessUnitID: from.tenant.BuID,
		CustomerID:     statement.customer.ID,
		Kind:           collection.ActivityEmail,
		Summary:        summary,
		CurrencyCode:   statement.currency,
		CreatedByID:    createdBy,
	})
}

// recordDocument logs the sent PDF against the template version that drew
// it. The email is already queued, so a failure here is a warning.
func (s *Service) recordDocument(
	ctx context.Context,
	from *issuer,
	statement *prepared,
	record *customerstatement.Statement,
	userID pulid.ID,
) {
	if _, err := s.templates.RecordGeneratedDocument(
		ctx,
		&documenttemplateports.RecordGeneratedDocumentRequest{
			TenantInfo:    from.tenant,
			Kind:          documenttemplate.KindCustomerStatementPDF,
			Rendered:      statement.document,
			ReferenceType: referenceType,
			ReferenceID:   record.ID,
			FileName:      statement.fileName,
			FileSize:      int64(len(statement.document.PDF)),
			UserID:        userID,
		},
	); err != nil {
		s.l.Warn("could not record the generated customer statement",
			zap.String("statementId", record.ID.String()), zap.Error(err))
	}
}

// factorOf is the factor the customer's invoices are assigned to now.
func (s *Service) factorOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cus *customer.Customer,
) (*factoringcompany.FactoringCompany, error) {
	if s.factors == nil || cus.BillingProfile == nil ||
		cus.BillingProfile.FactoringCompanyID == nil ||
		cus.BillingProfile.FactoringCompanyID.IsNil() {
		return nil, nil //nolint:nilnil // the customer has no factor
	}
	factor, err := s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:           *cus.BillingProfile.FactoringCompanyID,
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
	if err != nil {
		return nil, err
	}
	return factor, nil
}

func (s *Service) issuerFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	actorID pulid.ID,
) (*issuer, error) {
	return s.issuer(ctx, tenantInfo, actorID, true)
}

func (s *Service) issuer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	actorID pulid.ID,
	inlineLogo bool,
) (*issuer, error) {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
	if err != nil {
		return nil, err
	}
	control, err := s.billingControls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	var logo template.URL
	if inlineLogo {
		logo, err = documenttemplateports.ResolveLogoDataURI(ctx, s.inliner, org.LogoURL)
		if err != nil {
			// A statement without a logo still says what is owed; one that fails
			// to send because a logo moved does not.
			s.l.Warn("could not inline the organization logo on a statement", zap.Error(err))
			logo = ""
		}
	}
	from := &issuer{tenant: tenantInfo, org: org, control: control, logo: logo, activityBy: actorID}
	if from.activityBy.IsNil() {
		system, sysErr := s.users.GetSystemUser(ctx, "id")
		if sysErr != nil {
			return nil, sysErr
		}
		from.activityBy = system.ID
	}
	return from, nil
}

func (s *Service) loadCustomer(
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
	customers, err := s.loadCustomers(ctx, tenantInfo, []pulid.ID{id})
	if err != nil {
		return nil, err
	}
	cus := customers[id]
	if cus == nil {
		return nil, errortypes.NewNotFoundError("Customer not found")
	}
	return cus, nil
}

func (s *Service) loadCustomers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]*customer.Customer, error) {
	found, err := s.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: ids,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeState:          true,
			IncludeBillingProfile: true,
			IncludeEmailProfile:   true,
		},
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*customer.Customer, len(found))
	for _, cus := range found {
		byID[cus.ID] = cus
	}
	return byID, nil
}

// periodOf checks the span a person asked for: today when no date is given,
// never ending in the future, and never starting after it ends.
func periodOf(
	req *customerstatementports.StatementRequest,
	now int64,
) (customerstatement.Period, error) {
	period := customerstatement.Period{Start: req.StartDate, End: req.AsOfDate}
	if period.End == 0 {
		period.End = now
	}
	switch {
	case period.Start < 0 || period.End < 0:
		return period, errortypes.NewValidationError(
			"startDate", errortypes.ErrInvalid, "Statement dates cannot be negative",
		)
	case period.End > now+futureSlack:
		return period, errortypes.NewValidationError(
			"asOfDate", errortypes.ErrInvalid, "A statement cannot be dated in the future",
		)
	case period.Start > period.End:
		return period, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"Start the statement on or before its statement date",
		)
	}
	return period, nil
}

func pastDueOf(aging customerledgerports.ARAgingBucketTotals) int64 {
	return aging.Days1To30Minor + aging.Days31To60Minor + aging.Days61To90Minor + aging.DaysOver90Minor
}

func customerLabel(cus *customer.Customer) string {
	if code := strings.TrimSpace(cus.Code); code != "" {
		return code
	}
	return cus.DisplayName()
}

func manualKey(
	customerID pulid.ID,
	period customerstatement.Period,
	actorID pulid.ID,
	note string,
	now int64,
) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		customerID.String(),
		strconv.FormatInt(period.Start, 10),
		strconv.FormatInt(period.End, 10),
		actorID.String(),
		note,
		strconv.FormatInt(now/int64(time.Minute.Seconds()), 10),
	}, "|")))
	return "customer-statement-" + hex.EncodeToString(sum[:16])
}
