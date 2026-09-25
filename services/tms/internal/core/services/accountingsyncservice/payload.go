package accountingsyncservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const trenovaNotePrefix = "Trenova "

func documentLabel(objectType accountingsync.SyncObjectType, number string) string {
	var kind string
	switch objectType {
	case accountingsync.SyncObjectCreditMemo:
		kind = "Credit memo"
	case accountingsync.SyncObjectDebitMemo:
		kind = "Debit memo"
	case accountingsync.SyncObjectCustomerPayment:
		kind = "Payment"
	case accountingsync.SyncObjectCreditApplication:
		kind = "Credit application"
	case accountingsync.SyncObjectCustomer:
		kind = "Customer"
	case accountingsync.SyncObjectInvoice:
		kind = "Invoice"
	default:
		kind = "Document"
	}
	if number == "" {
		return kind
	}
	return kind + " " + number
}

func (s *Service) checkBooks(
	sess *pushSession,
	currency string,
	documentDate int64,
	label string,
) error {
	home := strings.ToUpper(strings.TrimSpace(sess.conn.ExternalHomeCurrency))
	code := strings.ToUpper(strings.TrimSpace(currency))
	if home != "" && code != "" && code != home && !sess.conn.ExternalMultiCurrencyEnabled {
		return blocked(
			accountingsync.SyncErrorCurrency,
			label+" is in "+code+" but the books are kept in "+home,
			"Turn on multicurrency in "+sess.providerName+", or skip this record",
		)
	}
	if sess.conn.BooksClosedOn(documentDate) {
		return blocked(
			accountingsync.SyncErrorClosedPeriod,
			label+" is dated in a period closed in "+sess.providerName,
			"Reopen the period in "+sess.providerName+", or skip this record",
		)
	}
	return nil
}

func (s *Service) customerRef(
	ctx context.Context,
	sess *pushSession,
	res *resolver,
	customerID pulid.ID,
) (string, error) {
	row, err := res.mapping(ctx, mappingTarget{
		TargetType: accountingsync.TargetCustomer,
		ObjectID:   customerID,
	})
	if err != nil {
		return "", err
	}
	if row.State == accountingsync.MappingStateConfirmed && row.ExternalID != "" {
		res.used[row.ID] = struct{}{}
		return row.ExternalID, nil
	}

	dependency, err := s.enqueueDependency(ctx, sess, &services.AccountingSyncEnqueueRequest{
		ObjectType:   accountingsync.SyncObjectCustomer,
		ObjectID:     customerID,
		ObjectNumber: row.TargetLabel,
		Operation:    accountingsync.SyncOperationCreate,
		Revision:     1,
	})
	if err != nil {
		return "", err
	}
	if dependency == nil || dependency.Status == accountingsync.SyncStatusSynced ||
		dependency.Status == accountingsync.SyncStatusSkipped {
		return "", res.missing(row, mappingTarget{
			TargetType: accountingsync.TargetCustomer,
			ObjectID:   customerID,
			Label:      "Customer " + row.TargetLabel,
		})
	}
	s.kick(ctx, sess.tenant, sess.conn.ID)
	return "", waitingOn(dependency, "Customer "+row.TargetLabel)
}

func (s *Service) salesRef(
	ctx context.Context,
	sess *pushSession,
	inv *invoice.Invoice,
) (string, error) {
	objectType := accountingsync.SyncObjectTypeForBill(inv.BillType)
	label := documentLabel(objectType, inv.Number)
	existing, err := s.objectRecords(ctx, sess, objectType, inv.ID)
	if err != nil {
		return "", err
	}
	if created := latestOf(existing, accountingsync.SyncOperationCreate); created != nil {
		switch {
		case isSynced(created):
			return created.ExternalID, nil
		case !created.Status.IsFinal():
			return "", waitingOn(created, label)
		}
	} else if sess.conn.Covers(inv.InvoiceDate) && inv.PostedAt != nil {
		dependency, enqueueErr := s.enqueueDependency(
			ctx,
			sess,
			&services.AccountingSyncEnqueueRequest{
				ObjectType:   objectType,
				ObjectID:     inv.ID,
				ObjectNumber: inv.Number,
				Operation:    accountingsync.SyncOperationCreate,
				Revision:     1,
				DocumentDate: inv.InvoiceDate,
			},
		)
		if enqueueErr != nil {
			return "", enqueueErr
		}
		if dependency != nil && !dependency.Status.IsFinal() {
			s.kick(ctx, sess.tenant, sess.conn.ID)
			return "", waitingOn(dependency, label)
		}
	}

	found, ok, err := sess.writer.FindSalesDocument(ctx, &services.AccountingFindDocumentRequest{
		Auth:      sess.auth,
		Kind:      objectType,
		DocNumber: inv.Number,
	})
	if err != nil {
		return "", err
	}
	if !ok {
		return "", blocked(
			accountingsync.SyncErrorNotFound,
			label+" is not in "+sess.providerName,
			"Enter "+label+" in "+sess.providerName+" with the same number, then retry; or skip this record",
		)
	}
	return found.ExternalID, nil
}

func (s *Service) pushCustomer(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	res := newResolver(s, sess)
	row, err := res.mapping(ctx, mappingTarget{
		TargetType: accountingsync.TargetCustomer,
		ObjectID:   record.ObjectID,
	})
	if err != nil {
		return nil, err
	}

	if record.Operation == accountingsync.SyncOperationCreate {
		if row.State == accountingsync.MappingStateConfirmed && row.ExternalID != "" {
			res.used[row.ID] = struct{}{}
			return finishedResult(sess, record, &services.AccountingDocumentResult{
				ExternalID: row.ExternalID,
				DocNumber:  row.ExternalName,
			}, nil, res.mappingIDs())
		}
		created, createErr := s.mappingService.CreateReferenceRecord(
			ctx,
			&services.CreateAccountingReferenceRecordRequest{
				TenantInfo: sess.tenant,
				UserID:     sess.conn.ConnectedByID,
				MappingID:  row.ID,
				Source:     accountingsync.MappingSourceCreatedInProvider,
			},
		)
		if createErr != nil {
			return nil, customerCreateError(sess, row, createErr)
		}
		return finishedResult(sess, record, &services.AccountingDocumentResult{
			ExternalID: created.ExternalID,
			DocNumber:  created.ExternalName,
		}, nil, []string{created.ID.String()})
	}

	if row.State != accountingsync.MappingStateConfirmed || row.ExternalID == "" {
		return nil, &noopError{reason: "The customer is not mapped, so there is nothing to update"}
	}
	party, err := s.mappingService.CustomerParty(ctx, sess.tenant, record.ObjectID)
	if err != nil {
		return nil, err
	}
	doc := &services.AccountingCustomerDocument{
		Auth:       sess.auth,
		RequestID:  record.RequestID,
		ExternalID: row.ExternalID,
		Party:      *party,
	}
	written, err := sess.writer.UpsertCustomer(ctx, doc)
	if err != nil {
		return nil, err
	}
	res.used[row.ID] = struct{}{}
	return finishedResult(sess, record, written, doc, res.mappingIDs())
}

func customerCreateError(
	sess *pushSession,
	row *accountingsync.AccountingMapping,
	err error,
) error {
	if errortypes.IsNotFoundError(err) {
		return blocked(accountingsync.SyncErrorNotFound, err.Error(), "Skip this record")
	}
	classified := sess.writer.ClassifyDocumentError(err)
	if classified != nil && classified.Category == accountingsync.SyncErrorDuplicate {
		return blocked(
			accountingsync.SyncErrorMapping,
			err.Error(),
			"Map "+row.TargetLabel+" to the existing "+sess.providerName+" customer, then retry",
		)
	}
	if classified != nil &&
		(classified.Category.Retries() || classified.Category.WaitsOnConnection()) {
		return err
	}
	if errortypes.IsBusinessError(err) || errortypes.IsError(err) {
		return blocked(
			accountingsync.SyncErrorValidation,
			err.Error(),
			"Correct the customer in Trenova or map it by hand, then retry",
		)
	}
	return err
}

func (s *Service) pushSales(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	inv, err := s.invoices.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         record.ObjectID,
		TenantInfo: sess.tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, blocked(accountingsync.SyncErrorNotFound,
				"The document no longer exists in Trenova", "Skip this record")
		}
		return nil, err
	}
	label := documentLabel(record.ObjectType, inv.Number)
	if inv.PostedAt == nil ||
		(inv.Status != invoice.StatusPosted && inv.Status != invoice.StatusVoided) {
		return nil, blocked(accountingsync.SyncErrorValidation,
			label+" is not posted", "Post it in Trenova, or skip this record")
	}
	if err = s.checkBooks(sess, inv.CurrencyCode, inv.InvoiceDate, label); err != nil {
		return nil, err
	}

	res := newResolver(s, sess)
	customerID, err := s.customerRef(ctx, sess, res, inv.CustomerID)
	if err != nil {
		return nil, err
	}

	doc, err := s.salesDocument(ctx, sess, res, record, inv)
	if err != nil {
		return nil, err
	}
	doc.CustomerExternalID = customerID

	written, err := sess.writer.CreateSalesDocument(ctx, doc)
	if err != nil {
		return partial(written), err
	}
	return finishedResult(sess, record, written, doc, res.mappingIDs())
}

func (s *Service) salesDocument(
	ctx context.Context,
	sess *pushSession,
	res *resolver,
	record *accountingsync.AccountingSyncRecord,
	inv *invoice.Invoice,
) (*services.AccountingSalesDocument, error) {
	sign := decimal.NewFromInt(1)
	if record.ObjectType == accountingsync.SyncObjectCreditMemo && inv.TotalAmount.IsNegative() {
		sign = decimal.NewFromInt(-1)
	}

	doc := &services.AccountingSalesDocument{
		Auth:         sess.auth,
		RequestID:    record.RequestID,
		Kind:         record.ObjectType,
		DocNumber:    inv.Number,
		TxnDate:      timeutils.FormatCalendarDate(inv.InvoiceDate, sess.loc),
		CurrencyCode: inv.CurrencyCode,
		PrivateNote:  salesNote(inv),
		CustomerMemo: customerMemo(inv),
		Lines:        make([]services.AccountingDocumentLine, 0, len(inv.Lines)),
		Refs:         record.ExternalRefs,
	}
	if limit := sess.limits.MaxDocNumberLength; limit > 0 && len([]rune(inv.Number)) > limit {
		doc.DocNumber = ""
	}
	if inv.DueDate != nil && record.ObjectType != accountingsync.SyncObjectCreditMemo {
		doc.DueDate = timeutils.FormatCalendarDate(*inv.DueDate, sess.loc)
	}
	if inv.PaymentTerm != "" && record.ObjectType != accountingsync.SyncObjectCreditMemo {
		term, err := res.optional(ctx, mappingTarget{
			TargetType: accountingsync.TargetPaymentTerm,
			Key:        string(inv.PaymentTerm),
		})
		if err != nil {
			return nil, err
		}
		doc.TermExternalID = term
	}

	for _, line := range inv.Lines {
		if line == nil || line.Amount.IsZero() {
			continue
		}
		itemID, err := res.require(ctx, lineTarget(line))
		if err != nil {
			return nil, err
		}
		entry := services.AccountingDocumentLine{
			Description:    line.Description,
			ItemExternalID: itemID,
			Quantity:       line.Quantity.Abs(),
			UnitPrice:      line.UnitPrice.Mul(sign),
			Amount:         line.Amount.Mul(sign),
		}
		if inv.ServiceDate != nil {
			entry.ServiceDate = timeutils.FormatCalendarDate(*inv.ServiceDate, sess.loc)
		}
		doc.Lines = append(doc.Lines, entry)
	}
	if len(doc.Lines) == 0 {
		return nil, &noopError{reason: documentLabel(record.ObjectType, inv.Number) +
			" has no amounts to send"}
	}

	if record.ObjectType == accountingsync.SyncObjectCreditMemo {
		if err := s.applyReversal(ctx, sess, inv, doc); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

func lineTarget(line *invoice.InvoiceLine) mappingTarget {
	if !line.AccessorialChargeID.IsNil() {
		label := "Charge code " + line.ChargeCode
		if line.ChargeCode == "" {
			label = "Accessorial charge " + line.Description
		}
		return mappingTarget{
			TargetType: accountingsync.TargetAccessorialCharge,
			ObjectID:   line.AccessorialChargeID,
			Label:      label,
		}
	}
	key := string(line.Type)
	if line.Type != invoice.InvoiceLineTypeMemo {
		key = string(invoice.InvoiceLineTypeFreight)
	}
	return mappingTarget{
		TargetType: accountingsync.TargetLineType,
		Key:        key,
		Label:      strings.ToLower(key) + " lines",
	}
}

func salesNote(inv *invoice.Invoice) string {
	parts := []string{trenovaNotePrefix + inv.Number}
	if inv.ShipmentProNumber != "" {
		parts = append(parts, "PRO "+inv.ShipmentProNumber)
	}
	if inv.MemoReason != "" {
		parts = append(parts, inv.MemoReason)
	}
	return strings.Join(parts, " · ")
}

func customerMemo(inv *invoice.Invoice) string {
	parts := make([]string, 0, 3)
	if inv.ShipmentProNumber != "" {
		parts = append(parts, "PRO "+inv.ShipmentProNumber)
	}
	if inv.ShipmentBOL != "" {
		parts = append(parts, "BOL "+inv.ShipmentBOL)
	}
	if inv.OrderNumber != "" {
		parts = append(parts, "Order "+inv.OrderNumber)
	}
	return strings.Join(parts, " · ")
}

func (s *Service) applyReversal(
	ctx context.Context,
	sess *pushSession,
	memo *invoice.Invoice,
	doc *services.AccountingSalesDocument,
) error {
	if memo.SourceInvoiceAdjustmentID.IsNil() {
		return nil
	}
	adjustment, err := s.adjustments.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID:         memo.SourceInvoiceAdjustmentID,
		TenantInfo: sess.tenant,
	})
	if err != nil {
		return err
	}
	if adjustment.Kind != invoiceadjustment.KindFullReversal {
		return nil
	}
	original, err := s.invoices.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         adjustment.OriginalInvoiceID,
		TenantInfo: sess.tenant,
	})
	if err != nil {
		return err
	}

	externalID, err := s.salesRef(ctx, sess, original)
	if err != nil {
		var syncErr *accountingsync.SyncError
		if !errors.As(err, &syncErr) || syncErr.Category != accountingsync.SyncErrorNotFound {
			return err
		}
		return &noopError{reason: documentLabel(accountingsync.SyncObjectInvoice, original.Number) +
			" was never in " + sess.providerName + ", so its reversal is not sent either"}
	}
	doc.ApplyToExternalID = externalID
	doc.ApplyAmount = memo.TotalAmount.Abs()
	return nil
}

func (s *Service) pushSalesVoid(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	existing, err := s.objectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	created := latestOf(existing, accountingsync.SyncOperationCreate)
	if done, voidErr := s.settleUnsent(ctx, created, record); done {
		return nil, voidErr
	}
	ref := &services.AccountingDocumentRef{
		Auth:       sess.auth,
		RequestID:  record.RequestID,
		Kind:       record.ObjectType,
		ExternalID: created.ExternalID,
		Refs:       mergedRefs(existing),
	}
	written, err := sess.writer.VoidSalesDocument(ctx, ref)
	if err != nil {
		return partial(written), err
	}
	return finishedResult(sess, record, written, ref, nil)
}

func (s *Service) settleUnsent(
	ctx context.Context,
	created *accountingsync.AccountingSyncRecord,
	record *accountingsync.AccountingSyncRecord,
) (bool, error) {
	switch {
	case created == nil:
		return true, &noopError{reason: "It never reached the books, so there is nothing to undo"}
	case isSynced(created):
		return false, nil
	case created.Status == accountingsync.SyncStatusInFlight:
		return true, waitFor(created.ID, "Waiting for the original to finish sending")
	default:
		if created.Supersede() {
			if _, err := s.records.Update(ctx, created); err != nil {
				return true, err
			}
		}
		if _, err := s.records.SupersedeOlder(
			ctx,
			&repositories.SupersedeAccountingSyncRecordsRequest{
				TenantInfo:     record.TenantInfo(),
				ConnectionID:   record.ConnectionID,
				ObjectType:     record.ObjectType,
				ObjectID:       record.ObjectID,
				Operation:      accountingsync.SyncOperationUpdate,
				BeforeRevision: maxRevision,
			},
		); err != nil {
			return true, err
		}
		return true, &noopError{
			reason: "It never reached the books, so the queued send was withdrawn",
		}
	}
}

const maxRevision = int64(1) << 62

func paymentMethodLabel(method customerpayment.Method) string {
	return "Payment method " + string(method)
}

func (s *Service) pushPayment(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	payment, err := s.payments.GetByID(ctx, repositories.GetCustomerPaymentByIDRequest{
		ID:         record.ObjectID,
		TenantInfo: sess.tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, blocked(accountingsync.SyncErrorNotFound,
				"The payment no longer exists in Trenova", "Skip this record")
		}
		return nil, err
	}
	label := documentLabel(accountingsync.SyncObjectCustomerPayment, payment.ReferenceNumber)
	if err = s.checkBooks(sess, payment.CurrencyCode, payment.AccountingDate, label); err != nil {
		return nil, err
	}

	existing, err := s.objectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	externalID := ""
	if record.Operation == accountingsync.SyncOperationUpdate {
		created := latestOf(existing, accountingsync.SyncOperationCreate)
		if created == nil {
			dependency, enqueueErr := s.enqueueDependency(
				ctx,
				sess,
				&services.AccountingSyncEnqueueRequest{
					ObjectType:   record.ObjectType,
					ObjectID:     record.ObjectID,
					ObjectNumber: record.ObjectNumber,
					Operation:    accountingsync.SyncOperationCreate,
					Revision:     1,
					DocumentDate: payment.AccountingDate,
				},
			)
			if enqueueErr != nil {
				return nil, enqueueErr
			}
			created = dependency
		}
		switch {
		case created == nil:
			return nil, &noopError{
				reason: "The payment was never sent, so there is nothing to update",
			}
		case isSynced(created):
			externalID = created.ExternalID
		case created.Status.IsFinal():
			return nil, &noopError{reason: "The payment was skipped, so its changes are not sent"}
		default:
			s.kick(ctx, sess.tenant, sess.conn.ID)
			return nil, waitingOn(created, label)
		}
	}

	res := newResolver(s, sess)
	refs, err := s.paymentRefs(ctx, sess, res, payment)
	if err != nil {
		return nil, err
	}

	doc := &services.AccountingPaymentDocument{
		Auth:                     sess.auth,
		RequestID:                record.RequestID,
		ExternalID:               externalID,
		CustomerExternalID:       refs.customer,
		TxnDate:                  timeutils.FormatCalendarDate(payment.AccountingDate, sess.loc),
		CurrencyCode:             payment.CurrencyCode,
		PaymentMethodExternalID:  refs.method,
		DepositAccountExternalID: refs.deposit,
		ReferenceNumber:          payment.ReferenceNumber,
		PrivateNote:              paymentNote(payment),
		TotalAmount:              money.DecimalFromMinor(payment.AmountMinor),
		Applications: make(
			[]services.AccountingPaymentApplication,
			0,
			len(payment.Applications),
		),
		Refs: record.ExternalRefs,
	}
	if err = s.paymentApplications(ctx, sess, res, payment, existing, doc); err != nil {
		return nil, err
	}

	written, err := sess.writer.SavePayment(ctx, doc)
	if err != nil {
		return partial(written), err
	}
	return finishedResult(sess, record, written, doc, res.mappingIDs())
}

type paymentExternalRefs struct {
	customer string
	deposit  string
	method   string
}

func (s *Service) paymentRefs(
	ctx context.Context,
	sess *pushSession,
	res *resolver,
	payment *customerpayment.Payment,
) (paymentExternalRefs, error) {
	customerID, err := s.customerRef(ctx, sess, res, payment.CustomerID)
	if err != nil {
		return paymentExternalRefs{}, err
	}
	deposit, err := res.require(ctx, mappingTarget{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleDeposit,
		Label:      "The deposit account",
	})
	if err != nil {
		return paymentExternalRefs{}, err
	}
	method, err := res.require(ctx, mappingTarget{
		TargetType: accountingsync.TargetPaymentMethod,
		Key:        string(payment.PaymentMethod),
		Label:      paymentMethodLabel(payment.PaymentMethod),
	})
	if err != nil {
		return paymentExternalRefs{}, err
	}
	return paymentExternalRefs{customer: customerID, deposit: deposit, method: method}, nil
}

func paymentNote(payment *customerpayment.Payment) string {
	note := trenovaNotePrefix + "payment"
	if payment.ReferenceNumber != "" {
		note += " " + payment.ReferenceNumber
	}
	if payment.Memo != "" {
		note += " · " + payment.Memo
	}
	return note
}

func (s *Service) paymentApplications(
	ctx context.Context,
	sess *pushSession,
	res *resolver,
	payment *customerpayment.Payment,
	existing []*accountingsync.AccountingSyncRecord,
	doc *services.AccountingPaymentDocument,
) error {
	if len(payment.Applications) == 0 {
		return nil
	}
	ids := make([]pulid.ID, 0, len(payment.Applications))
	for _, app := range payment.Applications {
		ids = append(ids, app.InvoiceID)
	}
	invoices, err := s.invoices.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
		TenantInfo: sess.tenant,
		InvoiceIDs: ids,
	})
	if err != nil {
		return err
	}
	byID := make(map[pulid.ID]*invoice.Invoice, len(invoices))
	for _, inv := range invoices {
		byID[inv.ID] = inv
	}

	priorRefs := mergedRefs(existing)
	shortPaid := false
	for _, app := range payment.Applications {
		inv, ok := byID[app.InvoiceID]
		if !ok {
			return blocked(accountingsync.SyncErrorNotFound,
				"An invoice this payment applies to no longer exists", "Skip this record")
		}
		externalID, refErr := s.salesRef(ctx, sess, inv)
		if refErr != nil {
			return refErr
		}
		entry := services.AccountingPaymentApplication{
			InvoiceExternalID: externalID,
			InvoiceNumber:     inv.Number,
			AppliedAmount:     money.DecimalFromMinor(app.AppliedAmountMinor),
			ShortPayAmount:    money.DecimalFromMinor(app.ShortPayAmountMinor),
			ShortPayCreditExternalID: priorRefs[accountingsync.ExternalRefShortPayPrefix+
				externalID],
		}
		shortPaid = shortPaid || app.ShortPayAmountMinor > 0
		doc.Applications = append(doc.Applications, entry)
	}

	if shortPaid {
		item, itemErr := res.require(ctx, mappingTarget{
			TargetType: accountingsync.TargetItemRole,
			Key:        accountingsync.ItemRoleShortPayWriteOff,
			Label:      "The short-pay write-off item",
		})
		if itemErr != nil {
			return itemErr
		}
		doc.ShortPayItemExternalID = item
	}
	return nil
}

func (s *Service) pushPaymentVoid(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	existing, err := s.objectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	created := latestOf(existing, accountingsync.SyncOperationCreate)
	if done, voidErr := s.settleUnsent(ctx, created, record); done {
		return nil, voidErr
	}
	for _, other := range existing {
		if other.Operation == accountingsync.SyncOperationUpdate &&
			other.Status == accountingsync.SyncStatusInFlight {
			return nil, waitFor(other.ID, "Waiting for a change to the payment to finish sending")
		}
	}
	if _, err = s.records.SupersedeOlder(ctx, &repositories.SupersedeAccountingSyncRecordsRequest{
		TenantInfo:     record.TenantInfo(),
		ConnectionID:   record.ConnectionID,
		ObjectType:     record.ObjectType,
		ObjectID:       record.ObjectID,
		Operation:      accountingsync.SyncOperationUpdate,
		BeforeRevision: maxRevision,
	}); err != nil {
		return nil, err
	}

	ref := &services.AccountingDocumentRef{
		Auth:       sess.auth,
		RequestID:  record.RequestID,
		Kind:       record.ObjectType,
		ExternalID: created.ExternalID,
		Refs:       mergedRefs(existing),
	}
	written, err := sess.writer.VoidPayment(ctx, ref)
	if err != nil {
		return partial(written), err
	}
	return finishedResult(sess, record, written, ref, nil)
}

func (s *Service) pushCreditApplication(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	app, err := s.payments.GetCreditMemoApplicationByID(
		ctx,
		repositories.GetCreditMemoApplicationRequest{
			ID:         record.ObjectID,
			TenantInfo: sess.tenant,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, blocked(accountingsync.SyncErrorNotFound,
				"The credit application no longer exists in Trenova", "Skip this record")
		}
		return nil, err
	}
	invoices, err := s.invoices.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
		TenantInfo: sess.tenant,
		InvoiceIDs: []pulid.ID{app.InvoiceID, app.CreditMemoInvoiceID},
	})
	if err != nil {
		return nil, err
	}
	var target, memo *invoice.Invoice
	for _, inv := range invoices {
		switch inv.ID {
		case app.InvoiceID:
			target = inv
		case app.CreditMemoInvoiceID:
			memo = inv
		}
	}
	if target == nil || memo == nil {
		return nil, blocked(accountingsync.SyncErrorNotFound,
			"A document this credit application links no longer exists", "Skip this record")
	}
	label := documentLabel(accountingsync.SyncObjectCreditApplication, memo.Number)
	if err = s.checkBooks(sess, memo.CurrencyCode, app.AccountingDate, label); err != nil {
		return nil, err
	}

	res := newResolver(s, sess)
	customerID, err := s.customerRef(ctx, sess, res, target.CustomerID)
	if err != nil {
		return nil, err
	}
	invoiceID, err := s.salesRef(ctx, sess, target)
	if err != nil {
		return nil, err
	}
	memoID, err := s.salesRef(ctx, sess, memo)
	if err != nil {
		return nil, err
	}

	doc := &services.AccountingCreditApplicationDocument{
		Auth:                 sess.auth,
		RequestID:            record.RequestID,
		CustomerExternalID:   customerID,
		TxnDate:              timeutils.FormatCalendarDate(app.AccountingDate, sess.loc),
		CurrencyCode:         memo.CurrencyCode,
		InvoiceExternalID:    invoiceID,
		CreditMemoExternalID: memoID,
		Amount:               money.DecimalFromMinor(app.AppliedAmountMinor),
		PrivateNote:          "Applies " + memo.Number + " to " + target.Number,
	}
	written, err := sess.writer.CreateCreditApplication(ctx, doc)
	if err != nil {
		return partial(written), err
	}
	return finishedResult(sess, record, written, doc, res.mappingIDs())
}

func (s *Service) pushCreditApplicationVoid(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	existing, err := s.objectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	created := latestOf(existing, accountingsync.SyncOperationCreate)
	if done, voidErr := s.settleUnsent(ctx, created, record); done {
		return nil, voidErr
	}
	ref := &services.AccountingDocumentRef{
		Auth:       sess.auth,
		RequestID:  record.RequestID,
		Kind:       record.ObjectType,
		ExternalID: created.ExternalID,
		Refs:       mergedRefs(existing),
	}
	written, err := sess.writer.VoidCreditApplication(ctx, ref)
	if err != nil {
		return partial(written), err
	}
	return finishedResult(sess, record, written, ref, nil)
}
