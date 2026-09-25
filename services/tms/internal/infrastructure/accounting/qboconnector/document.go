package qboconnector

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/shopspring/decimal"
)

var _ services.AccountingDocumentWriter = (*Connector)(nil)

const (
	providerName        = "QuickBooks Online"
	debitMemoNotePrefix = "Debit memo"
	shortPayMemoNote    = "Short pay on "
	creditApplyNote     = "Applies credit memo "
	maxFaultMessage     = 2000
)

var errDocumentKind = errors.New("quickbooks: this record type is not a document QuickBooks holds")

func (c *Connector) DocumentLimits() services.AccountingDocumentLimits {
	return services.AccountingDocumentLimits{
		MaxDocNumberLength:      quickbooks.MaxDocNumberLength,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       false,
		CanVoidPurchaseDocument: false,
	}
}

func (c *Connector) DocumentURL(kind accountingsync.SyncObjectType, externalID string) string {
	var path string
	switch kind {
	case accountingsync.SyncObjectCustomer:
		path = quickbooks.CustomerAppPath()
	case accountingsync.SyncObjectCarrierVendor, accountingsync.SyncObjectDriverVendor:
		path = quickbooks.VendorAppPath()
	case accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo:
		path = quickbooks.TxnInvoice.AppPath()
	case accountingsync.SyncObjectCreditMemo:
		path = quickbooks.TxnCreditMemo.AppPath()
	case accountingsync.SyncObjectCustomerPayment, accountingsync.SyncObjectCreditApplication:
		path = quickbooks.TxnPayment.AppPath()
	case accountingsync.SyncObjectCarrierBill, accountingsync.SyncObjectDriverBill:
		path = quickbooks.TxnBill.AppPath()
	case accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncObjectDriverBillPay:
		path = quickbooks.TxnBillPayment.AppPath()
	default:
		return ""
	}
	return c.appURL(path, externalID)
}

func (c *Connector) appURL(path, externalID string) string {
	id := strings.TrimSpace(externalID)
	if id == "" || path == "" {
		return ""
	}
	return c.env.AppBaseURL() + path + url.QueryEscape(id)
}

func (c *Connector) client(auth services.AccountingDocumentAuth) (*quickbooks.Client, error) {
	return quickbooks.New(c.env, auth.RealmID, auth.AccessToken, c.apiOpts...)
}

func (c *Connector) UpsertCustomer(
	ctx context.Context,
	doc *services.AccountingCustomerDocument,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	draft := partyDraftOf(&doc.Party)
	draft.DisplayName = quickbooks.SanitizeName(draft.DisplayName, quickbooks.MaxPartyNameLength)

	var saved *quickbooks.ReferenceObject
	if id := strings.TrimSpace(doc.ExternalID); id != "" {
		saved, err = client.UpdateCustomer(ctx, doc.RequestID, id, draft)
	} else {
		saved, err = client.CreateCustomer(ctx, doc.RequestID, draft)
	}
	if err != nil {
		return nil, err
	}
	return &services.AccountingDocumentResult{ExternalID: saved.ID, DocNumber: saved.Name}, nil
}

func (c *Connector) CreateSalesDocument(
	ctx context.Context,
	doc *services.AccountingSalesDocument,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)

	if existing := result.Refs[accountingsync.ExternalRefDocument]; existing != "" {
		result.ExternalID = existing
	} else {
		txn := salesTxnOf(doc)
		var created *quickbooks.TxnResult
		switch doc.Kind {
		case accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo:
			created, err = client.CreateInvoice(ctx, doc.RequestID, txn)
		case accountingsync.SyncObjectCreditMemo:
			created, err = client.CreateCreditMemo(ctx, doc.RequestID, txn)
		case accountingsync.SyncObjectCustomer,
			accountingsync.SyncObjectCustomerPayment,
			accountingsync.SyncObjectCreditApplication,
			accountingsync.SyncObjectCarrierVendor,
			accountingsync.SyncObjectDriverVendor,
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncObjectCarrierBillPay,
			accountingsync.SyncObjectDriverBill,
			accountingsync.SyncObjectDriverBillPay:
			return nil, errDocumentKind
		default:
			return nil, errDocumentKind
		}
		if err != nil {
			return result, err
		}
		result.ExternalID = created.ID
		result.DocNumber = created.DocNumber
		result.Refs[accountingsync.ExternalRefDocument] = created.ID
	}

	if doc.Kind != accountingsync.SyncObjectCreditMemo ||
		strings.TrimSpace(doc.ApplyToExternalID) == "" ||
		!doc.ApplyAmount.IsPositive() ||
		result.Refs[accountingsync.ExternalRefApplication] != "" {
		return result, nil
	}

	applied, err := client.CreatePayment(
		ctx,
		accountingsync.SyncStepRequestID(doc.RequestID, 1),
		&quickbooks.PaymentTxn{
			CustomerID:   doc.CustomerExternalID,
			TxnDate:      doc.TxnDate,
			CurrencyCode: doc.CurrencyCode,
			PrivateNote:  creditApplyNote + doc.DocNumber,
			TotalAmount:  decimal.Zero,
			Links: []quickbooks.PaymentLink{
				{
					TxnID:   doc.ApplyToExternalID,
					TxnKind: quickbooks.TxnInvoice,
					Amount:  doc.ApplyAmount,
				},
				{
					TxnID:   result.ExternalID,
					TxnKind: quickbooks.TxnCreditMemo,
					Amount:  doc.ApplyAmount,
				},
			},
		},
	)
	if err != nil {
		return result, err
	}
	result.Refs[accountingsync.ExternalRefApplication] = applied.ID
	return result, nil
}

func salesTxnOf(doc *services.AccountingSalesDocument) *quickbooks.SalesTxn {
	note := doc.PrivateNote
	if doc.Kind == accountingsync.SyncObjectDebitMemo {
		note = strings.TrimSpace(debitMemoNotePrefix + ". " + note)
	}
	txn := &quickbooks.SalesTxn{
		CustomerID:   doc.CustomerExternalID,
		DocNumber:    doc.DocNumber,
		TxnDate:      doc.TxnDate,
		DueDate:      doc.DueDate,
		TermID:       doc.TermExternalID,
		CurrencyCode: doc.CurrencyCode,
		PrivateNote:  note,
		CustomerMemo: doc.CustomerMemo,
		Lines:        make([]quickbooks.SalesLine, 0, len(doc.Lines)),
	}
	if doc.Kind == accountingsync.SyncObjectCreditMemo {
		txn.DueDate = ""
		txn.TermID = ""
	}
	for idx := range doc.Lines {
		line := &doc.Lines[idx]
		txn.Lines = append(txn.Lines, quickbooks.SalesLine{
			Description: line.Description,
			ItemID:      line.ItemExternalID,
			Quantity:    line.Quantity,
			UnitPrice:   line.UnitPrice,
			Amount:      line.Amount,
			ServiceDate: line.ServiceDate,
		})
	}
	return txn
}

func (c *Connector) VoidSalesDocument(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	switch ref.Kind {
	case accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo:
		if _, err = client.VoidInvoice(ctx, ref.RequestID, ref.ExternalID); err != nil &&
			!quickbooks.IsObjectNotFound(err) {
			return result, err
		}
		return result, nil
	case accountingsync.SyncObjectCreditMemo:
		if application := result.Refs[accountingsync.ExternalRefApplication]; application != "" {
			if _, err = client.VoidPayment(
				ctx,
				accountingsync.SyncStepRequestID(ref.RequestID, 1),
				application,
			); err != nil && !quickbooks.IsObjectNotFound(err) {
				return result, err
			}
		}
		if _, err = client.DeleteCreditMemo(ctx, ref.RequestID, ref.ExternalID); err != nil &&
			!quickbooks.IsObjectNotFound(err) {
			return result, err
		}
		return result, nil
	case accountingsync.SyncObjectCustomer,
		accountingsync.SyncObjectCustomerPayment,
		accountingsync.SyncObjectCreditApplication,
		accountingsync.SyncObjectCarrierVendor,
		accountingsync.SyncObjectDriverVendor,
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectCarrierBillPay,
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay:
		return nil, errDocumentKind
	default:
		return nil, errDocumentKind
	}
}

func (c *Connector) SavePayment(
	ctx context.Context,
	doc *services.AccountingPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)

	links := make([]quickbooks.PaymentLink, 0, len(doc.Applications)*2)
	for idx := range doc.Applications {
		app := &doc.Applications[idx]
		links = append(links, quickbooks.PaymentLink{
			TxnID:   app.InvoiceExternalID,
			TxnKind: quickbooks.TxnInvoice,
			Amount:  app.AppliedAmount.Add(app.ShortPayAmount),
		})
		if !app.ShortPayAmount.IsPositive() {
			continue
		}
		memoID, memoErr := c.shortPayMemo(ctx, client, doc, idx, result)
		if memoErr != nil {
			return result, memoErr
		}
		links = append(links, quickbooks.PaymentLink{
			TxnID:   memoID,
			TxnKind: quickbooks.TxnCreditMemo,
			Amount:  app.ShortPayAmount,
		})
	}

	txn := &quickbooks.PaymentTxn{
		CustomerID:       doc.CustomerExternalID,
		TxnDate:          doc.TxnDate,
		CurrencyCode:     doc.CurrencyCode,
		PaymentMethodID:  doc.PaymentMethodExternalID,
		DepositAccountID: doc.DepositAccountExternalID,
		PaymentRefNum:    doc.ReferenceNumber,
		PrivateNote:      doc.PrivateNote,
		TotalAmount:      doc.TotalAmount,
		Links:            links,
	}

	var saved *quickbooks.TxnResult
	if id := strings.TrimSpace(doc.ExternalID); id != "" {
		saved, err = client.UpdatePayment(ctx, doc.RequestID, id, txn)
	} else {
		saved, err = client.CreatePayment(ctx, doc.RequestID, txn)
	}
	if err != nil {
		return result, err
	}
	result.ExternalID = saved.ID
	result.Refs[accountingsync.ExternalRefDocument] = saved.ID
	return result, nil
}

func shortPayRefKey(app *services.AccountingPaymentApplication) string {
	return accountingsync.ExternalRefShortPayPrefix + app.InvoiceExternalID
}

func (c *Connector) shortPayMemo(
	ctx context.Context,
	client *quickbooks.Client,
	doc *services.AccountingPaymentDocument,
	idx int,
	result *services.AccountingDocumentResult,
) (string, error) {
	app := &doc.Applications[idx]
	if id := strings.TrimSpace(app.ShortPayCreditExternalID); id != "" {
		return id, nil
	}
	key := shortPayRefKey(app)
	if id := result.Refs[key]; id != "" {
		return id, nil
	}
	if strings.TrimSpace(doc.ShortPayItemExternalID) == "" {
		return "", &accountingsync.SyncError{
			Category:   accountingsync.SyncErrorMapping,
			Code:       accountingsync.ItemRoleShortPayWriteOff,
			Message:    "The short pay write-off item is not mapped",
			Resolution: "Map the short pay write-off item to a " + providerName + " item",
		}
	}

	memo, err := client.CreateCreditMemo(
		ctx,
		accountingsync.SyncStepRequestID(doc.RequestID, idx+1),
		&quickbooks.SalesTxn{
			CustomerID:   doc.CustomerExternalID,
			TxnDate:      doc.TxnDate,
			CurrencyCode: doc.CurrencyCode,
			PrivateNote:  shortPayMemoNote + app.InvoiceNumber,
			Lines: []quickbooks.SalesLine{{
				Description: shortPayMemoNote + app.InvoiceNumber,
				ItemID:      doc.ShortPayItemExternalID,
				Amount:      app.ShortPayAmount,
			}},
		},
	)
	if err != nil {
		return "", err
	}
	result.Refs[key] = memo.ID
	return memo.ID, nil
}

func (c *Connector) VoidPayment(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	if _, err = client.VoidPayment(ctx, ref.RequestID, ref.ExternalID); err != nil &&
		!quickbooks.IsObjectNotFound(err) {
		return result, err
	}

	step := 0
	for _, key := range slices.Sorted(maps.Keys(result.Refs)) {
		if !strings.HasPrefix(key, accountingsync.ExternalRefShortPayPrefix) {
			continue
		}
		step++
		if _, err = client.DeleteCreditMemo(
			ctx,
			accountingsync.SyncStepRequestID(ref.RequestID, step),
			result.Refs[key],
		); err != nil && !quickbooks.IsObjectNotFound(err) {
			return result, err
		}
	}
	return result, nil
}

func (c *Connector) CreateCreditApplication(
	ctx context.Context,
	doc *services.AccountingCreditApplicationDocument,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	if !doc.Amount.IsPositive() {
		return nil, quickbooks.ErrNegativeAmount
	}

	applied, err := client.CreatePayment(ctx, doc.RequestID, &quickbooks.PaymentTxn{
		CustomerID:   doc.CustomerExternalID,
		TxnDate:      doc.TxnDate,
		CurrencyCode: doc.CurrencyCode,
		PrivateNote:  doc.PrivateNote,
		TotalAmount:  decimal.Zero,
		Links: []quickbooks.PaymentLink{
			{TxnID: doc.InvoiceExternalID, TxnKind: quickbooks.TxnInvoice, Amount: doc.Amount},
			{
				TxnID:   doc.CreditMemoExternalID,
				TxnKind: quickbooks.TxnCreditMemo,
				Amount:  doc.Amount,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: applied.ID,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: applied.ID},
	}, nil
}

func (c *Connector) VoidCreditApplication(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	if _, err = client.VoidPayment(ctx, ref.RequestID, ref.ExternalID); err != nil &&
		!quickbooks.IsObjectNotFound(err) {
		return nil, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: ref.ExternalID,
		Refs:       map[string]string{},
	}, nil
}

func (c *Connector) FindSalesDocument(
	ctx context.Context,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	var kind quickbooks.TxnKind
	switch req.Kind {
	case accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo:
		kind = quickbooks.TxnInvoice
	case accountingsync.SyncObjectCreditMemo:
		kind = quickbooks.TxnCreditMemo
	case accountingsync.SyncObjectCustomer,
		accountingsync.SyncObjectCustomerPayment,
		accountingsync.SyncObjectCreditApplication,
		accountingsync.SyncObjectCarrierVendor,
		accountingsync.SyncObjectDriverVendor,
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectCarrierBillPay,
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay:
		return nil, false, errDocumentKind
	default:
		return nil, false, errDocumentKind
	}
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, false, err
	}
	found, ok, err := client.FindTransactionByDocNumber(ctx, kind, req.DocNumber)
	if err != nil || !ok {
		return nil, false, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: found.ID,
		DocNumber:  found.DocNumber,
		Refs:       map[string]string{},
	}, true, nil
}

func (c *Connector) ClassifyDocumentError(err error) *accountingsync.SyncError {
	if err == nil {
		return nil
	}
	var syncErr *accountingsync.SyncError
	if errors.As(err, &syncErr) {
		return syncErr
	}

	code := strings.Join(quickbooks.FaultCodes(err), ",")
	message := faultMessage(err)
	classified := func(
		category accountingsync.SyncErrorCategory,
		resolution string,
	) *accountingsync.SyncError {
		return &accountingsync.SyncError{
			Category:   category,
			Code:       code,
			Message:    message,
			Resolution: resolution,
		}
	}

	switch {
	case errors.Is(err, ErrNotConfigured):
		return classified(accountingsync.SyncErrorConfiguration,
			providerName+" is not configured on this server")
	case quickbooks.IsInvalidGrant(err),
		quickbooks.IsUnauthorized(err),
		quickbooks.IsForbidden(err):
		return classified(accountingsync.SyncErrorAuth,
			"Reconnect "+providerName+" from the accounting integration")
	case quickbooks.IsRateLimited(err):
		return classified(accountingsync.SyncErrorRateLimited, "")
	case quickbooks.IsStaleObject(err),
		quickbooks.IsTransient(err),
		errors.Is(err, context.DeadlineExceeded):
		return classified(accountingsync.SyncErrorTransient, "")
	case quickbooks.IsDuplicateDocNumber(err):
		return classified(accountingsync.SyncErrorDuplicate,
			"A document with this number already exists in "+providerName+
				". Skip this record if it is the same document, or renumber the one in "+providerName)
	case quickbooks.IsDuplicateName(err):
		return classified(accountingsync.SyncErrorDuplicate,
			"A customer, vendor or employee with this name already exists in "+providerName+
				". Map the record to it instead, or rename one of them")
	case quickbooks.IsClosedPeriod(err):
		return classified(accountingsync.SyncErrorClosedPeriod,
			"The books are closed for this date in "+providerName+
				". Reopen the period there or skip this record")
	case quickbooks.IsObjectNotFound(err):
		return classified(accountingsync.SyncErrorNotFound,
			"The linked record no longer exists in "+providerName)
	case quickbooks.IsInvalidReference(err):
		return classified(accountingsync.SyncErrorMapping,
			"A mapped "+providerName+" record is missing or inactive. "+
				"Refresh the reference data and fix the mapping")
	case isLocalValidation(err), quickbooks.IsValidationFault(err):
		return classified(accountingsync.SyncErrorValidation,
			"Correct the document in Trenova, then retry")
	default:
		return classified(accountingsync.SyncErrorTransient, "")
	}
}

func isLocalValidation(err error) bool {
	for _, target := range []error{
		quickbooks.ErrCustomerRequired,
		quickbooks.ErrLinesRequired,
		quickbooks.ErrItemRequired,
		quickbooks.ErrDocNumberTooLong,
		quickbooks.ErrNegativeAmount,
		quickbooks.ErrTxnIDRequired,
		quickbooks.ErrUnknownTxnKind,
		quickbooks.ErrInvalidName,
		quickbooks.ErrRequestIDRequired,
		quickbooks.ErrVendorRequired,
		quickbooks.ErrAccountRequired,
		quickbooks.ErrBankAccountRequired,
		quickbooks.ErrBillRequired,
		quickbooks.ErrNonPositiveAmount,
		errDocumentKind,
		errPurchaseDocumentType,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func faultMessage(err error) string {
	var fault *quickbooks.FaultError
	if errors.As(err, &fault) && len(fault.Errors) > 0 {
		parts := make([]string, 0, len(fault.Errors))
		for idx := range fault.Errors {
			detail := strings.TrimSpace(fault.Errors[idx].Detail)
			if detail == "" {
				detail = strings.TrimSpace(fault.Errors[idx].Message)
			}
			parts = append(parts, detail)
		}
		return truncateMessage(strings.Join(parts, "; "))
	}
	return truncateMessage(fmt.Sprint(err))
}

func truncateMessage(message string) string {
	runes := []rune(message)
	if len(runes) <= maxFaultMessage {
		return message
	}
	return string(runes[:maxFaultMessage])
}

func newResult(refs map[string]string) *services.AccountingDocumentResult {
	copied := make(map[string]string, len(refs)+2)
	maps.Copy(copied, refs)
	return &services.AccountingDocumentResult{Refs: copied}
}
