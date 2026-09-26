package accountingsyncservice

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

func partyLabel(objectType accountingsync.SyncObjectType) string {
	if objectType == accountingsync.SyncObjectCarrierVendor {
		return "Carrier"
	}
	if objectType == accountingsync.SyncObjectDriverVendor {
		return "Driver"
	}
	return "Customer"
}

func payableKind(objectType accountingsync.SyncObjectType) repositories.PayableKind {
	if objectType.IsDriverSettlement() {
		return repositories.PayableDriver
	}
	return repositories.PayableCarrier
}

func (s *Service) partyRef(
	ctx context.Context,
	sess *pushSession,
	res *resolver,
	objectType accountingsync.SyncObjectType,
	partyID pulid.ID,
) (string, error) {
	targetType, _ := objectType.PartyTarget()
	row, err := res.mapping(ctx, mappingTarget{TargetType: targetType, ObjectID: partyID})
	if err != nil {
		return "", err
	}
	if row.State == accountingsync.MappingStateConfirmed && row.ExternalID != "" {
		res.used[row.ID] = struct{}{}
		return row.ExternalID, nil
	}

	label := partyLabel(objectType) + " " + row.TargetLabel
	dependency, err := s.enqueueDependency(ctx, sess, &services.AccountingSyncEnqueueRequest{
		ObjectType:   objectType,
		ObjectID:     partyID,
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
			TargetType: targetType,
			ObjectID:   partyID,
			Label:      label,
		})
	}
	s.kick(ctx, sess.tenant, sess.conn.ID)
	return "", waitingOn(dependency, label)
}

func (s *Service) pushParty(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	targetType, _ := record.ObjectType.PartyTarget()
	res := newResolver(s, sess)
	row, err := res.mapping(ctx, mappingTarget{TargetType: targetType, ObjectID: record.ObjectID})
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
			return nil, partyCreateError(sess, record.ObjectType, row, createErr)
		}
		return finishedResult(sess, record, &services.AccountingDocumentResult{
			ExternalID: created.ExternalID,
			DocNumber:  created.ExternalName,
		}, nil, []string{created.ID.String()})
	}

	if row.State != accountingsync.MappingStateConfirmed || row.ExternalID == "" {
		return nil, &noopError{
			reason: "The " + strings.ToLower(partyLabel(record.ObjectType)) +
				" is not mapped, so there is nothing to update",
		}
	}
	written, doc, err := s.upsertParty(ctx, sess, record, targetType, row.ExternalID)
	if err != nil {
		return nil, err
	}
	res.used[row.ID] = struct{}{}
	return finishedResult(sess, record, written, doc, res.mappingIDs())
}

func (s *Service) upsertParty(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
	targetType accountingsync.MappingTargetType,
	externalID string,
) (*services.AccountingDocumentResult, any, error) {
	if record.ObjectType == accountingsync.SyncObjectCustomer {
		party, err := s.mappingService.CustomerParty(ctx, sess.tenant, record.ObjectID)
		if err != nil {
			return nil, nil, err
		}
		doc := &services.AccountingCustomerDocument{
			Auth:       sess.auth,
			RequestID:  record.RequestID,
			ExternalID: externalID,
			Party:      *party,
		}
		written, err := sess.writer.UpsertCustomer(ctx, doc)
		return written, doc, err
	}

	party, err := s.mappingService.VendorParty(ctx, sess.tenant, targetType, record.ObjectID)
	if err != nil {
		return nil, nil, err
	}
	doc := &services.AccountingVendorDocument{
		Auth:       sess.auth,
		RequestID:  record.RequestID,
		ExternalID: externalID,
		Party:      *party,
	}
	written, err := sess.writer.UpsertVendor(ctx, doc)
	return written, doc, err
}

func partyCreateError(
	sess *pushSession,
	objectType accountingsync.SyncObjectType,
	row *accountingsync.AccountingMapping,
	err error,
) error {
	if errortypes.IsNotFoundError(err) {
		return blocked(accountingsync.SyncErrorNotFound, err.Error(), "Skip this record")
	}
	kind := "customer"
	if objectType.IsVendor() {
		kind = "vendor"
	}
	classified := sess.writer.ClassifyDocumentError(err)
	if classified != nil && classified.Category == accountingsync.SyncErrorDuplicate {
		return blocked(
			accountingsync.SyncErrorMapping,
			err.Error(),
			"Map "+row.TargetLabel+" to the existing "+sess.providerName+" "+kind+", then retry",
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
			"Correct the "+strings.ToLower(partyLabel(objectType))+
				" in Trenova or map it by hand, then retry",
		)
	}
	return err
}

func (s *Service) payableSettlement(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*repositories.PayableSettlement, error) {
	settlement, err := s.payables.GetSettlement(ctx, &repositories.GetPayableSettlementRequest{
		TenantInfo: sess.tenant,
		Kind:       payableKind(record.ObjectType),
		ID:         record.ObjectID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, blocked(
				accountingsync.SyncErrorNotFound,
				"The settlement no longer exists in Trenova",
				"Skip this record",
			)
		}
		return nil, err
	}
	if settlement.Kind == repositories.PayableDriver && !settlement.OwnerOperator {
		return nil, &noopError{
			reason: "Company driver pay belongs to the payroll system, so it is not sent",
		}
	}
	if settlement.PostedAt == nil {
		return nil, blocked(
			accountingsync.SyncErrorValidation,
			documentLabel(record.ObjectType, settlement.Number)+" was never posted",
			"Skip this record",
		)
	}
	return settlement, nil
}

type accountRefRequest struct {
	accountID pulid.ID
	lines     []repositories.PayableJournalLine
	defaults  repositories.PayableDefaultAccounts
	missing   string
}

func (s *Service) accountRef(
	ctx context.Context,
	res *resolver,
	req *accountRefRequest,
) (string, error) {
	if req.accountID.IsNil() {
		return "", blocked(
			accountingsync.SyncErrorConfiguration,
			req.missing,
			"Set the default account in the accounting settings, then retry",
		)
	}

	target := mappingTarget{
		TargetType: accountingsync.TargetGLAccount,
		ObjectID:   req.accountID,
		Label:      accountLabel(req.accountID, req.lines),
	}
	row, err := res.mapping(ctx, target)
	if err != nil {
		return "", err
	}
	if row.State == accountingsync.MappingStateConfirmed && row.ExternalID != "" {
		res.used[row.ID] = struct{}{}
		return row.ExternalID, nil
	}

	if role := roleForAccount(req.accountID, req.defaults); role != "" {
		externalID, roleErr := res.optional(ctx, mappingTarget{
			TargetType: accountingsync.TargetAccountRole,
			Key:        role,
		})
		if roleErr != nil {
			return "", roleErr
		}
		if externalID != "" {
			return externalID, nil
		}
	}
	return "", res.missing(row, target)
}

func roleForAccount(accountID pulid.ID, defaults repositories.PayableDefaultAccounts) string {
	switch accountID {
	case defaults.Payable:
		return accountingsync.AccountRoleAP
	case defaults.PurchasedTransportation:
		return accountingsync.AccountRolePurchasedTransportation
	case defaults.Cash:
		return accountingsync.AccountRoleDeposit
	default:
		return ""
	}
}

func accountLabel(accountID pulid.ID, lines []repositories.PayableJournalLine) string {
	for idx := range lines {
		if lines[idx].AccountID != accountID {
			continue
		}
		name := strings.TrimSpace(
			strings.TrimSpace(
				lines[idx].AccountCode,
			) + " " + strings.TrimSpace(
				lines[idx].AccountName,
			),
		)
		if name != "" {
			return "GL account " + name
		}
	}
	return ""
}

func (s *Service) pushBill(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	settlement, err := s.payableSettlement(ctx, sess, record)
	if err != nil {
		return nil, err
	}
	label := documentLabel(record.ObjectType, settlement.Number)
	if len(settlement.Lines) == 0 {
		if settlement.NetMinor == 0 {
			return nil, &noopError{reason: label + " has no amounts, so there is nothing to send"}
		}
		return nil, blocked(
			accountingsync.SyncErrorValidation,
			label+" has no journal entry to send",
			"Check the settlement's posting in the general ledger, then retry; or skip this record",
		)
	}
	if err = s.checkBooks(sess, settlement.CurrencyCode, *settlement.PostedAt, label); err != nil {
		return nil, err
	}
	var target *accountingsync.AccountingSyncRecord
	if record.Operation == accountingsync.SyncOperationUpdate {
		if target, err = s.updateTarget(ctx, sess, record, label); err != nil {
			return nil, err
		}
	}

	res := newResolver(s, sess)
	vendorID, err := s.partyRef(
		ctx,
		sess,
		res,
		record.ObjectType.VendorOf(),
		settlement.PartyID,
	)
	if err != nil {
		return nil, err
	}
	apAccount, err := s.accountRef(ctx, res, &accountRefRequest{
		accountID: settlement.PayableAccountID,
		lines:     settlement.Lines,
		defaults:  settlement.Defaults,
		missing:   label + " has no payable account",
	})
	if err != nil {
		return nil, err
	}

	lines, err := s.billLines(ctx, res, settlement)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, &noopError{reason: label + " has no amounts, so there is nothing to send"}
	}
	total := decimal.Zero
	for idx := range lines {
		total = total.Add(lines[idx].Amount)
	}
	credit := total.IsNegative()
	if credit {
		for idx := range lines {
			lines[idx].Amount = lines[idx].Amount.Neg()
		}
	}

	doc := &services.AccountingPurchaseDocument{
		Auth:                sess.auth,
		RequestID:           record.RequestID,
		Kind:                record.ObjectType,
		VendorCredit:        credit,
		VendorExternalID:    vendorID,
		APAccountExternalID: apAccount,
		DocNumber:           billNumber(settlement),
		TxnDate:             timeutils.FormatCalendarDate(*settlement.PostedAt, sess.loc),
		CurrencyCode:        settlement.CurrencyCode,
		PrivateNote:         billNote(record.ObjectType, settlement, sess.loc),
		Lines:               lines,
	}
	if !credit {
		doc.DueDate = timeutils.FormatCalendarDate(settlement.PayDate, sess.loc)
	}
	if limit := sess.limits.MaxDocNumberLength; limit > 0 && len([]rune(doc.DocNumber)) > limit {
		doc.DocNumber = ""
	}

	written, err := writeBill(ctx, sess, target, doc, label)
	if err != nil {
		return partial(written), err
	}
	return withProviderURL(finishedResult(sess, record, written, doc, res.mappingIDs()))
}

func (s *Service) billLines(
	ctx context.Context,
	res *resolver,
	settlement *repositories.PayableSettlement,
) ([]services.AccountingPurchaseLine, error) {
	lines := make([]services.AccountingPurchaseLine, 0, len(settlement.Lines))
	for idx := range settlement.Lines {
		line := &settlement.Lines[idx]
		if line.AccountID == settlement.PayableAccountID || line.NetMinor() == 0 {
			continue
		}
		account, err := s.accountRef(ctx, res, &accountRefRequest{
			accountID: line.AccountID,
			lines:     settlement.Lines,
			defaults:  settlement.Defaults,
		})
		if err != nil {
			return nil, err
		}
		lines = append(lines, services.AccountingPurchaseLine{
			Description:       strings.TrimSpace(line.AccountName),
			AccountExternalID: account,
			Amount:            money.DecimalFromMinor(line.NetMinor()),
		})
	}
	return lines, nil
}

func fitDocNumber(number string, limit int) string {
	number = strings.TrimSpace(number)
	if runes := []rune(number); limit > 0 && len(runes) > limit {
		return string(runes[:limit])
	}
	return number
}

func withProviderURL(result *pushResult, err error) (*pushResult, error) {
	if err != nil || result == nil || result.result == nil {
		return result, err
	}
	if url := result.refs[accountingsync.ExternalRefURL]; url != "" {
		result.result.ExternalURL = url
	}
	return result, nil
}

func billNumber(settlement *repositories.PayableSettlement) string {
	if len(settlement.InvoiceNumbers) == 1 {
		return settlement.InvoiceNumbers[0]
	}
	return settlement.Number
}

func billNote(
	objectType accountingsync.SyncObjectType,
	settlement *repositories.PayableSettlement,
	loc *time.Location,
) string {
	kind := "carrier settlement"
	if objectType.IsDriverSettlement() {
		kind = "owner-operator settlement"
	}
	parts := []string{trenovaNotePrefix + kind + " " + settlement.Number}
	if settlement.PeriodStart > 0 && settlement.PeriodEnd > 0 {
		parts = append(parts, "Period "+
			timeutils.FormatCalendarDate(settlement.PeriodStart, loc)+" to "+
			timeutils.FormatCalendarDate(settlement.PeriodEnd, loc))
	}
	switch {
	case settlement.ShipmentCount == 1:
		parts = append(parts, "1 shipment")
	case settlement.ShipmentCount > 1:
		parts = append(parts, strconv.Itoa(settlement.ShipmentCount)+" shipments")
	}
	if len(settlement.InvoiceNumbers) > 0 {
		parts = append(parts, "Carrier invoices "+strings.Join(settlement.InvoiceNumbers, ", "))
	}
	return strings.Join(parts, " · ")
}

func (s *Service) pushBillVoid(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	existing, err := s.objectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	created := documentRecord(existing)
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
	written, err := sess.writer.VoidPurchaseDocument(ctx, ref)
	if err != nil {
		return partial(written), err
	}
	return finishedResult(sess, record, written, ref, nil)
}

func (s *Service) pushBillPayment(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	settlement, err := s.payableSettlement(ctx, sess, record)
	if err != nil {
		return nil, err
	}
	billType := record.ObjectType.BillOf()
	label := documentLabel(record.ObjectType, settlement.Number)
	switch {
	case settlement.PaidAt == nil:
		return nil, blocked(
			accountingsync.SyncErrorValidation,
			documentLabel(billType, settlement.Number)+" is not marked paid",
			"Skip this record",
		)
	case settlement.NetMinor < 0:
		return nil, &noopError{
			reason: documentLabel(billType, settlement.Number) +
				" is owed to you, so it was sent as a vendor credit; record the refund in " +
				sess.providerName + " against that credit",
		}
	case settlement.NetMinor == 0:
		return nil, &noopError{reason: label + " is for nothing, so there is nothing to send"}
	}
	if err = s.checkBooks(sess, settlement.CurrencyCode, *settlement.PaidAt, label); err != nil {
		return nil, err
	}

	var target *accountingsync.AccountingSyncRecord
	if record.Operation == accountingsync.SyncOperationUpdate {
		if target, err = s.updateTarget(ctx, sess, record, label); err != nil {
			return nil, err
		}
	}

	billID, err := s.billRef(ctx, sess, billType, settlement)
	if err != nil {
		return nil, err
	}

	res := newResolver(s, sess)
	vendorID, err := s.partyRef(ctx, sess, res, record.ObjectType.VendorOf(), settlement.PartyID)
	if err != nil {
		return nil, err
	}
	bank, err := s.accountRef(ctx, res, &accountRefRequest{
		accountID: settlement.BankAccountID,
		lines:     settlement.Lines,
		defaults:  settlement.Defaults,
		missing:   label + " has no cash account to pay from",
	})
	if err != nil {
		return nil, err
	}

	doc := &services.AccountingBillPaymentDocument{
		Auth:                  sess.auth,
		RequestID:             record.RequestID,
		Kind:                  record.ObjectType,
		VendorExternalID:      vendorID,
		BankAccountExternalID: bank,
		BillExternalID:        billID,
		DocNumber: fitDocNumber(
			settlement.PaymentReference,
			sess.limits.MaxDocNumberLength,
		),
		TxnDate:      timeutils.FormatCalendarDate(*settlement.PaidAt, sess.loc),
		CurrencyCode: settlement.CurrencyCode,
		PrivateNote:  billPaymentNote(record.ObjectType, settlement),
		Amount:       money.DecimalFromMinor(settlement.NetMinor),
	}
	var written *services.AccountingDocumentResult
	if target != nil {
		doc.ExternalID = target.ExternalID
		written, err = sess.writer.UpdateBillPayment(ctx, doc)
	} else {
		written, err = sess.writer.CreateBillPayment(ctx, doc)
	}
	if err != nil {
		return partial(written), err
	}
	return finishedResult(sess, record, written, doc, res.mappingIDs())
}

func (s *Service) billRef(
	ctx context.Context,
	sess *pushSession,
	billType accountingsync.SyncObjectType,
	settlement *repositories.PayableSettlement,
) (string, error) {
	label := documentLabel(billType, settlement.Number)
	existing, err := s.objectRecords(ctx, sess, billType, settlement.ID)
	if err != nil {
		return "", err
	}
	created := documentRecord(existing)
	if created == nil {
		if !sess.conn.Covers(*settlement.PostedAt) {
			return "", &noopError{
				reason: label + " was posted before the start date, so its payment is not sent either",
			}
		}
		created, err = s.enqueueDependency(ctx, sess, &services.AccountingSyncEnqueueRequest{
			ObjectType:   billType,
			ObjectID:     settlement.ID,
			ObjectNumber: settlement.Number,
			Operation:    accountingsync.SyncOperationCreate,
			Revision:     1,
			DocumentDate: *settlement.PostedAt,
		})
		if err != nil {
			return "", err
		}
		if created == nil {
			return "", blocked(
				accountingsync.SyncErrorNotFound,
				label+" could not be queued",
				"Retry this record",
			)
		}
	}
	switch {
	case isSynced(created):
		return created.ExternalID, nil
	case created.Status.IsFinal():
		return "", &noopError{reason: label + " was not sent, so its payment is not sent either"}
	default:
		s.kick(ctx, sess.tenant, sess.conn.ID)
		return "", waitingOn(created, label)
	}
}

func billPaymentNote(
	objectType accountingsync.SyncObjectType,
	settlement *repositories.PayableSettlement,
) string {
	kind := "carrier settlement"
	if objectType.IsDriverSettlement() {
		kind = "owner-operator settlement"
	}
	note := trenovaNotePrefix + "payment of " + kind + " " + settlement.Number
	if method := strings.TrimSpace(settlement.PaymentMethod); method != "" {
		note += " · Paid by " + method
	}
	if reference := strings.TrimSpace(settlement.PaymentReference); reference != "" {
		note += " · Reference " + reference
	}
	return note
}

func sentAsCredit(record *accountingsync.AccountingSyncRecord) bool {
	return record.ExternalRefs[accountingsync.ExternalRefDocumentType] == "VendorCredit"
}

func writeBill(
	ctx context.Context,
	sess *pushSession,
	target *accountingsync.AccountingSyncRecord,
	doc *services.AccountingPurchaseDocument,
	label string,
) (*services.AccountingDocumentResult, error) {
	if target == nil {
		return sess.writer.CreatePurchaseDocument(ctx, doc)
	}
	if sentAsCredit(target) != doc.VendorCredit {
		return nil, blocked(
			accountingsync.SyncErrorValidation,
			label+" now nets to the other side of zero, so "+sess.providerName+
				" holds it as the wrong kind of document",
			"Send it again as a new document, then delete the old one in "+sess.providerName,
		)
	}
	doc.ExternalID = target.ExternalID
	return sess.writer.UpdatePurchaseDocument(ctx, doc)
}
