//go:build netsuitesandbox

package nsconnector

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sandboxEnvPrefix   = "TRENOVA_NETSUITE_SANDBOX_"
	sandboxLookback    = 30 * 24 * time.Hour
	sandboxStateSample = 20
	sandboxAmount      = 100
	sandboxCleanupWait = time.Minute
)

type sandbox struct {
	conn    *Connector
	account string
	ref     netsuite.CompanyRef
	grant   *accountingconnectorports.AccountingTokenGrant
	facts   *accountingconnection.CompanyFacts
	auth    accountingconnectorports.AccountingDocumentAuth
}

func sandboxEnv(name string) string {
	return strings.TrimSpace(os.Getenv(sandboxEnvPrefix + name))
}

func requireSandboxEnv(t *testing.T, names ...string) {
	t.Helper()
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if sandboxEnv(name) == "" {
			missing = append(missing, sandboxEnvPrefix+name)
		}
	}
	if len(missing) > 0 {
		t.Skipf("set %s to run this against a NetSuite sandbox", strings.Join(missing, ", "))
	}
}

func openSandbox(t *testing.T) *sandbox {
	t.Helper()
	requireSandboxEnv(t, "ACCOUNT_ID", "CLIENT_ID", "CERTIFICATE_ID", "PRIVATE_KEY_FILE")

	key, err := os.ReadFile(sandboxEnv("PRIVATE_KEY_FILE"))
	require.NoError(t, err)

	provider := newProvider(t, config.NetSuiteConfig{})
	conn, err := provider.bind(&accountingconnectorports.AccountingApp{
		Source:      accountingconnection.AppSourceTenant,
		Environment: accountingprovider.AppEnvironmentProduction,
		ClientID:    sandboxEnv("CLIENT_ID"),
	})
	require.NoError(t, err)

	account, err := conn.NormalizeAccountID(sandboxEnv("ACCOUNT_ID"))
	require.NoError(t, err)
	grant, err := conn.TokenFromCertificate(
		t.Context(),
		&accountingconnectorports.AccountingCertificateGrant{
			RealmID:       account,
			CertificateID: sandboxEnv("CERTIFICATE_ID"),
			PrivateKeyPEM: key,
		},
	)
	require.NoError(t, err, "the client-credentials request was refused")

	ref := netsuite.CompanyRef{AccountID: account, SubsidiaryID: sandboxEnv("SUBSIDIARY_ID")}
	facts, err := conn.CompanyFacts(t.Context(), ref.String(), grant.AccessToken)
	require.NoError(t, err, "companyinformation was not readable through the account host")

	return &sandbox{
		conn:    conn,
		account: account,
		ref:     ref,
		grant:   grant,
		facts:   facts,
		auth: accountingconnectorports.AccountingDocumentAuth{
			RealmID:     ref.String(),
			AccessToken: grant.AccessToken,
			CompanyCode: facts.HomeCurrency,
			CompanyName: facts.CompanyName,
		},
	}
}

func TestSandboxCertificateGrantsAnAccessToken(t *testing.T) {
	s := openSandbox(t)

	assert.NotEmpty(t, s.grant.AccessToken)
	assert.Positive(t, s.grant.AccessTokenTTL)
	assert.Empty(t, s.grant.RefreshToken, "a certificate connection never holds a refresh token")
}

func TestSandboxListsCompaniesAndReadsFacts(t *testing.T) {
	s := openSandbox(t)

	companies, err := s.conn.Companies(t.Context(), s.grant, s.account)
	require.NoError(t, err)
	require.NotEmpty(t, companies)
	ids := make([]string, 0, len(companies))
	for idx := range companies {
		ids = append(ids, companies[idx].ID)
	}
	assert.Contains(t, ids, s.ref.String())

	assert.NotEmpty(t, s.facts.CompanyName)
	assert.NotEmpty(t, s.facts.HomeCurrency)
	t.Logf("company %q in %s, fiscal year starts in %s, books closed through %v",
		s.facts.CompanyName, s.facts.HomeCurrency, s.facts.FiscalYearStartMonth,
		s.facts.BooksClosedThrough)
}

func TestSandboxReadsEveryReferenceKind(t *testing.T) {
	s := openSandbox(t)

	pageSize := min(50, s.conn.MaxReferencePageSize())
	for _, kind := range accountingprovider.MustProfile(integration.TypeNetSuite).ReferenceKinds {
		t.Run(string(kind), func(t *testing.T) {
			page, err := s.conn.ListReference(
				t.Context(),
				&accountingconnectorports.AccountingReferencePageRequest{
					RealmID:     s.auth.RealmID,
					AccessToken: s.auth.AccessToken,
					Kind:        kind,
					PageSize:    pageSize,
				},
			)
			require.NoError(t, err)
			t.Logf("%s: %d on the first page", kind, len(page.Objects))
		})
	}
}

func TestSandboxReadsChangesAndDocumentStates(t *testing.T) {
	s := openSandbox(t)

	now := time.Now().UTC()
	page, err := s.conn.ReadChanges(
		t.Context(),
		&accountingconnectorports.ReadAccountingChangesRequest{
			Auth:           s.auth,
			Cursor:         s.conn.ChangeCursorAt(now.Add(-sandboxLookback)),
			Now:            now,
			Payments:       true,
			BillPayments:   true,
			Documents:      true,
			ReferenceKinds: accountingprovider.MustProfile(integration.TypeNetSuite).ReferenceKinds,
		},
	)
	require.NoError(t, err, "a SuiteQL table or column name in the change feed is wrong")
	t.Logf("changes: %d payments, %d documents, %d references, more=%v, deletes unread=%v",
		len(page.Payments), len(page.Documents), len(page.References), page.More,
		page.DeletesUnread)
	if page.DeletesUnread {
		t.Log("the role cannot read deletedrecord; the drift reconciler finds deletions instead")
	}

	byKind := make(
		map[accountingsync.SyncObjectType][]accountingconnectorports.AccountingDocumentTarget,
	)
	sampled := 0
	for idx := range page.Documents {
		doc := &page.Documents[idx]
		if doc.Operation == accountingconnectorports.AccountingChangeDelete ||
			len(doc.ObjectTypes) == 0 ||
			sampled == sandboxStateSample {
			continue
		}
		byKind[doc.ObjectTypes[0]] = append(byKind[doc.ObjectTypes[0]],
			accountingconnectorports.AccountingDocumentTarget{ExternalID: doc.ExternalID})
		sampled++
	}
	if sampled == 0 {
		t.Log("no changed documents in the lookback window; document states were not sampled")
		return
	}

	named := 0
	for kind, targets := range byKind {
		states, readErr := s.conn.ReadDocuments(
			t.Context(),
			&accountingconnectorports.ReadAccountingDocumentsRequest{
				Auth:    s.auth,
				Kind:    kind,
				Targets: targets,
			},
		)
		require.NoError(t, readErr)
		for _, state := range states {
			assert.Truef(
				t,
				state.Found,
				"%s %s changed but reads as missing",
				kind,
				state.ExternalID,
			)
			if state.ModifiedBy != "" {
				named++
			}
		}
	}
	t.Logf("system notes named the last editor of %d of %d sampled documents", named, sampled)
}

func TestSandboxInvoicePaymentLifecycle(t *testing.T) {
	requireSandboxEnv(t, "CUSTOMER_ID", "ITEM_ID", "DEPOSIT_ACCOUNT_ID")
	s := openSandbox(t)
	ctx := t.Context()

	started := time.Now().UTC()
	suffix := strconv.FormatInt(started.UnixNano(), 36)
	requestID := "trn-sandbox-" + suffix
	amount := decimal.NewFromInt(sandboxAmount)
	today := started.Format(dateLayout)

	doc := &accountingconnectorports.AccountingSalesDocument{
		Auth:               s.auth,
		RequestID:          requestID,
		Kind:               accountingsync.SyncObjectInvoice,
		CustomerExternalID: sandboxEnv("CUSTOMER_ID"),
		DocNumber:          "TRN-SBX-" + suffix,
		TxnDate:            today,
		DueDate:            started.AddDate(0, 0, 30).Format(dateLayout),
		CurrencyCode:       s.facts.HomeCurrency,
		PrivateNote:        "Trenova sandbox run " + suffix,
		Lines: []accountingconnectorports.AccountingDocumentLine{{
			Description:    "Trenova sandbox line",
			ItemExternalID: sandboxEnv("ITEM_ID"),
			Quantity:       decimal.NewFromInt(1),
			UnitPrice:      amount,
			Amount:         amount,
		}},
	}
	created, err := s.conn.CreateSalesDocument(ctx, doc)
	require.NoError(t, err)
	invoiceID := created.ExternalID
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), sandboxCleanupWait)
		defer cancel()
		_, voidErr := s.conn.VoidSalesDocument(
			cleanupCtx,
			&accountingconnectorports.AccountingDocumentRef{
				Auth:       s.auth,
				RequestID:  requestID + "-cleanup",
				Kind:       accountingsync.SyncObjectInvoice,
				ExternalID: invoiceID,
				Refs:       created.Refs,
			},
		)
		if voidErr != nil {
			t.Logf("cleanup left invoice %s in the sandbox: %v", invoiceID, voidErr)
		}
	})

	again, err := s.conn.CreateSalesDocument(ctx, doc)
	require.NoError(t, err)
	assert.Equal(t, invoiceID, again.ExternalID,
		"PUT eid: did not upsert the invoice; creates need the POST fallback (§9.10 decision 6)")

	found, ok, err := s.conn.FindDocument(
		ctx,
		&accountingconnectorports.AccountingFindDocumentRequest{
			Auth:                   s.auth,
			Kind:                   accountingsync.SyncObjectInvoice,
			DocNumber:              doc.DocNumber,
			CounterpartyExternalID: doc.CustomerExternalID,
			TxnDate:                today,
			Total:                  amount,
			RequestID:              requestID,
		},
	)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, invoiceID, found.ExternalID)

	before := readSandboxInvoice(t, s, invoiceID)
	require.True(t, before.Found)
	assert.True(t, before.Total.Equal(amount))
	require.NotNil(t, before.Balance)
	assert.True(t, before.Balance.Equal(amount))
	t.Logf("open in NetSuite to confirm the deep link: %s", s.conn.DocumentURL(s.auth,
		accountingconnectorports.AccountingDocumentLink{
			Kind:       accountingsync.SyncObjectInvoice,
			ExternalID: invoiceID,
			DocNumber:  created.DocNumber,
		}))

	time.Sleep(2 * time.Second)
	paid, err := s.conn.SavePayment(ctx, &accountingconnectorports.AccountingPaymentDocument{
		Auth:                     s.auth,
		RequestID:                requestID + "-payment",
		CustomerExternalID:       doc.CustomerExternalID,
		TxnDate:                  today,
		CurrencyCode:             s.facts.HomeCurrency,
		DepositAccountExternalID: sandboxEnv("DEPOSIT_ACCOUNT_ID"),
		ReferenceNumber:          "SBX-" + suffix,
		PrivateNote:              "Trenova sandbox payment " + suffix,
		TotalAmount:              amount,
		Applications: []accountingconnectorports.AccountingPaymentApplication{{
			InvoiceExternalID: invoiceID,
			InvoiceNumber:     created.DocNumber,
			AppliedAmount:     amount,
		}},
	})
	require.NoError(t, err)
	paymentID := paid.ExternalID
	paymentRef := &accountingconnectorports.AccountingDocumentRef{
		Auth:       s.auth,
		RequestID:  requestID + "-payment-void",
		Kind:       accountingsync.SyncObjectCustomerPayment,
		ExternalID: paymentID,
		Refs:       paid.Refs,
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), sandboxCleanupWait)
		defer cancel()
		if _, voidErr := s.conn.VoidPayment(cleanupCtx, paymentRef); voidErr != nil {
			t.Logf("cleanup left payment %s in the sandbox: %v", paymentID, voidErr)
		}
	})

	after := readSandboxInvoice(t, s, invoiceID)
	require.NotNil(t, after.Balance)
	assert.True(t, after.Balance.IsZero(), "the payment did not apply to the invoice")
	t.Logf("applying the payment moved the invoice's lastmodifieddate: %v",
		after.ModifiedAt > before.ModifiedAt)

	changes, err := s.conn.ReadChanges(ctx, &accountingconnectorports.ReadAccountingChangesRequest{
		Auth:     s.auth,
		Cursor:   s.conn.ChangeCursorAt(started.Add(-time.Minute)),
		Now:      time.Now().UTC(),
		Payments: true,
	})
	require.NoError(t, err)
	var readBack *accountingconnectorports.AccountingInboundPayment
	for idx := range changes.Payments {
		if changes.Payments[idx].ExternalID == paymentID {
			readBack = &changes.Payments[idx]
		}
	}
	require.NotNil(t, readBack, "the payment did not come back through the change feed")
	assert.Equal(t, accountinginbound.InboundCustomerPayment, readBack.Kind)
	assert.True(t, readBack.Amount.Equal(amount))
	require.Len(t, readBack.Lines, 1)
	assert.Equal(t, invoiceID, readBack.Lines[0].DocumentExternalID)
	t.Logf("system notes name the payment's editor as %q", readBack.ModifiedBy)

	_, err = s.conn.VoidPayment(ctx, paymentRef)
	require.NoError(t, err)
	_, err = s.conn.VoidSalesDocument(ctx, &accountingconnectorports.AccountingDocumentRef{
		Auth:       s.auth,
		RequestID:  requestID + "-void",
		Kind:       accountingsync.SyncObjectInvoice,
		ExternalID: invoiceID,
		Refs:       created.Refs,
	})
	require.NoError(t, err)
	assert.False(t, readSandboxInvoice(t, s, invoiceID).Found,
		"an invoice in an open period is deleted on void")
}

func readSandboxInvoice(
	t *testing.T,
	s *sandbox,
	id string,
) *accountingconnectorports.AccountingDocumentState {
	t.Helper()
	states, err := s.conn.ReadDocuments(
		t.Context(),
		&accountingconnectorports.ReadAccountingDocumentsRequest{
			Auth:    s.auth,
			Kind:    accountingsync.SyncObjectInvoice,
			Targets: []accountingconnectorports.AccountingDocumentTarget{{ExternalID: id}},
		},
	)
	require.NoError(t, err)
	require.Len(t, states, 1)
	return states[0]
}
