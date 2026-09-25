package accountingsyncservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	periodStart   = time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC).Unix()
	periodEnd     = time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC).Unix()
	settledPosted = time.Date(2026, time.April, 10, 15, 0, 0, 0, time.UTC).Unix()
	settledPayDay = time.Date(2026, time.April, 17, 12, 0, 0, 0, time.UTC).Unix()
	settledPaidAt = time.Date(2026, time.April, 18, 16, 0, 0, 0, time.UTC).Unix()
)

type payableAccounts struct {
	ap                 pulid.ID
	settlementsPayable pulid.ID
	pt                 pulid.ID
	cash               pulid.ID
	fuel               pulid.ID
	escrow             pulid.ID
	advance            pulid.ID
	driverPay          pulid.ID
}

func newPayableAccounts() payableAccounts {
	return payableAccounts{
		ap:                 pulid.MustNew("gla_"),
		settlementsPayable: pulid.MustNew("gla_"),
		pt:                 pulid.MustNew("gla_"),
		cash:               pulid.MustNew("gla_"),
		fuel:               pulid.MustNew("gla_"),
		escrow:             pulid.MustNew("gla_"),
		advance:            pulid.MustNew("gla_"),
		driverPay:          pulid.MustNew("gla_"),
	}
}

func (a payableAccounts) defaults() repositories.PayableDefaultAccounts {
	return repositories.PayableDefaultAccounts{
		Payable:                 a.ap,
		PurchasedTransportation: a.pt,
		Cash:                    a.cash,
	}
}

func glLine(
	accountID pulid.ID,
	code, name string,
	debitMinor, creditMinor int64,
) repositories.PayableJournalLine {
	return repositories.PayableJournalLine{
		AccountID:   accountID,
		AccountCode: code,
		AccountName: name,
		DebitMinor:  debitMinor,
		CreditMinor: creditMinor,
	}
}

func (h *harness) carrierSettlement(accts payableAccounts) *repositories.PayableSettlement {
	posted := settledPosted
	settlement := &repositories.PayableSettlement{
		Kind:             repositories.PayableCarrier,
		ID:               pulid.MustNew("carstl_"),
		Number:           "CS-1042",
		PartyID:          pulid.MustNew("car_"),
		PartyName:        "Swift Haulers",
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		PayDate:          settledPayDay,
		PostedAt:         &posted,
		NetMinor:         165000,
		ShipmentCount:    3,
		CurrencyCode:     "USD",
		PaymentMethod:    "ACH",
		PaymentReference: "CHK-5521",
		PayableAccountID: accts.ap,
		BankAccountID:    accts.cash,
		Defaults:         accts.defaults(),
		Lines: []repositories.PayableJournalLine{
			glLine(accts.pt, "5000", "Purchased transportation", 150000, 0),
			glLine(accts.fuel, "5010", "Fuel surcharge expense", 20000, 0),
			glLine(accts.escrow, "2150", "Escrow liability", 0, 5000),
			glLine(accts.ap, "2000", "Accounts payable", 0, 165000),
		},
		InvoiceNumbers: []string{"SWH-88121"},
	}
	h.payables.put(h.tenant, settlement)
	return settlement
}

func (h *harness) driverSettlement(accts payableAccounts) *repositories.PayableSettlement {
	posted := settledPosted
	settlement := &repositories.PayableSettlement{
		Kind:             repositories.PayableDriver,
		ID:               pulid.MustNew("dstl_"),
		Number:           "DS-2201",
		PartyID:          pulid.MustNew("wrk_"),
		PartyName:        "Dana Ortiz",
		OwnerOperator:    true,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		PayDate:          settledPayDay,
		PostedAt:         &posted,
		NetMinor:         210000,
		ShipmentCount:    1,
		CurrencyCode:     "USD",
		PaymentMethod:    "Direct deposit",
		PaymentReference: "DD-778",
		PayableAccountID: accts.settlementsPayable,
		BankAccountID:    accts.cash,
		Defaults:         accts.defaults(),
		Lines: []repositories.PayableJournalLine{
			glLine(accts.driverPay, "5100", "Owner-operator pay", 250000, 0),
			glLine(accts.advance, "1250", "Fuel advances", 0, 30000),
			glLine(accts.escrow, "2150", "Escrow liability", 0, 10000),
			glLine(accts.settlementsPayable, "2100", "Settlements payable", 0, 210000),
		},
	}
	h.payables.put(h.tenant, settlement)
	return settlement
}

type payableMappings struct {
	vendor  *accountingsync.AccountingMapping
	apRole  *accountingsync.AccountingMapping
	ptRole  *accountingsync.AccountingMapping
	deposit *accountingsync.AccountingMapping
	fuel    *accountingsync.AccountingMapping
	escrow  *accountingsync.AccountingMapping
}

func carrierVendorTarget(carrierID pulid.ID) mappingTarget {
	return mappingTarget{
		TargetType: accountingsync.TargetCarrier,
		ObjectID:   carrierID,
		Label:      "Swift Haulers",
	}
}

func driverVendorTarget(workerID pulid.ID) mappingTarget {
	return mappingTarget{
		TargetType: accountingsync.TargetDriver,
		ObjectID:   workerID,
		Label:      "Dana Ortiz",
	}
}

func roleTarget(key, label string) mappingTarget {
	return mappingTarget{TargetType: accountingsync.TargetAccountRole, Key: key, Label: label}
}

func glTarget(accountID pulid.ID, label string) mappingTarget {
	return mappingTarget{TargetType: accountingsync.TargetGLAccount, ObjectID: accountID, Label: label}
}

func (h *harness) mapPayableRoles() payableMappings {
	return payableMappings{
		apRole: h.confirm(roleTarget(accountingsync.AccountRoleAP, "Accounts payable"), "qb-ap"),
		ptRole: h.confirm(
			roleTarget(accountingsync.AccountRolePurchasedTransportation, "Purchased transportation"),
			"qb-pt",
		),
		deposit: h.confirm(roleTarget(accountingsync.AccountRoleDeposit, "Deposit account"), "qb-bank"),
	}
}

func (h *harness) mapCarrierBill(
	accts payableAccounts,
	settlement *repositories.PayableSettlement,
) payableMappings {
	mapped := h.mapPayableRoles()
	mapped.vendor = h.confirm(carrierVendorTarget(settlement.PartyID), "qb-vendor-swift")
	mapped.fuel = h.confirm(glTarget(accts.fuel, "GL account 5010 Fuel surcharge expense"), "qb-fuel")
	mapped.escrow = h.confirm(glTarget(accts.escrow, "GL account 2150 Escrow liability"), "qb-escrow")
	return mapped
}

func settlementSource(
	objectType accountingsync.SyncObjectType,
	operation accountingsync.SyncOperation,
) accountingsync.SyncSourceEvent {
	driver := objectType.IsDriverSettlement()
	switch {
	case objectType.IsBillPayment() && driver:
		return accountingsync.SyncSourceDriverSettlementPaid
	case objectType.IsBillPayment():
		return accountingsync.SyncSourceCarrierSettlementPaid
	case operation == accountingsync.SyncOperationVoid && driver:
		return accountingsync.SyncSourceDriverSettlementVoided
	case operation == accountingsync.SyncOperationVoid:
		return accountingsync.SyncSourceCarrierSettlementVoided
	case driver:
		return accountingsync.SyncSourceDriverSettlementPosted
	default:
		return accountingsync.SyncSourceCarrierSettlementPosted
	}
}

func settlementRequest(
	h *harness,
	objectType accountingsync.SyncObjectType,
	operation accountingsync.SyncOperation,
	settlement *repositories.PayableSettlement,
) *services.AccountingSyncEnqueueRequest {
	return services.SettlementSync(&services.SettlementSyncRequest{
		TenantInfo:  h.tenant,
		ObjectType:  objectType,
		ObjectID:    settlement.ID,
		Number:      settlement.Number,
		Operation:   operation,
		SourceEvent: settlementSource(objectType, operation),
		PostedAt:    settlement.PostedAt,
	})
}

func (h *harness) enqueueSettlement(
	t *testing.T,
	objectType accountingsync.SyncObjectType,
	operation accountingsync.SyncOperation,
	settlement *repositories.PayableSettlement,
) *accountingsync.AccountingSyncRecord {
	t.Helper()
	req := settlementRequest(h, objectType, operation, settlement)
	require.NotNil(t, req)
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), req))
	return h.only(t, objectType, settlement.ID, operation)
}

func (h *harness) syncedSettlementRecord(
	objectType accountingsync.SyncObjectType,
	settlement *repositories.PayableSettlement,
	externalID string,
	refs map[string]string,
) *accountingsync.AccountingSyncRecord {
	record := NewRecordFor(
		h.conn,
		settlementRequest(h, objectType, accountingsync.SyncOperationCreate, settlement),
		*settlement.PostedAt,
	)
	record.MarkSynced(&accountingsync.SyncResult{ExternalID: externalID, ExternalRefs: refs}, *settlement.PostedAt)
	h.records.put(record)
	return record
}

func (h *harness) sendDriverSettlements(at int64) {
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		enabled := at
		conn.DriverSettlementsEnabledAt = &enabled
	})
}

func purchaseDocs(t *testing.T, h *harness) []*services.AccountingPurchaseDocument {
	t.Helper()
	calls := h.writer.callsTo("CreatePurchaseDocument")
	out := make([]*services.AccountingPurchaseDocument, 0, len(calls))
	for _, call := range calls {
		doc, ok := call.doc.(*services.AccountingPurchaseDocument)
		require.True(t, ok)
		out = append(out, doc)
	}
	return out
}

func onlyPurchaseDoc(t *testing.T, h *harness) *services.AccountingPurchaseDocument {
	t.Helper()
	docs := purchaseDocs(t, h)
	require.Len(t, docs, 1)
	return docs[0]
}

func onlyBillPayment(t *testing.T, h *harness) *services.AccountingBillPaymentDocument {
	t.Helper()
	calls := h.writer.callsTo("CreateBillPayment")
	require.Len(t, calls, 1)
	doc, ok := calls[0].doc.(*services.AccountingBillPaymentDocument)
	require.True(t, ok)
	return doc
}

type purchaseLineWant struct {
	account     string
	amount      string
	description string
}

func assertPurchaseLines(t *testing.T, want []purchaseLineWant, got []services.AccountingPurchaseLine) {
	t.Helper()
	require.Len(t, got, len(want))
	for idx := range want {
		assert.Equal(t, want[idx].account, got[idx].AccountExternalID, "line %d account", idx)
		decimalEqual(t, want[idx].amount, got[idx].Amount)
		assert.Equal(t, want[idx].description, got[idx].Description, "line %d description", idx)
	}
}

func assertNothingSent(t *testing.T, h *harness, record *accountingsync.AccountingSyncRecord, reason string) {
	t.Helper()
	got := h.records.get(record.ID)
	assert.True(t, got.Status.IsFinal(), "status %s", got.Status)
	assert.NotEqual(t, accountingsync.SyncStatusSuperseded, got.Status)
	assert.Empty(t, got.ExternalID)
	assert.Contains(t, got.Resolution, reason)
	for _, method := range []string{
		"CreatePurchaseDocument", "CreateBillPayment", "VoidPurchaseDocument", "UpsertVendor",
	} {
		assert.Empty(t, h.writer.callsTo(method), "%s is not called", method)
	}
}

func TestCarrierBillIsBuiltFromTheJournalEntryWithoutThePayableLine(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	mapped := h.mapCarrierBill(accts, settlement)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	result := h.drain(t)

	assert.Equal(t, 1, result.Synced)
	doc := onlyPurchaseDoc(t, h)
	assert.Equal(t, accountingsync.SyncObjectCarrierBill, doc.Kind)
	assert.False(t, doc.VendorCredit)
	assert.Equal(t, record.RequestID, doc.RequestID)
	assert.Equal(t, testAccessToken, doc.Auth.AccessToken)
	assert.Equal(t, "qb-vendor-swift", doc.VendorExternalID)
	assert.Equal(t, "qb-ap", doc.APAccountExternalID,
		"the payable line's account is the bill's AP account, through the AP role")
	assert.Equal(t, "2026-04-10", doc.TxnDate, "the bill date is the journal's accounting date")
	assert.Equal(t, "2026-04-17", doc.DueDate, "the due date is the settlement's pay date")
	assert.Equal(t, "USD", doc.CurrencyCode)
	assert.Equal(t, "SWH-88121", doc.DocNumber, "a single matched carrier invoice number is the bill number")
	assertPurchaseLines(t, []purchaseLineWant{
		{account: "qb-pt", amount: "1500", description: "Purchased transportation"},
		{account: "qb-fuel", amount: "200", description: "Fuel surcharge expense"},
		{account: "qb-escrow", amount: "-50", description: "Escrow liability"},
	}, doc.Lines)
	assert.Equal(t,
		"Trenova carrier settlement CS-1042 · Period 2026-04-01 to 2026-04-07 · 3 shipments · "+
			"Carrier invoices SWH-88121",
		doc.PrivateNote)

	synced := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, synced.Status)
	assert.Equal(t, "qb-CreatePurchaseDocument-1", synced.ExternalID)
	assert.Equal(t, "https://qbo.test/CarrierBill/qb-CreatePurchaseDocument-1", synced.ExternalURL)
	assert.ElementsMatch(t, []string{
		mapped.vendor.ID.String(),
		mapped.apRole.ID.String(),
		mapped.ptRole.ID.String(),
		mapped.fuel.ID.String(),
		mapped.escrow.ID.String(),
	}, synced.MappingIDs)
	assert.NotContains(t, synced.MappingIDs, mapped.deposit.ID.String(), "a mapping the bill did not use is not named")
	assert.Equal(t, "SWH-88121", synced.Payload["DocNumber"])

	for _, accountID := range []pulid.ID{accts.ap, accts.pt} {
		row, err := fakeMappingRepo{store: h.mappings}.GetByTarget(t.Context(),
			&repositories.GetAccountingMappingByTargetRequest{
				TenantInfo:      h.tenant,
				ConnectionID:    h.conn.ID,
				TargetType:      accountingsync.TargetGLAccount,
				TrenovaObjectID: accountID,
			})
		require.NoError(t, err, "every account the bill touches gets a GL account mapping row to review")
		assert.Equal(t, accountingsync.MappingStateUnmatched, row.State)
	}

	requests := h.payables.requests()
	require.NotEmpty(t, requests)
	assert.Equal(t, repositories.PayableCarrier, requests[0].Kind)
	assert.Equal(t, settlement.ID, requests[0].ID)
	assert.Equal(t, h.tenant, requests[0].TenantInfo)
}

func TestBillDatesAreCalendarDatesInTheOrganizationsTimezone(t *testing.T) {
	t.Parallel()

	h := newHarness(t, withTimezone("America/Chicago"))
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	posted := time.Date(2026, time.April, 10, 3, 0, 0, 0, time.UTC).Unix()
	settlement.PostedAt = &posted
	settlement.PayDate = time.Date(2026, time.April, 17, 2, 0, 0, 0, time.UTC).Unix()
	h.mapCarrierBill(accts, settlement)
	h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	doc := onlyPurchaseDoc(t, h)
	assert.Equal(t, "2026-04-09", doc.TxnDate)
	assert.Equal(t, "2026-04-16", doc.DueDate)
}

func TestNegativeNetCarrierSettlementIsAVendorCreditWithEverySignFlipped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	settlement.NetMinor = -20000
	settlement.Lines = []repositories.PayableJournalLine{
		glLine(accts.pt, "5000", "Purchased transportation", 10000, 0),
		glLine(accts.fuel, "5010", "Fuel surcharge expense", 2500, 0),
		glLine(accts.escrow, "2150", "Escrow liability", 0, 32500),
		glLine(accts.ap, "2000", "Accounts payable", 20000, 0),
	}
	h.mapCarrierBill(accts, settlement)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	doc := onlyPurchaseDoc(t, h)
	assert.True(t, doc.VendorCredit, "a negative net is a vendor credit")
	assert.Equal(t, accountingsync.SyncObjectCarrierBill, doc.Kind)
	assert.Empty(t, doc.DueDate, "a vendor credit has no due date")
	assert.Equal(t, "2026-04-10", doc.TxnDate)
	assert.Equal(t, "qb-ap", doc.APAccountExternalID)
	assertPurchaseLines(t, []purchaseLineWant{
		{account: "qb-pt", amount: "-100", description: "Purchased transportation"},
		{account: "qb-fuel", amount: "-25", description: "Fuel surcharge expense"},
		{account: "qb-escrow", amount: "325", description: "Escrow liability"},
	}, doc.Lines)
	total := doc.Lines[0].Amount.Add(doc.Lines[1].Amount).Add(doc.Lines[2].Amount)
	decimalEqual(t, "200", total)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
}

func TestZeroNetSettlementStillCreatesTheBill(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	settlement.NetMinor = 0
	settlement.Lines = []repositories.PayableJournalLine{
		glLine(accts.pt, "5000", "Purchased transportation", 40000, 0),
		glLine(accts.escrow, "2150", "Escrow liability", 0, 40000),
	}
	h.mapCarrierBill(accts, settlement)
	h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	doc := onlyPurchaseDoc(t, h)
	assert.False(t, doc.VendorCredit, "a zero net is a bill, not a vendor credit")
	assert.Equal(t, "2026-04-17", doc.DueDate)
	assertPurchaseLines(t, []purchaseLineWant{
		{account: "qb-pt", amount: "400", description: "Purchased transportation"},
		{account: "qb-escrow", amount: "-400", description: "Escrow liability"},
	}, doc.Lines)
}

func TestBillNumberRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		number     string
		invoices   []string
		wantNumber string
		wantNote   []string
	}{
		{
			name:       "no matched invoice uses the settlement number",
			number:     "CS-1042",
			wantNumber: "CS-1042",
			wantNote:   []string{"Trenova carrier settlement CS-1042"},
		},
		{
			name:       "two matched invoices use the settlement number and list both",
			number:     "CS-1042",
			invoices:   []string{"SWH-88121", "SWH-88122"},
			wantNumber: "CS-1042",
			wantNote:   []string{"Carrier invoices SWH-88121, SWH-88122"},
		},
		{
			name:       "a settlement number longer than the provider allows is left for the provider",
			number:     "CS-2026-0000000000001042",
			wantNumber: "",
			wantNote:   []string{"Trenova carrier settlement CS-2026-0000000000001042"},
		},
		{
			name:       "a single invoice number longer than the provider allows goes in the note",
			number:     "CS-1042",
			invoices:   []string{"SWIFT-HAULERS-INV-0000088121"},
			wantNumber: "",
			wantNote:   []string{"Carrier invoices SWIFT-HAULERS-INV-0000088121"},
		},
		{
			name:       "a number exactly at the limit is kept",
			number:     "CS-2026-0000000001042",
			wantNumber: "CS-2026-0000000001042",
			wantNote:   []string{"CS-2026-0000000001042"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			accts := newPayableAccounts()
			settlement := h.carrierSettlement(accts)
			settlement.Number = tc.number
			settlement.InvoiceNumbers = tc.invoices
			h.mapCarrierBill(accts, settlement)
			h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

			h.drain(t)

			doc := onlyPurchaseDoc(t, h)
			assert.Equal(t, tc.wantNumber, doc.DocNumber)
			for _, part := range tc.wantNote {
				assert.Contains(t, doc.PrivateNote, part)
			}
		})
	}
}

func TestBillURLComesFromTheProviderWhenItReturnsOne(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.writer.extraRefs["CreatePurchaseDocument"] = map[string]string{
		accountingsync.ExternalRefURL:          "https://app.qbo.test/app/vendorcredit?txnId=91",
		accountingsync.ExternalRefDocumentType: "VendorCredit",
	}
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	synced := h.records.get(record.ID)
	assert.Equal(t, "https://app.qbo.test/app/vendorcredit?txnId=91", synced.ExternalURL,
		"the adapter's URL names the right document when a bill became a vendor credit")
	assert.Equal(t, "VendorCredit", synced.ExternalRefs[accountingsync.ExternalRefDocumentType])
}

func TestSettlementWithNothingToSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(accts payableAccounts, s *repositories.PayableSettlement)
		reason string
	}{
		{
			name: "no journal lines and no amount",
			mutate: func(_ payableAccounts, s *repositories.PayableSettlement) {
				s.Lines = nil
				s.NetMinor = 0
			},
			reason: "has no amounts",
		},
		{
			name: "only the payable line",
			mutate: func(accts payableAccounts, s *repositories.PayableSettlement) {
				s.Lines = []repositories.PayableJournalLine{glLine(accts.ap, "2000", "Accounts payable", 0, 0)}
				s.NetMinor = 0
			},
			reason: "has no amounts",
		},
		{
			name: "every line nets to zero",
			mutate: func(accts payableAccounts, s *repositories.PayableSettlement) {
				s.Lines = []repositories.PayableJournalLine{
					glLine(accts.pt, "5000", "Purchased transportation", 5000, 5000),
					glLine(accts.ap, "2000", "Accounts payable", 0, 0),
				}
				s.NetMinor = 0
			},
			reason: "has no amounts",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			accts := newPayableAccounts()
			settlement := h.carrierSettlement(accts)
			tc.mutate(accts, settlement)
			h.mapCarrierBill(accts, settlement)
			record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill,
				accountingsync.SyncOperationCreate, settlement)

			result := h.drain(t)

			assert.Equal(t, 1, result.Synced)
			assertNothingSent(t, h, record, tc.reason)
		})
	}
}

func TestSettlementThatCannotBeSentBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(s *repositories.PayableSettlement)
		remove   bool
		category accountingsync.SyncErrorCategory
		message  string
	}{
		{
			name: "an amount but no journal entry",
			mutate: func(s *repositories.PayableSettlement) {
				s.Lines = nil
			},
			category: accountingsync.SyncErrorValidation,
			message:  "Carrier settlement CS-1042 has no journal entry to send",
		},
		{
			name: "never posted",
			mutate: func(s *repositories.PayableSettlement) {
				s.PostedAt = nil
			},
			category: accountingsync.SyncErrorValidation,
			message:  "never posted",
		},
		{
			name: "no payable account",
			mutate: func(s *repositories.PayableSettlement) {
				s.PayableAccountID = pulid.Nil
			},
			category: accountingsync.SyncErrorConfiguration,
			message:  "Carrier settlement CS-1042 has no payable account",
		},
		{
			name:     "deleted in Trenova",
			remove:   true,
			category: accountingsync.SyncErrorNotFound,
			message:  "no longer exists",
		},
		{
			name: "in a currency the books do not keep",
			mutate: func(s *repositories.PayableSettlement) {
				s.CurrencyCode = "CAD"
			},
			category: accountingsync.SyncErrorCurrency,
			message:  "CAD",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			accts := newPayableAccounts()
			settlement := h.carrierSettlement(accts)
			h.mapCarrierBill(accts, settlement)
			record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill,
				accountingsync.SyncOperationCreate, settlement)
			if tc.mutate != nil {
				tc.mutate(settlement)
			}
			if tc.remove {
				h.payables.mu.Lock()
				delete(h.payables.rows, settlement.ID)
				h.payables.mu.Unlock()
			}

			result := h.drain(t)

			assert.Equal(t, 1, result.Blocked)
			blocked := h.records.get(record.ID)
			assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
			assert.Equal(t, tc.category, blocked.ErrorCategory)
			assert.Contains(t, blocked.ErrorMessage, tc.message)
			assert.Empty(t, h.writer.callsTo("CreatePurchaseDocument"))
		})
	}
}

func TestConfirmedGLAccountMappingWinsOverTheRole(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	mapped := h.mapCarrierBill(accts, settlement)
	ptAccount := h.confirm(glTarget(accts.pt, "GL account 5000 Purchased transportation"), "qb-gl-5000")
	apAccount := h.confirm(glTarget(accts.ap, "GL account 2000 Accounts payable"), "qb-gl-2000")
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	doc := onlyPurchaseDoc(t, h)
	assert.Equal(t, "qb-gl-2000", doc.APAccountExternalID)
	assert.Equal(t, "qb-gl-5000", doc.Lines[0].AccountExternalID)
	synced := h.records.get(record.ID)
	assert.Contains(t, synced.MappingIDs, ptAccount.ID.String())
	assert.Contains(t, synced.MappingIDs, apAccount.ID.String())
	assert.NotContains(t, synced.MappingIDs, mapped.ptRole.ID.String(), "the role was not used")
	assert.NotContains(t, synced.MappingIDs, mapped.apRole.ID.String(), "the role was not used")
}

func TestUnconfirmedGLAccountMappingFallsBackToTheRole(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	h.mappings.set(h.conn, glTarget(accts.pt, "GL account 5000 Purchased transportation"),
		accountingsync.MappingStateProposed, "qb-proposed")
	h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)
	h.drain(t)

	assert.Equal(t, "qb-pt", onlyPurchaseDoc(t, h).Lines[0].AccountExternalID,
		"a proposed GL account mapping is not used; the default account maps through its role")
}

func TestUnmappedOverrideAccountBlocksNamingTheAccount(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	h.mapPayableRoles()
	h.confirm(carrierVendorTarget(settlement.PartyID), "qb-vendor-swift")
	h.confirm(glTarget(accts.fuel, "GL account 5010 Fuel surcharge expense"), "qb-fuel")
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	result := h.drain(t)

	assert.Equal(t, 1, result.Blocked)
	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Equal(t, "GL account 2150 Escrow liability is not mapped", blocked.ErrorMessage)
	assert.Equal(t, "Map GL account 2150 Escrow liability to a QuickBooks Online account", blocked.Resolution,
		"an override account is never filed under a role silently")
	escrow, err := fakeMappingRepo{store: h.mappings}.GetByTarget(t.Context(),
		&repositories.GetAccountingMappingByTargetRequest{
			TenantInfo:      h.tenant,
			ConnectionID:    h.conn.ID,
			TargetType:      accountingsync.TargetGLAccount,
			TrenovaObjectID: accts.escrow,
		})
	require.NoError(t, err)
	assert.Equal(t, escrow.ID.String(), blocked.ErrorCode, "the block names the mapping row to fill in")
	assert.Empty(t, h.writer.callsTo("CreatePurchaseDocument"))
}

func TestDefaultAccountWithoutItsRoleMappingBlocks(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	h.confirm(carrierVendorTarget(settlement.PartyID), "qb-vendor-swift")
	h.confirm(roleTarget(accountingsync.AccountRoleAP, "Accounts payable"), "qb-ap")
	h.confirm(glTarget(accts.fuel, "GL account 5010 Fuel surcharge expense"), "qb-fuel")
	h.confirm(glTarget(accts.escrow, "GL account 2150 Escrow liability"), "qb-escrow")
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Equal(t, "Map GL account 5000 Purchased transportation to a QuickBooks Online account",
		blocked.Resolution)
}

func TestAPRoleDoesNotCoverANonDefaultPayableAccount(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	override := pulid.MustNew("gla_")
	settlement.PayableAccountID = override
	settlement.Lines[3] = glLine(override, "2010", "Carrier payables", 0, 165000)
	h.mapCarrierBill(accts, settlement)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Equal(t, "Map GL account 2010 Carrier payables to a QuickBooks Online account", blocked.Resolution)
}

func TestUnmappedCarrierQueuesAVendorCreateAndTheBillWaits(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	mapped := h.mapCarrierBill(accts, settlement)
	vendorRow := h.mappings.set(h.conn, carrierVendorTarget(settlement.PartyID),
		accountingsync.MappingStateUnmatched, "")
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	first := h.drain(t)

	assert.Equal(t, 1, first.Waiting)
	assert.Empty(t, h.writer.callsTo("CreatePurchaseDocument"))
	dependency := h.only(t, accountingsync.SyncObjectCarrierVendor, settlement.PartyID,
		accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncSourceDependencyOf, dependency.SourceEvent)
	assert.Equal(t, accountingsync.SyncStatusQueued, dependency.Status)
	assert.Equal(t, "Swift Haulers", dependency.ObjectNumber)
	waiting := h.records.get(record.ID)
	assert.Equal(t, dependency.ID, waiting.DependsOnRecordID)
	assert.Contains(t, waiting.ErrorMessage, "Carrier Swift Haulers")
	assert.Equal(t, 0, waiting.AttemptCount)
	assert.Contains(t, h.dispatcher.kicked(), h.conn.ID)

	second := h.drain(t)

	assert.Equal(t, 1, second.Synced)
	require.Len(t, h.mappings.created, 1)
	assert.Equal(t, vendorRow.ID, h.mappings.created[0].MappingID)
	assert.Equal(t, accountingsync.MappingSourceCreatedInProvider, h.mappings.created[0].Source)
	assert.Equal(t, h.conn.ConnectedByID, h.mappings.created[0].UserID)
	assert.Empty(t, h.writer.callsTo("UpsertVendor"), "a vendor is created through the mapping service")
	created := h.records.get(dependency.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, created.Status)
	assert.Equal(t, "qb-created-"+settlement.PartyID.String(), created.ExternalID)
	assert.Equal(t, []string{vendorRow.ID.String()}, created.MappingIDs)

	h.records.makeDue(record.ID)
	third := h.drain(t)

	assert.Equal(t, 1, third.Synced)
	assert.Equal(t, "qb-created-"+settlement.PartyID.String(), onlyPurchaseDoc(t, h).VendorExternalID)
	assert.Contains(t, h.records.get(record.ID).MappingIDs, vendorRow.ID.String())
	assert.NotContains(t, h.records.get(record.ID).MappingIDs, mapped.deposit.ID.String())
}

func TestDuplicateVendorNameBlocksAsMapping(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	h.mappings.set(h.conn, carrierVendorTarget(settlement.PartyID), accountingsync.MappingStateUnmatched, "")
	h.mappings.createErr = errortypes.NewBusinessError(
		"{0} already exists in the accounting system; map {1} to it instead",
		"Swift Haulers LLC (Vendor)",
		"Swift Haulers",
	).WithInternal(providerFault(accountingsync.SyncErrorDuplicate, "Duplicate Name Exists Error", ""))
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)
	h.drain(t)

	dependency := h.only(t, accountingsync.SyncObjectCarrierVendor, settlement.PartyID,
		accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncStatusBlocked, dependency.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, dependency.ErrorCategory)
	assert.Equal(t, "Map Swift Haulers to the existing QuickBooks Online vendor, then retry", dependency.Resolution)

	h.records.makeDue(record.ID)
	h.drain(t)

	held := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusQueued, held.Status)
	assert.Equal(t, dependency.ID, held.DependsOnRecordID)
	assert.Contains(t, held.Resolution, "Carrier Swift Haulers")
	nearNow(t, accountingsync.SyncAuthWait, held.NextAttemptAt)
	assert.Empty(t, h.writer.callsTo("CreatePurchaseDocument"))
}

func TestVendorCreateThatFailsValidationBlocksAsValidation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sendDriverSettlements(syncEnabledAt)
	workerID := pulid.MustNew("wrk_")
	h.mappings.set(h.conn, driverVendorTarget(workerID), accountingsync.MappingStateUnmatched, "")
	h.mappings.createErr = errortypes.NewBusinessError("The driver has no legal name")
	h.enqueue(t, &services.AccountingSyncEnqueueRequest{
		ObjectType:   accountingsync.SyncObjectDriverVendor,
		ObjectID:     workerID,
		ObjectNumber: "Dana Ortiz",
		Operation:    accountingsync.SyncOperationCreate,
		Revision:     1,
		SourceEvent:  accountingsync.SyncSourceDependencyOf,
	})

	h.drain(t)

	record := h.only(t, accountingsync.SyncObjectDriverVendor, workerID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncStatusBlocked, record.Status)
	assert.Equal(t, accountingsync.SyncErrorValidation, record.ErrorCategory)
	assert.Contains(t, record.Resolution, "Correct the driver in Trenova")
}

func TestVendorCreateWithAConfirmedMappingSendsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	carrierID := pulid.MustNew("car_")
	mapping := h.confirm(carrierVendorTarget(carrierID), "qb-vendor-swift")
	h.enqueue(t, &services.AccountingSyncEnqueueRequest{
		ObjectType:  accountingsync.SyncObjectCarrierVendor,
		ObjectID:    carrierID,
		Operation:   accountingsync.SyncOperationCreate,
		Revision:    1,
		SourceEvent: accountingsync.SyncSourceDependencyOf,
	})

	h.drain(t)

	record := h.only(t, accountingsync.SyncObjectCarrierVendor, carrierID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncStatusSynced, record.Status)
	assert.Equal(t, "qb-vendor-swift", record.ExternalID)
	assert.Equal(t, []string{mapping.ID.String()}, record.MappingIDs)
	assert.Empty(t, h.mappings.created, "a vendor mapped since the record was queued is not created twice")
	assert.Empty(t, h.writer.methods())
}

func TestVendorUpdatePushesThePartyToTheMappedVendor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectType accountingsync.SyncObjectType
		target     func(pulid.ID) mappingTarget
		prefix     string
	}{
		{
			name:       "carrier",
			objectType: accountingsync.SyncObjectCarrierVendor,
			target:     carrierVendorTarget,
			prefix:     "car_",
		},
		{
			name:       "owner-operator",
			objectType: accountingsync.SyncObjectDriverVendor,
			target:     driverVendorTarget,
			prefix:     "wrk_",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			h.sendDriverSettlements(syncEnabledAt)
			partyID := pulid.MustNew(tc.prefix)
			target := tc.target(partyID)
			mapping := h.confirm(target, "qb-vendor-77")
			h.mappings.parties[partyID] = &services.AccountingPartyDraft{DisplayName: target.Label, City: "Reno"}
			require.NoError(t, h.enqueuer.Enqueue(t.Context(),
				services.VendorSyncRequest(h.tenant, tc.objectType, partyID, target.Label, 4)))

			h.drain(t)

			calls := h.writer.callsTo("UpsertVendor")
			require.Len(t, calls, 1)
			doc, ok := calls[0].doc.(*services.AccountingVendorDocument)
			require.True(t, ok)
			assert.Equal(t, "qb-vendor-77", doc.ExternalID)
			assert.Equal(t, "Reno", doc.Party.City)
			assert.Equal(t, target.Label, doc.Party.DisplayName)
			require.Len(t, h.mappings.vendorCalls, 1)
			assert.Equal(t, target.TargetType, h.mappings.vendorCalls[0].targetType)
			assert.Equal(t, partyID, h.mappings.vendorCalls[0].objectID)
			record := h.only(t, tc.objectType, partyID, accountingsync.SyncOperationUpdate)
			assert.Equal(t, record.RequestID, doc.RequestID)
			assert.Equal(t, accountingsync.SyncStatusSynced, record.Status)
			assert.Equal(t, []string{mapping.ID.String()}, record.MappingIDs)
			assert.Empty(t, h.writer.callsTo("UpsertCustomer"))
		})
	}
}

func TestVendorUpdateWithoutAMappingSendsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	carrierID := pulid.MustNew("car_")
	h.confirm(carrierVendorTarget(carrierID), "qb-vendor-swift")
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.VendorSyncRequest(
		h.tenant, accountingsync.SyncObjectCarrierVendor, carrierID, "Swift Haulers", 2,
	)))
	record := h.only(t, accountingsync.SyncObjectCarrierVendor, carrierID, accountingsync.SyncOperationUpdate)
	h.mappings.set(h.conn, carrierVendorTarget(carrierID), accountingsync.MappingStateUnmatched, "")

	h.drain(t)

	assertNothingSent(t, h, record, "The carrier is not mapped, so there is nothing to update")
	assert.Empty(t, h.mappings.vendorCalls)
}

func TestOwnerOperatorBillUsesTheDriverVendorAndItsPayableAccount(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sendDriverSettlements(syncEnabledAt)
	accts := newPayableAccounts()
	settlement := h.driverSettlement(accts)
	h.mapPayableRoles()
	vendor := h.confirm(driverVendorTarget(settlement.PartyID), "qb-vendor-dana")
	h.confirm(glTarget(accts.settlementsPayable, "GL account 2100 Settlements payable"), "qb-settlements")
	h.confirm(glTarget(accts.driverPay, "GL account 5100 Owner-operator pay"), "qb-oo-pay")
	h.confirm(glTarget(accts.advance, "GL account 1250 Fuel advances"), "qb-advances")
	h.confirm(glTarget(accts.escrow, "GL account 2150 Escrow liability"), "qb-escrow")
	record := h.enqueueSettlement(t, accountingsync.SyncObjectDriverBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	doc := onlyPurchaseDoc(t, h)
	assert.Equal(t, accountingsync.SyncObjectDriverBill, doc.Kind)
	assert.Equal(t, "qb-vendor-dana", doc.VendorExternalID)
	assert.Equal(t, "qb-settlements", doc.APAccountExternalID)
	assert.Equal(t, "DS-2201", doc.DocNumber)
	assertPurchaseLines(t, []purchaseLineWant{
		{account: "qb-oo-pay", amount: "2500", description: "Owner-operator pay"},
		{account: "qb-advances", amount: "-300", description: "Fuel advances"},
		{account: "qb-escrow", amount: "-100", description: "Escrow liability"},
	}, doc.Lines)
	assert.Equal(t, "Trenova owner-operator settlement DS-2201 · Period 2026-04-01 to 2026-04-07 · 1 shipment",
		doc.PrivateNote)
	synced := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, synced.Status)
	assert.Contains(t, synced.MappingIDs, vendor.ID.String())
	requests := h.payables.requests()
	require.NotEmpty(t, requests)
	assert.Equal(t, repositories.PayableDriver, requests[0].Kind)
}

func TestUnmappedOwnerOperatorQueuesADriverVendorCreate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sendDriverSettlements(syncEnabledAt)
	accts := newPayableAccounts()
	settlement := h.driverSettlement(accts)
	h.mapPayableRoles()
	driverRow := h.mappings.set(h.conn, driverVendorTarget(settlement.PartyID),
		accountingsync.MappingStateUnmatched, "")
	record := h.enqueueSettlement(t, accountingsync.SyncObjectDriverBill, accountingsync.SyncOperationCreate, settlement)

	first := h.drain(t)

	assert.Equal(t, 1, first.Waiting)
	dependency := h.only(t, accountingsync.SyncObjectDriverVendor, settlement.PartyID,
		accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncSourceDependencyOf, dependency.SourceEvent)
	assert.Equal(t, dependency.ID, h.records.get(record.ID).DependsOnRecordID)
	assert.Contains(t, h.records.get(record.ID).ErrorMessage, "Driver Dana Ortiz")
	assert.Empty(t, h.records.find(accountingsync.SyncObjectCarrierVendor, settlement.PartyID,
		accountingsync.SyncOperationCreate))

	h.drain(t)

	require.Len(t, h.mappings.created, 1)
	assert.Equal(t, driverRow.ID, h.mappings.created[0].MappingID)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(dependency.ID).Status)
}

func TestCompanyDriverSettlementIsNeverSent(t *testing.T) {
	t.Parallel()

	for _, objectType := range []accountingsync.SyncObjectType{
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay,
	} {
		t.Run(string(objectType), func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			h.sendDriverSettlements(syncEnabledAt)
			accts := newPayableAccounts()
			settlement := h.driverSettlement(accts)
			settlement.OwnerOperator = false
			paid := settledPaidAt
			settlement.PaidAt = &paid
			record := h.enqueueSettlement(t, objectType, accountingsync.SyncOperationCreate, settlement)

			h.drain(t)

			assertNothingSent(t, h, record, "payroll")
			assert.Len(t, h.records.all(), 1, "no vendor or bill dependency is queued for a company driver")
			assert.Empty(t, h.mappings.created)
		})
	}
}

func TestBillVoidOfASyncedBillDeletesThatDocument(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	created := h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBill, settlement, "qb-vc-9", map[string]string{
		accountingsync.ExternalRefDocument:     "qb-vc-9",
		accountingsync.ExternalRefDocumentType: "VendorCredit",
	})
	void := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationVoid, settlement)

	h.drain(t)

	calls := h.writer.callsTo("VoidPurchaseDocument")
	require.Len(t, calls, 1)
	ref, ok := calls[0].doc.(*services.AccountingDocumentRef)
	require.True(t, ok)
	assert.Equal(t, "qb-vc-9", ref.ExternalID)
	assert.Equal(t, accountingsync.SyncObjectCarrierBill, ref.Kind)
	assert.Equal(t, void.RequestID, ref.RequestID)
	assert.Equal(t, "VendorCredit", ref.Refs[accountingsync.ExternalRefDocumentType],
		"the adapter learns whether it is deleting a bill or a vendor credit")
	assert.Equal(t, testAccessToken, ref.Auth.AccessToken)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(void.ID).Status)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(created.ID).Status)
	assert.Empty(t, h.writer.callsTo("CreatePurchaseDocument"))
}

func TestBillVoidBeforeTheBillWasSent(t *testing.T) {
	t.Parallel()

	t.Run("the queued bill is withdrawn with the void", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		accts := newPayableAccounts()
		settlement := h.carrierSettlement(accts)
		created := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncOperationCreate, settlement)
		h.records.postpone(created.ID)
		void := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncOperationVoid, settlement)

		result := h.drain(t)

		assert.Equal(t, 1, result.Synced)
		assert.Equal(t, accountingsync.SyncStatusSuperseded, h.records.get(created.ID).Status,
			"nothing reaches the books")
		assertNothingSent(t, h, void, "withdrawn")
	})

	t.Run("a void with no bill record sends nothing", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.sendDriverSettlements(syncEnabledAt)
		accts := newPayableAccounts()
		settlement := h.driverSettlement(accts)
		void := h.enqueueSettlement(t, accountingsync.SyncObjectDriverBill,
			accountingsync.SyncOperationVoid, settlement)

		h.drain(t)

		assertNothingSent(t, h, void, "never reached the books")
	})

	t.Run("a bill still being sent is waited for", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		accts := newPayableAccounts()
		settlement := h.carrierSettlement(accts)
		created := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncOperationCreate, settlement)
		inFlight := h.records.get(created.ID)
		inFlight.Status = accountingsync.SyncStatusInFlight
		lease := time.Now().Add(time.Hour).Unix()
		inFlight.LeaseExpiresAt = &lease
		h.records.put(inFlight)
		void := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncOperationVoid, settlement)

		result := h.drain(t)

		assert.Equal(t, 1, result.Waiting)
		assert.Equal(t, created.ID, h.records.get(void.ID).DependsOnRecordID)
		assert.Empty(t, h.writer.callsTo("VoidPurchaseDocument"))
	})
}

func (h *harness) paidCarrierSettlement(accts payableAccounts) *repositories.PayableSettlement {
	settlement := h.carrierSettlement(accts)
	paid := settledPaidAt
	settlement.PaidAt = &paid
	return settlement
}

func TestBillPaymentPaysTheSyncedBillFromTheCashAccount(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	mapped := h.mapCarrierBill(accts, settlement)
	h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBill, settlement, "qb-bill-1", nil)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)

	result := h.drain(t)

	assert.Equal(t, 1, result.Synced)
	doc := onlyBillPayment(t, h)
	assert.Equal(t, accountingsync.SyncObjectCarrierBillPay, doc.Kind)
	assert.Equal(t, record.RequestID, doc.RequestID)
	assert.Equal(t, "qb-vendor-swift", doc.VendorExternalID)
	assert.Equal(t, "qb-bill-1", doc.BillExternalID)
	assert.Equal(t, "qb-bank", doc.BankAccountExternalID, "the default cash account pays through the deposit role")
	assert.Equal(t, "CHK-5521", doc.DocNumber, "the number is the payment reference")
	assert.Equal(t, "2026-04-18", doc.TxnDate, "the payment is dated the paid date")
	assert.Equal(t, "USD", doc.CurrencyCode)
	decimalEqual(t, "1650", doc.Amount)
	assert.Equal(t, "Trenova payment of carrier settlement CS-1042 · Paid by ACH · Reference CHK-5521",
		doc.PrivateNote)
	synced := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, synced.Status)
	assert.ElementsMatch(t, []string{mapped.vendor.ID.String(), mapped.deposit.ID.String()}, synced.MappingIDs)
	assert.Equal(t, "https://qbo.test/CarrierBillPayment/"+synced.ExternalID, synced.ExternalURL)
	assert.Empty(t, h.writer.callsTo("CreatePurchaseDocument"))
}

func TestBillPaymentFromANonDefaultBankAccountNeedsItsOwnMapping(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	operating := pulid.MustNew("gla_")
	settlement.BankAccountID = operating
	h.mapCarrierBill(accts, settlement)
	h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBill, settlement, "qb-bill-1", nil)
	h.mappings.set(h.conn, glTarget(operating, "GL account 1010 Operating checking"),
		accountingsync.MappingStateUnmatched, "")
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Equal(t, "Map GL account 1010 Operating checking to a QuickBooks Online account", blocked.Resolution)
	assert.Empty(t, h.writer.callsTo("CreateBillPayment"))

	h.confirm(glTarget(operating, "GL account 1010 Operating checking"), "qb-operating")
	_, err := h.svc.Retry(t.Context(), &services.RetryAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: h.conn.IntegrationType,
		IDs:             []pulid.ID{record.ID},
	})
	require.NoError(t, err)
	h.drain(t)

	assert.Equal(t, "qb-operating", onlyBillPayment(t, h).BankAccountExternalID)
}

func TestBillPaymentWaitsForAnUnsentBill(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	bill := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)
	h.records.postpone(bill.ID)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)

	result := h.drain(t)

	assert.Equal(t, 1, result.Waiting)
	waiting := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusQueued, waiting.Status)
	assert.Equal(t, bill.ID, waiting.DependsOnRecordID)
	assert.Contains(t, waiting.ErrorMessage, "Carrier settlement CS-1042")
	assert.Empty(t, h.writer.callsTo("CreateBillPayment"))

	h.records.makeDue(bill.ID)
	h.drain(t)
	h.records.makeDue(record.ID)
	h.drain(t)

	billRecord := h.records.get(bill.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, billRecord.Status)
	assert.Equal(t, billRecord.ExternalID, onlyBillPayment(t, h).BillExternalID)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
}

func TestBillPaymentWithoutABillRecordQueuesTheBill(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) { conn.AutoSync = false })
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)
	_, err := h.svc.Release(t.Context(), &services.ReleaseAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: h.conn.IntegrationType,
	})
	require.NoError(t, err)

	first := h.drain(t)

	assert.Equal(t, 1, first.Waiting)
	bill := h.only(t, accountingsync.SyncObjectCarrierBill, settlement.ID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncSourceDependencyOf, bill.SourceEvent)
	assert.Equal(t, accountingsync.SyncStatusQueued, bill.Status, "a dependency is never held for approval")
	assert.Equal(t, "CS-1042", bill.ObjectNumber)
	require.NotNil(t, bill.DocumentDate)
	assert.Equal(t, settledPosted, *bill.DocumentDate)
	assert.Equal(t, bill.ID, h.records.get(record.ID).DependsOnRecordID)
	assert.Contains(t, h.dispatcher.kicked(), h.conn.ID)

	h.drain(t)
	h.records.makeDue(record.ID)
	h.drain(t)

	assert.Len(t, purchaseDocs(t, h), 1)
	assert.Equal(t, h.records.get(bill.ID).ExternalID, onlyBillPayment(t, h).BillExternalID)
}

func TestBillPaymentForABillBeforeTheStartDateSendsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		moved := settledPosted + 1
		conn.SyncStartDate = &moved
	})

	h.drain(t)

	assertNothingSent(t, h, record, "before the start date")
	assert.Empty(t, h.records.find(accountingsync.SyncObjectCarrierBill, settlement.ID,
		accountingsync.SyncOperationCreate), "no bill is queued for a settlement the books never got")
}

func TestBillPaymentOfABillThatWasNotSentSendsNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		apply func(*accountingsync.AccountingSyncRecord)
	}{
		{
			name: "skipped by a person",
			apply: func(r *accountingsync.AccountingSyncRecord) {
				r.Status = accountingsync.SyncStatusSkipped
				r.SkippedReason = "Entered by hand"
			},
		},
		{
			name: "withdrawn by a void",
			apply: func(r *accountingsync.AccountingSyncRecord) {
				r.Status = accountingsync.SyncStatusSuperseded
			},
		},
		{
			name: "nothing to send",
			apply: func(r *accountingsync.AccountingSyncRecord) {
				r.MarkSynced(&accountingsync.SyncResult{}, settledPosted)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			accts := newPayableAccounts()
			settlement := h.paidCarrierSettlement(accts)
			h.mapCarrierBill(accts, settlement)
			bill := NewRecordFor(h.conn, settlementRequest(h, accountingsync.SyncObjectCarrierBill,
				accountingsync.SyncOperationCreate, settlement), settledPosted)
			tc.apply(bill)
			h.records.put(bill)
			record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay,
				accountingsync.SyncOperationCreate, settlement)

			h.drain(t)

			assertNothingSent(t, h, record, "was not sent, so its payment is not sent either")
		})
	}
}

func TestBillPaymentAmountRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*repositories.PayableSettlement)
		reason string
	}{
		{
			name:   "a negative net was a vendor credit",
			mutate: func(s *repositories.PayableSettlement) { s.NetMinor = -20000 },
			reason: "vendor credit",
		},
		{
			name:   "a zero net has nothing to pay",
			mutate: func(s *repositories.PayableSettlement) { s.NetMinor = 0 },
			reason: "nothing to send",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			accts := newPayableAccounts()
			settlement := h.paidCarrierSettlement(accts)
			tc.mutate(settlement)
			h.mapCarrierBill(accts, settlement)
			h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBill, settlement, "qb-vc-1", nil)
			record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay,
				accountingsync.SyncOperationCreate, settlement)

			h.drain(t)

			assertNothingSent(t, h, record, tc.reason)
		})
	}
}

func TestNegativeNetPaymentIsNotSentEvenWithoutABill(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	settlement.NetMinor = -500
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)

	h.drain(t)

	assertNothingSent(t, h, record, "vendor credit")
	assert.Empty(t, h.records.find(accountingsync.SyncObjectCarrierBill, settlement.ID,
		accountingsync.SyncOperationCreate), "no bill is queued for a payment that is never sent")
}

func TestBillPaymentOfAnUnpaidSettlementBlocks(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBill, settlement, "qb-bill-1", nil)
	record := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorValidation, blocked.ErrorCategory)
	assert.Equal(t, "Carrier settlement CS-1042 is not marked paid", blocked.ErrorMessage)
	assert.Empty(t, h.writer.callsTo("CreateBillPayment"))
}

func TestOwnerOperatorBillPaymentUsesTheDriverVendor(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sendDriverSettlements(syncEnabledAt)
	accts := newPayableAccounts()
	settlement := h.driverSettlement(accts)
	paid := settledPaidAt
	settlement.PaidAt = &paid
	h.mapPayableRoles()
	h.confirm(driverVendorTarget(settlement.PartyID), "qb-vendor-dana")
	h.syncedSettlementRecord(accountingsync.SyncObjectDriverBill, settlement, "qb-bill-dana", nil)
	h.enqueueSettlement(t, accountingsync.SyncObjectDriverBillPay, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	doc := onlyBillPayment(t, h)
	assert.Equal(t, accountingsync.SyncObjectDriverBillPay, doc.Kind)
	assert.Equal(t, "qb-vendor-dana", doc.VendorExternalID)
	assert.Equal(t, "qb-bill-dana", doc.BillExternalID)
	assert.Equal(t, "qb-bank", doc.BankAccountExternalID)
	decimalEqual(t, "2100", doc.Amount)
	assert.Equal(t, "Trenova payment of owner-operator settlement DS-2201 · Paid by Direct deposit · Reference DD-778",
		doc.PrivateNote)
}

func TestVendorCreditFollowsTheLinesSentNotTheNetField(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	settlement.NetMinor = 5000
	settlement.Lines = []repositories.PayableJournalLine{
		glLine(accts.pt, "5000", "Purchased transportation", 10000, 0),
		glLine(accts.escrow, "2150", "Escrow liability", 0, 15000),
		glLine(accts.ap, "2000", "Accounts payable", 5000, 0),
	}
	h.mapCarrierBill(accts, settlement)
	h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)

	h.drain(t)

	doc := onlyPurchaseDoc(t, h)
	assert.True(t, doc.VendorCredit, "lines that total below zero are a vendor credit whatever the net says")
	assertPurchaseLines(t, []purchaseLineWant{
		{account: "qb-pt", amount: "-100", description: "Purchased transportation"},
		{account: "qb-escrow", amount: "150", description: "Escrow liability"},
	}, doc.Lines)
}

func TestBillPaymentReferenceIsCutToTheProvidersLimit(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.writer.limits.MaxDocNumberLength = 8
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	settlement.PaymentReference = "  WIRE-2026-04-18-7781  "
	h.mapCarrierBill(accts, settlement)
	h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBill, settlement, "qb-bill-1", nil)
	h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)

	h.drain(t)

	doc := onlyBillPayment(t, h)
	assert.Equal(t, "WIRE-202", doc.DocNumber)
	assert.Contains(t, doc.PrivateNote, "Reference WIRE-2026-04-18-7781",
		"the full reference stays in the note")
}
