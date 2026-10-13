// Package paymentreminderservice emails customers about their overdue
// invoices: on the days past due billing control schedules, and by hand from
// collections. One reminder covers every overdue invoice a customer has in a
// currency, so a customer with five late invoices receives one message.
package paymentreminderservice

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/paymentreminder"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxNoteLength = 2000

type Params struct {
	fx.In

	DB              kernelports.DBConnection
	Repo            paymentreminderports.Repository
	Customers       customerports.CustomerRepository
	Collections     collectionports.Repository
	Organizations   tenantports.OrganizationRepository
	Users           tenantports.UserRepository
	BillingControls tenantports.BillingControlRepository
	EmailService    emailports.EmailService
	Templates       documenttemplateports.DocumentTemplateResolver
	Logger          *zap.Logger
}

type Service struct {
	db              kernelports.DBConnection
	repo            paymentreminderports.Repository
	customers       customerports.CustomerRepository
	collections     collectionports.Repository
	organizations   tenantports.OrganizationRepository
	users           tenantports.UserRepository
	billingControls tenantports.BillingControlRepository
	emailService    emailports.EmailService
	templates       documenttemplateports.DocumentTemplateResolver
	l               *zap.Logger
}

var _ paymentreminderports.Service = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		db:              p.DB,
		repo:            p.Repo,
		customers:       p.Customers,
		collections:     p.Collections,
		organizations:   p.Organizations,
		users:           p.Users,
		billingControls: p.BillingControls,
		emailService:    p.EmailService,
		templates:       p.Templates,
		l:               p.Logger.Named("service.payment-reminder"),
	}
}

// reminder is one email to one customer about its overdue invoices in one
// currency, and the scheduled steps it answers for.
type reminder struct {
	customer *customer.Customer
	currency string
	// listed is every overdue invoice the email names.
	listed []*paymentreminderports.OverdueInvoice
	// steps is the scheduled step each due invoice reached; empty for a
	// reminder sent by hand.
	steps  map[pulid.ID]int
	number int
	final  bool
	note   string
	manual bool
	sentBy pulid.ID
	asOf   int64
}

// sender is what every reminder for one organization shares.
type sender struct {
	tenant      pagination.TenantInfo
	companyName string
	timezone    string
	remittance  string
	activityBy  pulid.ID
}

// SendDue sends the reminders one organization's schedule says are due. A
// customer is skipped when it opted out, when it is waiting on an open promise
// to pay, or when nothing has reached a new step. Customers with nobody to
// email are reported back so someone can add a recipient.
func (s *Service) SendDue(
	ctx context.Context,
	control *tenant.BillingControl,
	now int64,
) (*paymentreminderports.SweepResult, error) {
	result := new(paymentreminderports.SweepResult)
	if control == nil || !control.PaymentRemindersEnabled || len(control.PaymentReminderDays) == 0 {
		return result, nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: control.OrganizationID, BuID: control.BusinessUnitID}

	overdue, err := s.repo.ListOverdueInvoices(ctx, paymentreminderports.ListOverdueInvoicesRequest{
		TenantInfo: tenantInfo,
		AsOf:       now,
	})
	if err != nil {
		return nil, err
	}
	byCustomer := groupByCustomer(withoutDisputed(overdue))
	due := make([]pulid.ID, 0, len(byCustomer))
	for customerID, invoices := range byCustomer {
		if hasNewStep(invoices, control.PaymentReminderDays, now) {
			due = append(due, customerID)
		}
	}
	if len(due) == 0 {
		return result, nil
	}
	slices.SortFunc(due, func(a, b pulid.ID) int { return strings.Compare(a.String(), b.String()) })

	awaiting, err := s.collections.ListAwaitingPromise(ctx, collectionports.AwaitingPromiseRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: due,
		AsOf:        now,
	})
	if err != nil {
		return nil, err
	}
	customers, err := s.loadCustomers(ctx, tenantInfo, due)
	if err != nil {
		return nil, err
	}
	from, err := s.senderFor(ctx, tenantInfo, control, pulid.Nil)
	if err != nil {
		return nil, err
	}

	for _, customerID := range due {
		cus := customers[customerID]
		switch {
		case cus == nil:
			continue
		case cus.BillingProfile != nil && !cus.BillingProfile.SendPaymentReminders:
			continue
		case awaiting[customerID]:
			result.WaitedOnPromise++
			continue
		}
		result.Customers++
		for _, rem := range scheduledReminders(cus, byCustomer[customerID], control.PaymentReminderDays, now) {
			sent, sendErr := s.send(ctx, from, rem)
			switch {
			case sendErr != nil:
				result.Failed++
				s.l.Warn("failed to send a payment reminder",
					zap.String("customerId", customerID.String()), zap.Error(sendErr))
			case len(sent) == 0:
				result.NoRecipients = appendUnique(result.NoRecipients, displayName(cus))
			default:
				result.Sent++
			}
		}
	}
	return result, nil
}

// SendNow sends a customer a reminder of every overdue invoice, one per
// currency, whatever the schedule says.
func (s *Service) SendNow(
	ctx context.Context,
	req *paymentreminderports.SendNowRequest,
	actor *permissionports.RequestActor,
) ([]*paymentreminderports.SendNowResult, error) {
	note := strings.TrimSpace(req.Note)
	if len(note) > maxNoteLength {
		return nil, errortypes.NewValidationError(
			"note", errortypes.ErrInvalid, "Keep the note under 2,000 characters",
		)
	}
	customers, err := s.loadCustomers(ctx, req.TenantInfo, []pulid.ID{req.CustomerID})
	if err != nil {
		return nil, err
	}
	cus := customers[req.CustomerID]
	if cus == nil {
		return nil, errortypes.NewNotFoundError("Customer not found")
	}
	if cus.BillingProfile != nil && !cus.BillingProfile.SendPaymentReminders {
		return nil, errortypes.NewBusinessError(
			"{0} is set not to receive payment reminders. Turn reminders on in the customer's billing profile to send one.",
			displayName(cus),
		)
	}

	now := timeutils.NowUnix()
	overdue, err := s.repo.ListOverdueInvoices(ctx, paymentreminderports.ListOverdueInvoicesRequest{
		TenantInfo:  req.TenantInfo,
		AsOf:        now,
		CustomerIDs: []pulid.ID{req.CustomerID},
	})
	if err != nil {
		return nil, err
	}
	invoices := withoutDisputed(overdue)
	if len(invoices) == 0 {
		return nil, errortypes.NewBusinessError(
			"{0} has no overdue invoice that is not in dispute, so there is nothing to remind them of.",
			displayName(cus),
		)
	}

	control, err := s.billingControls.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	actorID := pulid.Nil
	if actor != nil {
		actorID = actor.UserID
	}
	from, err := s.senderFor(ctx, req.TenantInfo, control, actorID)
	if err != nil {
		return nil, err
	}

	results := make([]*paymentreminderports.SendNowResult, 0, 1)
	for currency, listed := range byCurrency(invoices) {
		rem := &reminder{
			customer: cus,
			currency: currency,
			listed:   listed,
			steps:    map[pulid.ID]int{},
			note:     note,
			manual:   true,
			sentBy:   actorID,
			asOf:     now,
		}
		sent, sendErr := s.send(ctx, from, rem)
		if sendErr != nil {
			return nil, sendErr
		}
		if len(sent) == 0 {
			return nil, errortypes.NewBusinessError(
				"{0} has no invoice email recipients. Add one to the customer's email profile to send reminders.",
				displayName(cus),
			)
		}
		results = append(results, &paymentreminderports.SendNowResult{
			SentTo:         sent,
			InvoiceNumbers: invoiceNumbers(listed),
			TotalOverdue:   money.FormatMinor(openTotal(listed), currency),
		})
	}
	slices.SortFunc(results, func(a, b *paymentreminderports.SendNowResult) int {
		return strings.Compare(a.TotalOverdue, b.TotalOverdue)
	})
	return results, nil
}

// send queues the reminder and records what it covered. The email's
// idempotency key is drawn from what it says, so a retried run whose records
// failed to save replays the queued message rather than sending it again.
// It returns who the reminder went to; nobody when the customer has no
// recipients.
func (s *Service) send(ctx context.Context, from *sender, rem *reminder) ([]string, error) {
	recipients := recipientsOf(rem.customer)
	if len(recipients.To) == 0 {
		return nil, nil
	}

	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:  from.tenant,
		Kind:        documenttemplate.KindPaymentReminderEmail,
		CustomerID:  &rem.customer.ID,
		Data:        templateContext(from, rem),
		ReferenceID: rem.customer.ID,
		UserID:      rem.sentBy,
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     from.tenant,
		Purpose:        email.PurposeBilling,
		To:             recipients.To,
		CC:             recipients.CC,
		BCC:            recipients.BCC,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: idempotencyKey(rem),
	}); err != nil {
		return nil, err
	}

	sentTo := strings.Join(recipients.To, ", ")
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if txErr := s.repo.InsertReminders(txCtx, records(from, rem, sentTo)); txErr != nil {
			return txErr
		}
		return s.logActivity(txCtx, from, rem, sentTo)
	})
	if err != nil {
		return nil, err
	}
	return recipients.To, nil
}

// logActivity writes the reminder into the customer's collections history,
// so a collector sees what the customer was already told.
func (s *Service) logActivity(ctx context.Context, from *sender, rem *reminder, sentTo string) error {
	createdBy := rem.sentBy
	if createdBy.IsNil() {
		createdBy = from.activityBy
	}
	if createdBy.IsNil() {
		return nil
	}
	label := "Payment reminder"
	switch {
	case rem.manual:
		label = "Payment reminder sent by hand"
	case rem.final:
		label = "Final payment reminder"
	case rem.number > 0:
		label = "Payment reminder " + strconv.Itoa(rem.number)
	}
	summary := label + " sent to " + sentTo + " for " +
		strings.Join(invoiceNumbers(rem.listed), ", ") + " (" +
		money.FormatMinor(openTotal(rem.listed), rem.currency) + " past due)"
	if rem.note != "" {
		summary += "\n" + rem.note
	}
	return s.collections.InsertActivity(ctx, &collection.Activity{
		OrganizationID: from.tenant.OrgID,
		BusinessUnitID: from.tenant.BuID,
		CustomerID:     rem.customer.ID,
		Kind:           collection.ActivityEmail,
		Summary:        summary,
		CurrencyCode:   rem.currency,
		CreatedByID:    createdBy,
	})
}

func (s *Service) senderFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *tenant.BillingControl,
	actorID pulid.ID,
) (*sender, error) {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo})
	if err != nil {
		return nil, err
	}
	from := &sender{tenant: tenantInfo, companyName: org.Name, timezone: org.Timezone, activityBy: actorID}
	if control != nil {
		from.remittance = strings.TrimSpace(control.DefaultInvoiceTerms)
	}
	if from.activityBy.IsNil() {
		system, sysErr := s.users.GetSystemUser(ctx, "id")
		if sysErr != nil {
			return nil, sysErr
		}
		from.activityBy = system.ID
	}
	return from, nil
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

// scheduledReminders builds a customer's reminders for one sweep: one per
// currency that has an invoice newly reaching a step, listing every overdue
// invoice in that currency.
func scheduledReminders(
	cus *customer.Customer,
	invoices []*paymentreminderports.OverdueInvoice,
	schedule []int32,
	now int64,
) []*reminder {
	reminders := make([]*reminder, 0, 1)
	currencies := byCurrency(invoices)
	codes := make([]string, 0, len(currencies))
	for code := range currencies {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	for _, code := range codes {
		listed := currencies[code]
		rem := &reminder{customer: cus, currency: code, listed: listed, steps: map[pulid.ID]int{}, asOf: now}
		for _, inv := range listed {
			step := paymentreminder.StepReached(paymentreminder.DaysPastDue(inv.DueDate, now), schedule)
			if step <= inv.LastStep {
				continue
			}
			rem.steps[inv.InvoiceID] = step
			rem.number = max(rem.number, step)
			if paymentreminder.IsFinal(step, schedule) {
				rem.final = true
			}
		}
		if len(rem.steps) > 0 {
			reminders = append(reminders, rem)
		}
	}
	return reminders
}

func hasNewStep(invoices []*paymentreminderports.OverdueInvoice, schedule []int32, now int64) bool {
	for _, inv := range invoices {
		if paymentreminder.StepReached(paymentreminder.DaysPastDue(inv.DueDate, now), schedule) > inv.LastStep {
			return true
		}
	}
	return false
}

// records are the rows a sent reminder leaves: one per invoice newly reaching
// a scheduled step, or one per listed invoice for a reminder sent by hand.
func records(from *sender, rem *reminder, sentTo string) []*paymentreminder.Reminder {
	rows := make([]*paymentreminder.Reminder, 0, len(rem.listed))
	for _, inv := range rem.listed {
		step, scheduled := rem.steps[inv.InvoiceID]
		if !rem.manual && !scheduled {
			continue
		}
		if rem.manual {
			step = paymentreminder.ManualStep
		}
		rows = append(rows, &paymentreminder.Reminder{
			OrganizationID:  from.tenant.OrgID,
			BusinessUnitID:  from.tenant.BuID,
			InvoiceID:       inv.InvoiceID,
			CustomerID:      rem.customer.ID,
			Step:            int16(step), //nolint:gosec // bounded by MaxPaymentReminderSteps
			DaysPastDue:     paymentreminder.DaysPastDue(inv.DueDate, rem.asOf),
			OpenAmountMinor: inv.OpenMinor,
			CurrencyCode:    rem.currency,
			SentTo:          sentTo,
			Manual:          rem.manual,
			SentByID:        rem.sentBy,
			SentAt:          rem.asOf,
		})
	}
	return rows
}

func templateContext(from *sender, rem *reminder) *documenttemplate.PaymentReminderEmailContext {
	ctx := &documenttemplate.PaymentReminderEmailContext{
		CustomerName:           rem.customer.Name,
		CustomerCode:           rem.customer.Code,
		CompanyName:            from.companyName,
		ReminderNumber:         rem.number,
		IsFinalReminder:        rem.final,
		CurrencyCode:           rem.currency,
		TotalOverdue:           money.FormatMinor(openTotal(rem.listed), rem.currency),
		RemittanceInstructions: from.remittance,
		Note:                   rem.note,
		Invoices:               make([]documenttemplate.PaymentReminderInvoice, 0, len(rem.listed)),
	}
	for _, inv := range rem.listed {
		days := paymentreminder.DaysPastDue(inv.DueDate, rem.asOf)
		ctx.OldestDaysPastDue = max(ctx.OldestDaysPastDue, days)
		ctx.Invoices = append(ctx.Invoices, documenttemplate.PaymentReminderInvoice{
			InvoiceNumber: inv.InvoiceNumber,
			InvoiceDate:   timeutils.FormatUnixDateIn(inv.InvoiceDate, from.timezone),
			DueDate:       timeutils.FormatUnixDateIn(inv.DueDate, from.timezone),
			DaysPastDue:   days,
			OpenAmount:    money.FormatMinor(inv.OpenMinor, rem.currency),
		})
	}
	return ctx
}

func idempotencyKey(rem *reminder) string {
	parts := make([]string, 0, len(rem.listed)+3)
	parts = append(parts, rem.customer.ID.String(), rem.currency)
	if rem.manual {
		parts = append(parts, "manual", strconv.FormatInt(rem.asOf, 10))
	}
	for _, inv := range rem.listed {
		parts = append(parts, inv.InvoiceID.String()+":"+strconv.Itoa(rem.steps[inv.InvoiceID]))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "payment-reminder-" + hex.EncodeToString(sum[:16])
}

type recipientList struct {
	To  []string
	CC  []string
	BCC []string
}

func recipientsOf(cus *customer.Customer) recipientList {
	if cus == nil || cus.EmailProfile == nil {
		return recipientList{}
	}
	return recipientList{
		To:  stringutils.SplitEmailList(cus.EmailProfile.ToRecipients),
		CC:  stringutils.SplitEmailList(cus.EmailProfile.CCRecipients),
		BCC: stringutils.SplitEmailList(cus.EmailProfile.BCCRecipients),
	}
}

func withoutDisputed(invoices []*paymentreminderports.OverdueInvoice) []*paymentreminderports.OverdueInvoice {
	kept := make([]*paymentreminderports.OverdueInvoice, 0, len(invoices))
	for _, inv := range invoices {
		if !inv.Disputed {
			kept = append(kept, inv)
		}
	}
	return kept
}

func groupByCustomer(
	invoices []*paymentreminderports.OverdueInvoice,
) map[pulid.ID][]*paymentreminderports.OverdueInvoice {
	grouped := make(map[pulid.ID][]*paymentreminderports.OverdueInvoice)
	for _, inv := range invoices {
		grouped[inv.CustomerID] = append(grouped[inv.CustomerID], inv)
	}
	return grouped
}

func byCurrency(
	invoices []*paymentreminderports.OverdueInvoice,
) map[string][]*paymentreminderports.OverdueInvoice {
	grouped := make(map[string][]*paymentreminderports.OverdueInvoice, 1)
	for _, inv := range invoices {
		code := money.CurrencyCode(inv.CurrencyCode)
		grouped[code] = append(grouped[code], inv)
	}
	for _, list := range grouped {
		slices.SortStableFunc(list, func(a, b *paymentreminderports.OverdueInvoice) int {
			return cmp.Or(cmp.Compare(a.DueDate, b.DueDate), strings.Compare(a.InvoiceNumber, b.InvoiceNumber))
		})
	}
	return grouped
}

func openTotal(invoices []*paymentreminderports.OverdueInvoice) int64 {
	var total int64
	for _, inv := range invoices {
		total += inv.OpenMinor
	}
	return total
}

func invoiceNumbers(invoices []*paymentreminderports.OverdueInvoice) []string {
	numbers := make([]string, 0, len(invoices))
	for _, inv := range invoices {
		numbers = append(numbers, inv.InvoiceNumber)
	}
	return numbers
}

func displayName(cus *customer.Customer) string {
	if cus.Name != "" {
		return cus.Name
	}
	return cus.Code
}

func appendUnique(list []string, value string) []string {
	if slices.Contains(list, value) {
		return list
	}
	return append(list, value)
}
