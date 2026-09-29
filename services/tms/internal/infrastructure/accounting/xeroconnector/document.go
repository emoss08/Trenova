package xeroconnector

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/shopspring/decimal"
)

var _ services.AccountingDocumentWriter = (*Connector)(nil)

const (
	providerName        = "Xero"
	appBaseURL          = "https://go.xero.com/app/"
	salesViewPath       = "/invoicing/view/"
	billsViewPath       = "/bills/view/"
	debitMemoNotePrefix = "Debit memo"
	defaultLineText     = "Charges"
	idempotencyWindow   = 6 * time.Minute
	offlineRetry        = 5 * time.Minute
	maxErrorMessage     = 2000
	moneyPlaces         = 2
	refCreditNote       = "creditNote"
)

var (
	errDocumentKind = errors.New("xero: this record type is not a document Xero holds")
	errInvalidDate  = errors.New("xero: a document date must be written as YYYY-MM-DD")
	errExternalID   = errors.New("xero: updating a document needs the id Xero gave it")
	errCreditNoteID = errors.New(
		"xero: the credit application does not name the credit note it was allocated from",
	)
)

func (c *Connector) DocumentLimits() services.AccountingDocumentLimits {
	return services.AccountingDocumentLimits{
		MaxDocNumberLength:      xero.MaxDocumentNumberLength,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       idempotencyWindow,
	}
}

func (c *Connector) DocumentURL(
	auth services.AccountingDocumentAuth,
	kind accountingsync.SyncObjectType,
	externalID string,
) string {
	code := strings.TrimSpace(auth.CompanyCode)
	id := strings.TrimSpace(externalID)
	if code == "" || id == "" {
		return ""
	}
	var path string
	switch {
	case kind.IsSalesDocument():
		path = salesViewPath
	case kind.IsBill():
		path = billsViewPath
	default:
		return ""
	}
	return appBaseURL + url.PathEscape(code) + path + url.PathEscape(id)
}

func (c *Connector) UpsertCustomer(
	ctx context.Context,
	doc *services.AccountingCustomerDocument,
) (*services.AccountingDocumentResult, error) {
	return c.upsertContact(ctx, &contactWrite{
		auth:       doc.Auth,
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

type contactWrite struct {
	auth       services.AccountingDocumentAuth
	requestID  string
	externalID string
	party      *services.AccountingPartyDraft
}

func (c *Connector) upsertContact(
	ctx context.Context,
	write *contactWrite,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(write.auth)
	if err != nil {
		return nil, err
	}
	input := contactInputOf(write.party)

	var saved *xero.Contact
	if id := strings.TrimSpace(write.externalID); id != "" {
		saved, err = client.UpdateContact(ctx, write.requestID, id, input)
	} else {
		saved, err = client.CreateContact(ctx, write.requestID, input)
	}
	if err != nil {
		return nil, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: saved.ContactID,
		DocNumber:  saved.Name,
		Refs:       map[string]string{},
	}, nil
}

type salesKind struct {
	credit bool
}

func salesKindOf(kind accountingsync.SyncObjectType) (salesKind, error) {
	switch kind {
	case accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo:
		return salesKind{}, nil
	case accountingsync.SyncObjectCreditMemo:
		return salesKind{credit: true}, nil
	case accountingsync.SyncObjectCustomer,
		accountingsync.SyncObjectCustomerPayment,
		accountingsync.SyncObjectCreditApplication,
		accountingsync.SyncObjectCarrierVendor,
		accountingsync.SyncObjectDriverVendor,
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectCarrierBillPay,
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay,
		accountingsync.SyncObjectJournalEntry,
		accountingsync.SyncObjectJournalSummary,
		accountingsync.DriftObjectGLAccount:
		return salesKind{}, errDocumentKind
	default:
		return salesKind{}, errDocumentKind
	}
}

type documentBody struct {
	contactID    string
	number       string
	reference    string
	date         time.Time
	dueDate      time.Time
	currencyCode string
	currencyRate decimal.Decimal
	lines        []xero.LineItem
}

func (b *documentBody) invoice(kind string) xero.InvoiceInput {
	return xero.InvoiceInput{
		Type:            kind,
		ContactID:       b.contactID,
		InvoiceNumber:   b.number,
		Reference:       b.reference,
		Date:            b.date,
		DueDate:         b.dueDate,
		CurrencyCode:    b.currencyCode,
		CurrencyRate:    b.currencyRate,
		Status:          xero.StatusAuthorised,
		LineAmountTypes: xero.LineAmountsNoTax,
		Lines:           b.lines,
	}
}

func (b *documentBody) creditNote(kind string) xero.CreditNoteInput {
	return xero.CreditNoteInput{
		Type:             kind,
		ContactID:        b.contactID,
		CreditNoteNumber: b.number,
		Reference:        b.reference,
		Date:             b.date,
		CurrencyCode:     b.currencyCode,
		CurrencyRate:     b.currencyRate,
		Status:           xero.StatusAuthorised,
		LineAmountTypes:  xero.LineAmountsNoTax,
		Lines:            b.lines,
	}
}

func (c *Connector) salesBody(
	ctx context.Context,
	client *xero.Client,
	doc *services.AccountingSalesDocument,
) (*documentBody, error) {
	date, err := parseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	due, err := parseDate(doc.DueDate)
	if err != nil {
		return nil, err
	}
	note := doc.PrivateNote
	if doc.Kind == accountingsync.SyncObjectDebitMemo {
		note = strings.TrimSpace(debitMemoNotePrefix + ". " + note)
	}
	body := &documentBody{
		contactID:    doc.CustomerExternalID,
		number:       doc.DocNumber,
		reference:    reference(note),
		date:         date,
		dueDate:      due,
		currencyCode: doc.CurrencyCode,
		currencyRate: doc.ExchangeRate,
		lines:        make([]xero.LineItem, 0, len(doc.Lines)),
	}
	if doc.Kind == accountingsync.SyncObjectCreditMemo {
		body.dueDate = time.Time{}
	}
	for idx := range doc.Lines {
		line := &doc.Lines[idx]
		code, codeErr := c.accountCode(
			ctx,
			client,
			line.AccountExternalID,
			"Line "+strconv.Itoa(idx+1)+" ("+lineText(line.Description)+")",
		)
		if codeErr != nil {
			return nil, codeErr
		}
		body.lines = append(body.lines, lineItem(&lineSpec{
			description: line.Description,
			quantity:    line.Quantity,
			unitPrice:   line.UnitPrice,
			amount:      line.Amount,
			accountCode: code,
		}))
	}
	return body, nil
}

type lineSpec struct {
	description string
	quantity    decimal.Decimal
	unitPrice   decimal.Decimal
	amount      decimal.Decimal
	accountCode string
}

func lineItem(spec *lineSpec) xero.LineItem {
	item := xero.LineItem{
		Description: stringutils.TruncateRunes(
			lineText(spec.description),
			xero.MaxDescriptionLength,
		),
		Quantity:    decimal.NewFromInt(1),
		UnitAmount:  spec.amount,
		LineAmount:  spec.amount,
		AccountCode: spec.accountCode,
	}
	if spec.quantity.IsPositive() && !spec.unitPrice.IsZero() &&
		spec.quantity.Mul(spec.unitPrice).Round(moneyPlaces).Equal(spec.amount.Round(moneyPlaces)) {
		item.Quantity = spec.quantity
		item.UnitAmount = spec.unitPrice
	}
	return item
}

func lineText(description string) string {
	if text := strings.TrimSpace(description); text != "" {
		return text
	}
	return defaultLineText
}

func reference(note string) string {
	return strings.TrimSpace(stringutils.OneLine(note, xero.MaxReferenceLength))
}

func (c *Connector) writeSales(
	ctx context.Context,
	client *xero.Client,
	doc *services.AccountingSalesDocument,
	externalID string,
) (id, number string, err error) {
	kind, err := salesKindOf(doc.Kind)
	if err != nil {
		return "", "", err
	}
	body, err := c.salesBody(ctx, client, doc)
	if err != nil {
		return "", "", err
	}

	if kind.credit {
		input := body.creditNote(xero.CreditNoteTypeReceivable)
		var note *xero.CreditNote
		if externalID == "" {
			note, err = client.CreateCreditNote(ctx, doc.RequestID, input)
		} else {
			note, err = client.UpdateCreditNote(ctx, doc.RequestID, externalID, input)
		}
		if err != nil {
			return "", "", err
		}
		return note.CreditNoteID, note.CreditNoteNumber, nil
	}

	input := body.invoice(xero.InvoiceTypeReceivable)
	var invoice *xero.Invoice
	if externalID == "" {
		invoice, err = client.CreateInvoice(ctx, doc.RequestID, input)
	} else {
		invoice, err = client.UpdateInvoice(ctx, doc.RequestID, externalID, input)
	}
	if err != nil {
		return "", "", err
	}
	return invoice.InvoiceID, invoice.InvoiceNumber, nil
}

func (c *Connector) CreateSalesDocument(
	ctx context.Context,
	doc *services.AccountingSalesDocument,
) (*services.AccountingDocumentResult, error) {
	kind, err := salesKindOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)

	if existing := result.Refs[accountingsync.ExternalRefDocument]; existing != "" {
		result.ExternalID = existing
	} else {
		result.ExternalID, result.DocNumber, err = c.writeSales(ctx, client, doc, "")
		if err != nil {
			return result, err
		}
		result.Refs[accountingsync.ExternalRefDocument] = result.ExternalID
	}

	if !kind.credit ||
		strings.TrimSpace(doc.ApplyToExternalID) == "" ||
		!doc.ApplyAmount.IsPositive() ||
		result.Refs[accountingsync.ExternalRefApplication] != "" {
		return result, nil
	}

	date, err := parseDate(doc.TxnDate)
	if err != nil {
		return result, err
	}
	allocation, err := client.Allocate(
		ctx,
		accountingsync.SyncStepRequestID(doc.RequestID, 1),
		result.ExternalID,
		xero.AllocationInput{
			InvoiceID: doc.ApplyToExternalID,
			Amount:    doc.ApplyAmount,
			Date:      date,
		},
	)
	if err != nil {
		return result, err
	}
	result.Refs[accountingsync.ExternalRefApplication] = allocation.AllocationID
	return result, nil
}

func (c *Connector) UpdateSalesDocument(
	ctx context.Context,
	doc *services.AccountingSalesDocument,
) (*services.AccountingDocumentResult, error) {
	if _, err := salesKindOf(doc.Kind); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalID
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}

	result := newResult(doc.Refs)
	result.ExternalID, result.DocNumber, err = c.writeSales(ctx, client, doc, id)
	if err != nil {
		return nil, err
	}
	result.Refs[accountingsync.ExternalRefDocument] = result.ExternalID
	return result, nil
}

func (c *Connector) VoidSalesDocument(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	kind, err := salesKindOf(ref.Kind)
	if err != nil {
		return nil, err
	}
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	if kind.credit {
		err = voidCreditNote(ctx, client, ref.RequestID, ref.ExternalID)
	} else {
		err = voidInvoice(ctx, client, ref.RequestID, ref.ExternalID)
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func voidInvoice(ctx context.Context, client *xero.Client, requestID, invoiceID string) error {
	found, err := client.InvoicesByID(ctx, []string{invoiceID})
	if err != nil {
		if xero.IsNotFound(err) {
			return nil
		}
		return err
	}
	if len(found) == 0 || closedStatus(found[0].Status) {
		return nil
	}
	invoice := &found[0]

	for idx := range invoice.Payments {
		if delErr := client.DeletePayment(
			ctx,
			accountingsync.SyncStepRequestID(requestID, idx+1),
			invoice.Payments[idx].PaymentID,
		); delErr != nil && !xero.IsNotFound(delErr) {
			return delErr
		}
	}
	if err = removeAllocationsTo(ctx, client, invoice); err != nil {
		return err
	}
	if _, err = client.VoidInvoice(ctx, requestID, invoice.InvoiceID); err != nil &&
		!xero.IsNotFound(err) {
		return err
	}
	return nil
}

func removeAllocationsTo(ctx context.Context, client *xero.Client, invoice *xero.Invoice) error {
	if len(invoice.CreditNotes) == 0 {
		return nil
	}
	noteIDs := make([]string, 0, len(invoice.CreditNotes))
	for idx := range invoice.CreditNotes {
		noteIDs = append(noteIDs, invoice.CreditNotes[idx].CreditNoteID)
	}
	for start := 0; start < len(noteIDs); start += xero.MaxIDsPerRead {
		notes, err := client.CreditNotesByID(
			ctx,
			noteIDs[start:min(start+xero.MaxIDsPerRead, len(noteIDs))],
		)
		if err != nil {
			return err
		}
		for idx := range notes {
			for _, allocation := range notes[idx].Allocations {
				if allocation.IsDeleted || !sameID(allocation.InvoiceID, invoice.InvoiceID) {
					continue
				}
				if delErr := client.DeleteAllocation(
					ctx,
					notes[idx].CreditNoteID,
					allocation.AllocationID,
				); delErr != nil && !xero.IsNotFound(delErr) {
					return delErr
				}
			}
		}
	}
	return nil
}

func voidCreditNote(ctx context.Context, client *xero.Client, requestID, noteID string) error {
	found, err := client.CreditNotesByID(ctx, []string{noteID})
	if err != nil {
		if xero.IsNotFound(err) {
			return nil
		}
		return err
	}
	if len(found) == 0 || closedStatus(found[0].Status) {
		return nil
	}
	note := &found[0]
	for _, allocation := range note.Allocations {
		if allocation.IsDeleted {
			continue
		}
		if delErr := client.DeleteAllocation(
			ctx,
			note.CreditNoteID,
			allocation.AllocationID,
		); delErr != nil && !xero.IsNotFound(delErr) {
			return delErr
		}
	}
	for idx := range note.Payments {
		if delErr := client.DeletePayment(
			ctx,
			accountingsync.SyncStepRequestID(requestID, idx+1),
			note.Payments[idx].PaymentID,
		); delErr != nil && !xero.IsNotFound(delErr) {
			return delErr
		}
	}
	if _, err = client.VoidCreditNote(ctx, requestID, note.CreditNoteID); err != nil &&
		!xero.IsNotFound(err) {
		return err
	}
	return nil
}

func closedStatus(status string) bool {
	return strings.EqualFold(status, xero.StatusVoided) ||
		strings.EqualFold(status, xero.StatusDeleted)
}

func (c *Connector) CreateCreditApplication(
	ctx context.Context,
	doc *services.AccountingCreditApplicationDocument,
) (*services.AccountingDocumentResult, error) {
	if !doc.Amount.IsPositive() {
		return nil, xero.ErrAmountNotPositive
	}
	date, err := parseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	allocation, err := client.Allocate(
		ctx,
		doc.RequestID,
		doc.CreditMemoExternalID,
		xero.AllocationInput{
			InvoiceID: doc.InvoiceExternalID,
			Amount:    doc.Amount,
			Date:      date,
		},
	)
	if err != nil {
		return nil, err
	}
	return allocationResult(allocation), nil
}

func allocationResult(allocation *xero.Allocation) *services.AccountingDocumentResult {
	return &services.AccountingDocumentResult{
		ExternalID: allocation.AllocationID,
		Refs: map[string]string{
			accountingsync.ExternalRefDocument: allocation.AllocationID,
			refCreditNote:                      allocation.CreditNoteID,
		},
	}
}

func (c *Connector) VoidCreditApplication(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	noteID := strings.TrimSpace(ref.Refs[refCreditNote])
	if noteID == "" {
		return nil, errCreditNoteID
	}
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	if err = client.DeleteAllocation(ctx, noteID, ref.ExternalID); err != nil &&
		!xero.IsNotFound(err) {
		return nil, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: ref.ExternalID,
		Refs:       map[string]string{},
	}, nil
}

type documentFault struct {
	matches    func(err error, apiErr *xero.APIError) bool
	category   accountingsync.SyncErrorCategory
	resolution string
}

func isConfigurationFault(err error, _ *xero.APIError) bool {
	return errors.Is(err, ErrNotConfigured) || errors.Is(err, errNoRedirect)
}

func isAuthFault(err error, _ *xero.APIError) bool {
	return xero.IsInvalidGrant(err) || xero.IsAuth(err) || xero.IsForbidden(err)
}

func isTransientFault(err error, _ *xero.APIError) bool {
	return xero.IsTransient(err) || errors.Is(err, context.DeadlineExceeded)
}

func isDuplicateName(err error, apiErr *xero.APIError) bool {
	return xero.IsDuplicateNumber(err) && validationMentions(apiErr, "contact name")
}

func isValidationFault(err error, apiErr *xero.APIError) bool {
	return isLocalValidation(err) || xero.IsValidation(err) ||
		hasStatus(apiErr, http.StatusBadRequest)
}

func ignoringAPIError(match func(error) bool) func(error, *xero.APIError) bool {
	return func(err error, _ *xero.APIError) bool { return match(err) }
}

func isMappingFaultOf(_ error, apiErr *xero.APIError) bool {
	return isMappingFault(apiErr)
}

var documentFaults = []documentFault{
	{
		matches:    isConfigurationFault,
		category:   accountingsync.SyncErrorConfiguration,
		resolution: providerName + " is not configured on this server",
	},
	{
		matches:    isAuthFault,
		category:   accountingsync.SyncErrorAuth,
		resolution: "Reconnect " + providerName + " from the accounting integration",
	},
	{matches: isTransientFault, category: accountingsync.SyncErrorTransient},
	{
		matches:  isDuplicateName,
		category: accountingsync.SyncErrorDuplicate,
		resolution: "A contact with this name already exists in " + providerName +
			". Map the record to it instead, or rename one of them",
	},
	{
		matches:  ignoringAPIError(xero.IsDuplicateNumber),
		category: accountingsync.SyncErrorDuplicate,
		resolution: "A document with this number already exists in " + providerName +
			". Skip this record if it is the same document, or renumber the one in " +
			providerName,
	},
	{
		matches:  ignoringAPIError(xero.IsLockDate),
		category: accountingsync.SyncErrorClosedPeriod,
		resolution: "The books are locked for this date in " + providerName +
			". Move the lock date there or skip this record",
	},
	{
		matches:    ignoringAPIError(xero.IsNotFound),
		category:   accountingsync.SyncErrorNotFound,
		resolution: "The linked record no longer exists in " + providerName,
	},
	{
		matches:  isMappingFaultOf,
		category: accountingsync.SyncErrorMapping,
		resolution: "A mapped " + providerName + " record is missing or archived. " +
			"Refresh the reference data and fix the mapping",
	},
	{
		matches:    isValidationFault,
		category:   accountingsync.SyncErrorValidation,
		resolution: "Correct the document in Trenova, then retry",
	},
}

func (c *Connector) ClassifyDocumentError(err error) *accountingsync.SyncError {
	if err == nil {
		return nil
	}
	var syncErr *accountingsync.SyncError
	if errors.As(err, &syncErr) {
		return syncErr
	}

	apiErr := apiErrorOf(err)
	switch {
	case xero.IsRateLimited(err):
		return rateLimited(err, apiErr)
	case xero.IsOrganisationOffline(err):
		return &accountingsync.SyncError{
			Category: accountingsync.SyncErrorTransient,
			Code:     "organisation-offline",
			Message: "The " + providerName + " organisation is offline. Retry in " +
				offlineRetry.String(),
			RetryAfter: offlineRetry,
		}
	}

	out := &accountingsync.SyncError{
		Category: accountingsync.SyncErrorTransient,
		Code:     errorCode(apiErr),
		Message:  errorMessage(err, apiErr),
	}
	for idx := range documentFaults {
		fault := &documentFaults[idx]
		if fault.matches(err, apiErr) {
			out.Category = fault.category
			out.Resolution = fault.resolution
			return out
		}
	}
	return out
}

func rateLimited(err error, apiErr *xero.APIError) *accountingsync.SyncError {
	out := &accountingsync.SyncError{
		Category: accountingsync.SyncErrorRateLimited,
		Code:     errorCode(apiErr),
		Message:  errorMessage(err, apiErr),
	}
	if apiErr != nil {
		out.RetryAfter = apiErr.RetryAfter
	}
	if apiErr == nil || apiErr.RateLimitProblem != xero.RateLimitDay {
		return out
	}
	out.Code = xero.RateLimitDay
	out.Message = providerName + " has refused further calls for this organisation today"
	if apiErr.RetryAfter > 0 {
		out.Message += "; retry after " + apiErr.RetryAfter.String()
	}
	out.Resolution = "Trenova retries when " + providerName + "'s daily allowance resets"
	return out
}

func apiErrorOf(err error) *xero.APIError {
	var apiErr *xero.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

func hasStatus(apiErr *xero.APIError, status int) bool {
	return apiErr != nil && apiErr.Status == status
}

func validationMentions(apiErr *xero.APIError, needles ...string) bool {
	if apiErr == nil {
		return false
	}
	for _, message := range apiErr.ValidationMessages {
		lower := strings.ToLower(message)
		for _, needle := range needles {
			if strings.Contains(lower, needle) {
				return true
			}
		}
	}
	return false
}

func isMappingFault(apiErr *xero.APIError) bool {
	if !hasStatus(apiErr, http.StatusBadRequest) {
		return false
	}
	return validationMentions(apiErr,
		"account code",
		"is not a valid code",
		"has been archived",
		"is archived",
		"item code",
		"contact could not be found",
	)
}

func errorCode(apiErr *xero.APIError) string {
	if apiErr == nil {
		return ""
	}
	if apiErr.Type != "" {
		return apiErr.Type
	}
	return strconv.Itoa(apiErr.Status)
}

func errorMessage(err error, apiErr *xero.APIError) string {
	message := err.Error()
	if apiErr != nil {
		switch {
		case len(apiErr.ValidationMessages) > 0:
			message = strings.Join(apiErr.ValidationMessages, "; ")
		case apiErr.Message != "":
			message = apiErr.Message
		}
	}
	return stringutils.TruncateRunes(message, maxErrorMessage)
}

func isLocalValidation(err error) bool {
	for _, target := range []error{
		xero.ErrTenantIDRequired,
		xero.ErrIDRequired,
		xero.ErrInvalidID,
		xero.ErrTooManyIDs,
		xero.ErrIdempotencyKey,
		xero.ErrDateRequired,
		xero.ErrContactRequired,
		xero.ErrLinesRequired,
		xero.ErrLineDescription,
		xero.ErrUnknownType,
		xero.ErrUnknownStatus,
		xero.ErrUnknownAmountTypes,
		xero.ErrAmountNotPositive,
		xero.ErrNegativeRate,
		xero.ErrTargetRequired,
		xero.ErrAccountRequired,
		xero.ErrPaymentsRequired,
		xero.ErrItemCodeInvalid,
		xero.ErrItemNameInvalid,
		xero.ErrContactNameInvalid,
		xero.ErrFieldTooLong,
		xero.ErrCurrencyCodeInvalid,
		xero.ErrEmptyFilter,
		xero.ErrInvalidFilter,
		errDocumentKind,
		errInvalidDate,
		errExternalID,
		errCreditNoteID,
		errItemDraftRequired,
		errPartyDraftRequired,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func parseDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %q", errInvalidDate, value)
	}
	return parsed, nil
}

func sameDate(t time.Time, raw string) bool {
	value := strings.TrimSpace(raw)
	return value == "" || (!t.IsZero() && t.UTC().Format(time.DateOnly) == value)
}

func sameMoney(a, b decimal.Decimal) bool {
	return a.Round(moneyPlaces).Equal(b.Round(moneyPlaces))
}

func sameID(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func newResult(refs map[string]string) *services.AccountingDocumentResult {
	copied := make(map[string]string, len(refs)+2)
	maps.Copy(copied, refs)
	return &services.AccountingDocumentResult{Refs: copied}
}
