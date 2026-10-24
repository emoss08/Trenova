//go:build integration

package factoringrepository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/pkg/dbscope"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/factoringcompanyrepository"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type factoringEnv struct {
	ctx         context.Context
	db          *keel.DB
	tenantInfo  pagination.TenantInfo
	customerID  pulid.ID
	shipmentID  pulid.ID
	stateID     pulid.ID
	docTypeID   pulid.ID
	agreements  factoringports.AgreementRepository
	noas        factoringports.NoaRepository
	states      factoringports.InvoiceStateRepository
	invoices    invoiceports.InvoiceRepository
	factors     factoringcompanyports.FactoringCompanyRepository
	invoiceSeed int
}

func setupFactoringEnv(t *testing.T) *factoringEnv {
	t.Helper()
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))

	var shipmentID, customerID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, customer_id FROM shipments WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1",
		orgID, buID,
	).
		Scan(&shipmentID, &customerID))

	var stateID pulid.ID
	require.NoError(
		t,
		db.Pool().QueryRow(ctx, "SELECT id FROM us_states ORDER BY id LIMIT 1").Scan(&stateID),
	)

	var docTypeID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id FROM document_types WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1",
		orgID, buID,
	).
		Scan(&docTypeID))

	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	p := Params{DB: conn, Logger: logger}
	return &factoringEnv{
		ctx:        ctx,
		db:         db,
		tenantInfo: tenantInfo,
		customerID: customerID,
		shipmentID: shipmentID,
		stateID:    stateID,
		docTypeID:  docTypeID,
		agreements: NewAgreementRepository(p),
		noas:       NewNoaRepository(p),
		states:     NewInvoiceStateRepository(p),
		invoices:   invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger}),
		factors: factoringcompanyrepository.New(
			factoringcompanyrepository.Params{DB: conn, Logger: logger},
		),
	}
}

func (e *factoringEnv) createFactor(t *testing.T, code string) *factoringcompany.FactoringCompany {
	t.Helper()
	created, err := e.factors.Create(e.ctx, &factoringcompany.FactoringCompany{
		OrganizationID: e.tenantInfo.OrgID,
		BusinessUnitID: e.tenantInfo.BuID,
		Status:         domaintypes.StatusActive,
		Code:           code,
		Name:           code + " Capital",
		AddressLine1:   "100 Main St",
		City:           "Dallas",
		StateID:        e.stateID,
		PostalCode:     "75201",
		Email:          "remit@" + code + ".example.com",
	})
	require.NoError(t, err)
	return created
}

func (e *factoringEnv) agreement(factorID pulid.ID, from int64, to *int64) *factoring.Agreement {
	return &factoring.Agreement{
		OrganizationID:       e.tenantInfo.OrgID,
		BusinessUnitID:       e.tenantInfo.BuID,
		FactoringCompanyID:   factorID,
		Status:               factoring.AgreementStatusActive,
		ActivatedAt:          new(from),
		EffectiveFrom:        from,
		EffectiveTo:          to,
		AdvanceRateBP:        9000,
		ReserveRateBP:        1000,
		ChargebackWindowDays: 90,
		Recourse:             factoring.RecourseFull,
		AccountingTreatment:  factoring.AccountingTreatmentSale,
		FundingDays:          1,
		SubmissionTime:       600,
		FeeTiers: []*factoring.FeeTier{
			{DaysFrom: 0, DaysTo: new(30), RateBP: 300},
			{DaysFrom: 30, RateBP: 500},
		},
		DocumentTypes: []*documenttype.DocumentType{{ID: e.docTypeID}},
	}
}

func (e *factoringEnv) postInvoice(
	t *testing.T,
	amountMinor int64,
	factorID *pulid.ID,
) *invoice.Invoice {
	t.Helper()
	e.invoiceSeed++
	number := "INV-FAC-" + time.Now().Format("150405") + "-" + string(rune('A'+e.invoiceSeed))
	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   e.tenantInfo.OrgID,
		BusinessUnitID:   e.tenantInfo.BuID,
		ShipmentID:       e.shipmentID,
		BillToCustomerID: e.customerID,
		Number:           number,
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(e.ctx, e.db, queue))

	amount := decimal.New(amountMinor, -2)
	postedAt := time.Now().Unix()
	dueDate := postedAt + 30*86400
	inv, err := e.invoices.Create(e.ctx, &invoice.Invoice{
		OrganizationID:     e.tenantInfo.OrgID,
		BusinessUnitID:     e.tenantInfo.BuID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         e.shipmentID,
		CustomerID:         e.customerID,
		FactoringCompanyID: factorID,
		Number:             number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusPosted,
		PostedAt:           &postedAt,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        postedAt,
		DueDate:            &dueDate,
		BillToName:         "Test Customer",
		SubtotalAmount:     amount,
		OtherAmount:        decimal.Zero,
		TotalAmount:        amount,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		Lines: []*invoice.InvoiceLine{{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Freight",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   amount,
			Amount:      amount,
		}},
	})
	require.NoError(t, err)
	return inv
}

func TestAgreementLifecycle(t *testing.T) {
	env := setupFactoringEnv(t)
	factor := env.createFactor(t, "AGR")
	now := time.Now().Unix()

	created, err := env.agreements.Create(env.ctx, env.agreement(factor.ID, now-86400, nil))
	require.NoError(t, err)
	require.Len(t, created.FeeTiers, 2)
	assert.Equal(t, 0, created.FeeTiers[0].DaysFrom)
	assert.Equal(t, []pulid.ID{env.docTypeID}, created.DocumentTypeIDs())

	_, err = env.agreements.Create(env.ctx, env.agreement(factor.ID, now, nil))
	require.Error(t, err)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "effectiveFrom", validation.Field)

	draft := env.agreement(factor.ID, now, nil)
	draft.Status = factoring.AgreementStatusDraft
	draft.ActivatedAt = nil
	_, err = env.agreements.Create(env.ctx, draft)
	require.NoError(t, err, "a draft may overlap the active agreement")

	inForce, err := env.agreements.GetInForce(env.ctx, factoringports.GetAgreementInForceRequest{
		TenantInfo:         env.tenantInfo,
		FactoringCompanyID: factor.ID,
		At:                 now,
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, inForce.ID)

	created.FeeTiers = []*factoring.FeeTier{{DaysFrom: 0, RateBP: 250}}
	created.DocumentTypes = nil
	created.EffectiveTo = new(now + 3600)
	updated, err := env.agreements.Update(env.ctx, created)
	require.NoError(t, err)
	require.Len(t, updated.FeeTiers, 1)
	assert.Equal(t, 250, updated.FeeTiers[0].RateBP)
	assert.Empty(t, updated.DocumentTypes)

	byFactor, err := env.agreements.ListInForce(
		env.ctx,
		&factoringports.ListAgreementsInForceRequest{
			TenantInfo:          env.tenantInfo,
			FactoringCompanyIDs: []pulid.ID{factor.ID},
			At:                  now + 7200,
		},
	)
	require.NoError(t, err)
	assert.Empty(t, byFactor, "the agreement ended before that time")

	all, err := env.agreements.ListByFactoringCompany(env.ctx, factoringports.ListAgreementsRequest{
		TenantInfo:         env.tenantInfo,
		FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestNoaOpenUniquenessAndExpiry(t *testing.T) {
	env := setupFactoringEnv(t)
	factor := env.createFactor(t, "NOA")
	now := time.Now().Unix()

	noa, err := env.noas.Create(env.ctx, &factoring.Noa{
		OrganizationID:     env.tenantInfo.OrgID,
		BusinessUnitID:     env.tenantInfo.BuID,
		CustomerID:         env.customerID,
		FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, factoring.NoaStatusDraft, noa.Status)

	_, err = env.noas.Create(env.ctx, &factoring.Noa{
		OrganizationID:     env.tenantInfo.OrgID,
		BusinessUnitID:     env.tenantInfo.BuID,
		CustomerID:         env.customerID,
		FactoringCompanyID: factor.ID,
	})
	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)

	expires := now + 10*86400
	noa.Status = factoring.NoaStatusAcknowledged
	noa.AcknowledgedAt = new(now)
	noa.SignedDocumentID = pulid.MustNew("doc_")
	noa.ExpiresAt = &expires
	noa, err = env.noas.Update(env.ctx, noa)
	require.NoError(t, err)

	pair := factoringports.NoaPair{CustomerID: env.customerID, FactoringCompanyID: factor.ID}
	open, err := env.noas.ListOpen(env.ctx, &factoringports.ListOpenNoasRequest{
		TenantInfo: env.tenantInfo,
		Pairs:      []factoringports.NoaPair{pair},
	})
	require.NoError(t, err)
	require.Contains(t, open, pair)
	assert.Equal(t, noa.ID, open[pair].ID)

	expiring, err := env.noas.ListExpiring(env.ctx, factoringports.ListExpiringNoasRequest{
		TenantInfo: env.tenantInfo,
		At:         now,
		Before:     now + factoring.NoaExpiryWarningDays*86400,
	})
	require.NoError(t, err)
	require.Len(t, expiring, 1)

	claimed, err := env.noas.ClaimExpiryWarning(
		env.ctx,
		factoringports.ClaimNoaExpiryWarningRequest{
			TenantInfo: env.tenantInfo,
			ID:         noa.ID,
			At:         now,
		},
	)
	require.NoError(t, err)
	assert.True(t, claimed)
	claimed, err = env.noas.ClaimExpiryWarning(env.ctx, factoringports.ClaimNoaExpiryWarningRequest{
		TenantInfo: env.tenantInfo,
		ID:         noa.ID,
		At:         now,
	})
	require.NoError(t, err)
	assert.False(t, claimed, "the warning is sent once")

	page, err := env.noas.ListConnection(env.ctx, &factoringports.ListNoaConnectionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: env.tenantInfo,
			Pagination: pagination.Info{Limit: 10},
		},
		Cursor:             pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		FactoringCompanyID: factor.ID,
		Statuses:           []factoring.NoaStatus{factoring.NoaStatusAcknowledged},
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 1, *page.TotalCount)
}

func TestInvoiceStateProjection(t *testing.T) {
	env := setupFactoringEnv(t)
	factor := env.createFactor(t, "STATE")
	now := time.Now().Unix()

	factored := env.postInvoice(t, 50000, &factor.ID)
	unfactored := env.postInvoice(t, 20000, nil)

	candidates, err := env.states.ListCandidateInvoiceIDs(
		env.ctx,
		factoringports.ListCandidateInvoicesRequest{TenantInfo: env.tenantInfo},
	)
	require.NoError(t, err)
	assert.Contains(t, candidates, factored.ID)
	assert.NotContains(t, candidates, unfactored.ID)

	ready := &factoring.Readiness{Verdict: factoring.VerdictReady}
	state := &factoring.InvoiceState{
		InvoiceID:          factored.ID,
		OrganizationID:     env.tenantInfo.OrgID,
		BusinessUnitID:     env.tenantInfo.BuID,
		FactoringCompanyID: &factor.ID,
		CustomerID:         env.customerID,
		OpenBalanceMinor:   factored.TotalAmountMinor,
	}
	factoring.NextState(nil, state, ready, now-2*factoring.ReadyUnsubmittedSeconds)
	require.NoError(t, env.states.Upsert(env.ctx, state))

	claimed, err := env.states.ClaimReadyAnnouncement(
		env.ctx,
		&factoringports.ClaimReadyAnnouncementRequest{
			TenantInfo: env.tenantInfo,
			InvoiceID:  factored.ID,
			ReadySince: *state.ReadySince,
			At:         now,
		},
	)
	require.NoError(t, err)
	assert.True(t, claimed)

	stale := &factoring.InvoiceState{
		InvoiceID:          factored.ID,
		OrganizationID:     env.tenantInfo.OrgID,
		BusinessUnitID:     env.tenantInfo.BuID,
		FactoringCompanyID: &factor.ID,
		CustomerID:         env.customerID,
		OpenBalanceMinor:   factored.TotalAmountMinor,
	}
	factoring.NextState(nil, stale, ready, now)
	require.NoError(t, env.states.Upsert(env.ctx, stale))

	saved, err := env.states.Get(env.ctx, factoringports.GetInvoiceStateRequest{
		TenantInfo: env.tenantInfo,
		InvoiceID:  factored.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, *state.ReadySince, *saved.ReadySince, "a stale writer keeps the ready period")
	require.NotNil(t, saved.ReadyAnnouncedAt, "and its announcement")

	summary, err := env.states.GetSummary(env.ctx, factoringports.GetSummaryRequest{
		TenantInfo:  env.tenantInfo,
		ReadyBefore: now - factoring.ReadyUnsubmittedSeconds,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Ready.Count)
	assert.Equal(t, factored.TotalAmountMinor, summary.Ready.AmountMinor)
	assert.Equal(t, 1, summary.ReadyUnsubmitted.Count)

	late, err := env.states.ListReadyUnsubmitted(
		env.ctx,
		factoringports.ListReadyUnsubmittedRequest{
			TenantInfo:  env.tenantInfo,
			ReadyBefore: now - factoring.ReadyUnsubmittedSeconds,
		},
	)
	require.NoError(t, err)
	require.Len(t, late, 1)

	notReady := &factoring.InvoiceState{
		InvoiceID:          factored.ID,
		OrganizationID:     env.tenantInfo.OrgID,
		BusinessUnitID:     env.tenantInfo.BuID,
		FactoringCompanyID: &factor.ID,
		CustomerID:         env.customerID,
		OpenBalanceMinor:   factored.TotalAmountMinor,
	}
	factoring.NextState(saved, notReady, &factoring.Readiness{
		Verdict: factoring.VerdictNotReady,
		Checks:  []factoring.Check{{Code: factoring.CheckNoa, Fix: factoring.FixSendNoa}},
	}, now)
	require.NoError(t, env.states.Upsert(env.ctx, notReady))

	page, err := env.states.ListConnection(
		env.ctx,
		&factoringports.ListInvoiceStateConnectionRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: env.tenantInfo,
				Pagination: pagination.Info{Limit: 10},
			},
			Cursor:   pagination.CursorInfo{Limit: 10},
			Verdicts: []factoring.Verdict{factoring.VerdictNotReady},
		},
	)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Nil(t, page.Items[0].ReadySince)
	require.Len(t, page.Items[0].Checks, 1)
	assert.Equal(t, factoring.FixSendNoa, page.Items[0].Checks[0].Fix)

	search := func(term string) *pagination.CursorListResult[*factoring.InvoiceState] {
		t.Helper()
		found, searchErr := env.states.ListConnection(
			env.ctx,
			&factoringports.ListInvoiceStateConnectionRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: env.tenantInfo,
					Pagination: pagination.Info{Limit: 10},
					Query:      term,
				},
				Cursor: pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
			},
		)
		require.NoError(t, searchErr)
		return found
	}
	byNumber := search(strings.ToLower(factored.Number))
	require.Len(t, byNumber.Items, 1)
	assert.Equal(t, factored.ID, byNumber.Items[0].InvoiceID)
	require.NotNil(t, byNumber.TotalCount)
	assert.Equal(t, 1, *byNumber.TotalCount)
	assert.Len(t, search(factor.Name).Items, 1)
	assert.Empty(t, search("no-such-invoice_%").Items)

	byID, err := env.states.ListByInvoiceIDs(
		env.ctx,
		&factoringports.ListInvoiceStatesByInvoiceIDsRequest{
			TenantInfo: env.tenantInfo,
			InvoiceIDs: []pulid.ID{factored.ID, unfactored.ID},
		},
	)
	require.NoError(t, err)
	assert.Len(t, byID, 1)

	customerInvoices, err := env.states.ListCustomerInvoiceIDs(
		env.ctx,
		&factoringports.ListCustomerInvoiceIDsRequest{
			TenantInfo:  env.tenantInfo,
			CustomerIDs: []pulid.ID{env.customerID},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{factored.ID}, customerInvoices)
}

func TestSetInvoiceOverrideKeepsTheReasonOnlyWhileExcluded(t *testing.T) {
	env := setupFactoringEnv(t)
	factor := env.createFactor(t, "OVRD")
	inv := env.postInvoice(t, 30000, &factor.ID)

	read := func() *invoice.Invoice {
		t.Helper()
		got, err := invoicedb.Invoices.Get(env.ctx, env.db, inv.ID)
		require.NoError(t, err)
		return got
	}

	require.NoError(
		t,
		env.states.SetInvoiceOverride(env.ctx, &factoringports.SetInvoiceOverrideRequest{
			TenantInfo: env.tenantInfo,
			InvoiceID:  inv.ID,
			Override:   invoice.FactoringOverrideExclude,
			Reason:     "Customer pays us directly",
		}),
	)
	excluded := read()
	assert.Equal(t, invoice.FactoringOverrideExclude, excluded.FactoringOverride)
	assert.Equal(t, "Customer pays us directly", excluded.FactoringOverrideReason)
	assert.Nil(t, excluded.FactoringCompanyID)
	assert.Greater(t, excluded.Version, inv.Version)

	require.NoError(
		t,
		env.states.SetInvoiceOverride(env.ctx, &factoringports.SetInvoiceOverrideRequest{
			TenantInfo:         env.tenantInfo,
			InvoiceID:          inv.ID,
			Override:           invoice.FactoringOverrideDefault,
			FactoringCompanyID: &factor.ID,
		}),
	)
	restored := read()
	assert.Equal(t, invoice.FactoringOverrideDefault, restored.FactoringOverride)
	assert.Empty(t, restored.FactoringOverrideReason)
	require.NotNil(t, restored.FactoringCompanyID)
	assert.Equal(t, factor.ID, *restored.FactoringCompanyID)

	err := env.states.SetInvoiceOverride(env.ctx, &factoringports.SetInvoiceOverrideRequest{
		TenantInfo: env.tenantInfo,
		InvoiceID:  inv.ID,
		Override:   invoice.FactoringOverrideDefault,
		Reason:     "A reason on an invoice that is not excluded",
	})
	require.Error(t, err, "the database refuses a reason on an invoice that follows its customer")
}
