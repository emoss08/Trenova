//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/m2msync"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/customerrepository"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/factoringcompanyrepository"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// factoringRepos are the repositories the factoring reads' services read
// and the test writes each tenant's rows through.
type factoringRepos struct {
	customers   customerports.CustomerRepository
	factors     factoringcompanyports.FactoringCompanyRepository
	agreements  factoringports.AgreementRepository
	noas        factoringports.NoaRepository
	states      factoringports.InvoiceStateRepository
	submissions factoringports.SubmissionRepository
	remittances factoringports.RemittanceRepository
	entries     factoringports.ReserveEntryRepository
	invoices    invoiceports.InvoiceRepository
}

func newFactoringRepos(conn *postgres.Connection) *factoringRepos {
	logger := zap.NewNop()
	params := factoringrepository.Params{DB: conn, Logger: logger}

	return &factoringRepos{
		customers: customerrepository.New(customerrepository.Params{
			DB: conn, Logger: logger, M2MSync: m2msync.NewSyncer(m2msync.SyncerParams{Logger: logger}),
		}),
		factors: factoringcompanyrepository.New(
			factoringcompanyrepository.Params{DB: conn, Logger: logger}),
		agreements:  factoringrepository.NewAgreementRepository(params),
		noas:        factoringrepository.NewNoaRepository(params),
		states:      factoringrepository.NewInvoiceStateRepository(params),
		submissions: factoringrepository.NewSubmissionRepository(params),
		remittances: factoringrepository.NewRemittanceRepository(params),
		entries:     factoringrepository.NewReserveEntryRepository(params),
		invoices:    invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger}),
	}
}

// resolvers are the factoring services the reads' resolvers call, over the
// repositories alone: nothing a read reaches sends, books or evaluates.
func (r *factoringRepos) resolvers(conn *postgres.Connection) agentread.ResolverParams {
	logger := zap.NewNop()

	return agentread.ResolverParams{
		FactoringAgreementService: factoringservice.NewAgreementService(
			factoringservice.AgreementParams{
				DB: conn, Repo: r.agreements, Factors: r.factors, Logger: logger,
			}),
		FactoringNoaService: factoringservice.NewNoaService(factoringservice.NoaParams{
			Repo: r.noas, Customers: r.customers, Factors: r.factors, Logger: logger,
		}),
		FactoringReadinessService: factoringservice.NewReadinessService(
			factoringservice.ReadinessParams{
				States: r.states, Agreements: r.agreements, Noas: r.noas,
				Submissions: r.submissions, Invoices: r.invoices, Customers: r.customers,
				Documents: documentrepository.New(documentrepository.Params{
					DB: conn, Logger: logger,
				}),
				Logger: logger,
			}),
		FactoringSubmissionService: factoringservice.NewSubmissionService(
			factoringservice.SubmissionParams{
				Repo: r.submissions, States: r.states, Agreements: r.agreements,
				Invoices: r.invoices, Factors: r.factors, Customers: r.customers,
				Logger: logger,
			}),
		FactoringLedgerService: factoringservice.NewLedgerService(factoringservice.LedgerParams{
			DB: conn, Submissions: r.submissions, Entries: r.entries,
			Agreements: r.agreements, Invoices: r.invoices, Logger: logger,
		}),
		FactoringRemittanceService: factoringservice.NewRemittanceService(
			factoringservice.RemittanceParams{
				DB: conn, Remittances: r.remittances, Submissions: r.submissions,
				Factors: r.factors, Logger: logger,
			}),
	}
}

// factoringTenant is one tenant's factoring rows: a factor and its
// agreement, notices, invoices in the queue, schedules and remittances.
type factoringTenant struct {
	ctx         context.Context
	factor      string
	agreements  []string
	noas        []string
	invoices    []string
	submissions []string
	remittances []string
}

func writeFactoringRows(
	t *testing.T,
	repos *factoringRepos,
	admin *bun.DB,
	tenant *agentreadtest.Tenant,
	label string,
) *factoringTenant {
	t.Helper()

	ctx := readscope.With(dbscope.WithTenant(t.Context(), tenant.Scope()), nil)
	org, bu := tenant.Actor.OrganizationID, tenant.Actor.BusinessUnitID
	now := timeutils.NowUnix()
	out := &factoringTenant{ctx: ctx}

	factor, err := repos.factors.Create(ctx, &factoringcompany.FactoringCompany{
		OrganizationID: org, BusinessUnitID: bu, Status: domaintypes.StatusActive,
		Code: "FAC" + label, Name: "Factor " + label, AddressLine1: "100 Main St",
		City: "Dallas", StateID: tenant.StateID, PostalCode: "75201",
		Email: "remit@" + label + ".example.com",
	})
	require.NoError(t, err)
	out.factor = factor.ID.String()

	agreement, err := repos.agreements.Create(ctx, &factoring.Agreement{
		OrganizationID: org, BusinessUnitID: bu, FactoringCompanyID: factor.ID,
		Status: factoring.AgreementStatusActive, ActivatedAt: &now, EffectiveFrom: now - 60,
		AdvanceRateBP: 9000, ReserveRateBP: 1000, ChargebackWindowDays: 90,
		Recourse: factoring.RecourseFull, AccountingTreatment: factoring.AccountingTreatmentSale,
		FundingDays: 1, SubmissionTime: 600,
		FeeTiers: []*factoring.FeeTier{{DaysFrom: 0, RateBP: 300}},
	})
	require.NoError(t, err)
	out.agreements = append(out.agreements, agreement.ID.String())

	for idx := range 3 {
		suffix := label + string(rune('1'+idx))
		cust, custErr := repos.customers.Create(ctx, &customer.Customer{
			BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.StatusActive,
			Code: "FC" + suffix, Name: "Factored " + suffix, StateID: tenant.StateID,
			AddressLine1: "1 Main", City: "Dallas", PostalCode: "75001",
		})
		require.NoError(t, custErr)
		noa, noaErr := repos.noas.Create(ctx, &factoring.Noa{
			OrganizationID: org, BusinessUnitID: bu, CustomerID: cust.ID,
			FactoringCompanyID: factor.ID, Status: factoring.NoaStatusSent, SentAt: &now,
			SentTo: "ap@" + suffix + ".example.com",
		})
		require.NoError(t, noaErr)
		out.noas = append(out.noas, noa.ID.String())

		inv := postFactoredInvoice(t, ctx, admin, repos, tenant, cust.ID, factor.ID, suffix)
		state := &factoring.InvoiceState{
			InvoiceID: inv.ID, OrganizationID: org, BusinessUnitID: bu,
			FactoringCompanyID: &factor.ID, CustomerID: cust.ID,
			OpenBalanceMinor: inv.TotalAmountMinor,
		}
		factoring.NextState(nil, state, &factoring.Readiness{Verdict: factoring.VerdictReady}, now)
		require.NoError(t, repos.states.Upsert(ctx, state))
		out.invoices = append(out.invoices, inv.ID.String())

		sub, subErr := repos.submissions.Create(ctx, &factoring.Submission{
			OrganizationID: org, BusinessUnitID: bu, Number: "SCH-" + suffix,
			FactoringCompanyID: factor.ID, AgreementID: agreement.ID,
			Status: factoring.SubmissionStatusSent, InvoiceCount: 1, FaceMinor: 100_00,
			ExpectedAdvanceMinor: 90_00, TransportMethod: factoring.TransportMethodEmail,
			TransportTarget: "funding@" + label + ".example.com",
			IdempotencyKey:  "key-" + suffix, SentAt: &now,
		})
		require.NoError(t, subErr)
		out.submissions = append(out.submissions, sub.ID.String())

		remittance, remErr := repos.remittances.Create(ctx, &factoring.Remittance{
			OrganizationID: org, BusinessUnitID: bu, FactoringCompanyID: factor.ID,
			FileName: "funding-" + suffix + ".csv", Status: factoring.RemittanceStatusPending,
			LineCount: 2, TotalCashMinor: 50_00,
		})
		require.NoError(t, remErr)
		cash := int64(25_00)
		remittance.Lines = []*factoring.RemittanceLine{
			{RowNumber: 1, Cells: []string{"INV-1", "25.00"}, InvoiceNumber: "INV-1",
				CashMinor: &cash, MatchStatus: factoring.RemittanceLineUnmatched},
			{RowNumber: 2, Cells: []string{"???", "x"}, Error: "no amount",
				MatchStatus: factoring.RemittanceLineError},
		}
		require.NoError(t, repos.remittances.ReplaceLines(ctx, remittance))
		out.remittances = append(out.remittances, remittance.ID.String())
	}

	return out
}

// postFactoredInvoice writes a posted invoice the factor holds, on an order
// of the customer's, so the queue has a record to read.
func postFactoredInvoice(
	t *testing.T,
	ctx context.Context,
	admin *bun.DB,
	repos *factoringRepos,
	tenant *agentreadtest.Tenant,
	customerID, factorID pulid.ID,
	suffix string,
) *invoice.Invoice {
	t.Helper()

	org, bu := tenant.Actor.OrganizationID, tenant.Actor.BusinessUnitID
	orderID := pulid.MustNew("ord_")
	_, err := admin.ExecContext(t.Context(), `INSERT INTO orders
		(id, business_unit_id, organization_id, customer_id, order_number)
		VALUES (?, ?, ?, ?, ?)`, orderID, bu, org, customerID, "ORD-"+suffix+orderID.String()[4:10])
	require.NoError(t, err)
	number := "INV-" + suffix + orderID.String()[4:10]
	queue := &billingqueue.BillingQueueItem{
		OrganizationID: org, BusinessUnitID: bu, OrderID: orderID,
		BillToCustomerID: customerID, Number: number, Status: billingqueue.StatusPosted,
		BillType: billingqueue.BillTypeInvoice,
	}
	_, err = admin.NewInsert().Model(queue).Exec(t.Context())
	require.NoError(t, err)

	amount := decimal.New(100_00, -2)
	posted := timeutils.NowUnix()
	due := posted + 30*timeutils.SecondsPerDay
	inv, err := repos.invoices.Create(ctx, &invoice.Invoice{
		OrganizationID: org, BusinessUnitID: bu, BillingQueueItemID: queue.ID,
		OrderID: orderID, CustomerID: customerID, FactoringCompanyID: &factorID,
		Number: number, BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted,
		PostedAt: &posted, PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD",
		InvoiceDate: posted, DueDate: &due, BillToName: "Factored " + suffix,
		SubtotalAmount: amount, OtherAmount: decimal.Zero, TotalAmount: amount,
		AppliedAmount: decimal.Zero, SettlementStatus: invoice.SettlementStatusUnpaid,
		DisputeStatus: invoice.DisputeStatusNone,
		Lines: []*invoice.InvoiceLine{{
			LineNumber: 1, Type: invoice.InvoiceLineTypeFreight, Description: "Freight",
			Quantity: decimal.NewFromInt(1), UnitPrice: amount, Amount: amount,
		}},
	})
	require.NoError(t, err)

	return inv
}

// walkByKey reads every page of a list whose rows name their record by key
// rather than by id (an invoice's readiness by its invoiceId).
func walkByKey(
	t *testing.T,
	ctx context.Context,
	tool agentports.AgentQueryTool,
	actor *permissionports.RequestActor,
	args map[string]any,
	key string,
) (ids []string, pages int) {
	t.Helper()

	call := map[string]any{"limit": 2}
	for name, value := range args {
		call[name] = value
	}
	for pages = 1; pages <= 20; pages++ {
		page := callTool(t, ctx, tool, actor, call)
		items, _ := page["items"].([]any)
		for _, item := range items {
			ids = append(ids, item.(map[string]any)[key].(string))
		}
		if page["hasMore"] != true {
			return ids, pages
		}
		call["after"] = page["nextCursor"]
	}
	require.FailNow(t, tool.Name()+" never reached its last page")

	return ids, pages
}

// Every factoring list reads, through the screen's resolvers under
// row-level security, the tenant's own rows and none of another's, pages
// through them, narrows by a typed filter, and withholds the money at a
// ceiling of Internal and every field at Public; every get reads the
// tenant's record and refuses another's.
func TestConvertedLists_FactoringReadsUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	repos := newFactoringRepos(conn)
	resolvers := repos.resolvers(conn)
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(
		t,
		agentreadtest.Options{Engine: engine, Resolvers: resolvers},
	)
	public := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &publicRole{}, Resolvers: resolvers,
	})

	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	ours := writeFactoringRows(t, repos, admin, mine, "A")
	other := writeFactoringRows(t, repos, admin, theirs, "B")
	actor := mine.Actor

	cases := []struct {
		tool, key   string
		ours, other []string
		narrow      map[string]any
		narrowed    []string
		money       string
	}{
		{
			tool: "factoring_list_readiness", key: "invoiceId",
			ours: ours.invoices, other: other.invoices,
			narrow:   map[string]any{"verdict": []any{"NotReady"}},
			narrowed: nil, money: "openBalanceMinor",
		},
		{
			tool: "factoring_list_noas", key: "id", ours: ours.noas, other: other.noas,
			narrow:   map[string]any{"sentTo": "ap@A1"},
			narrowed: ours.noas[:1],
		},
		{
			tool: "factoring_list_submissions", key: "id",
			ours: ours.submissions, other: other.submissions,
			narrow:   map[string]any{"query": "SCH-A2"},
			narrowed: ours.submissions[1:2], money: "faceMinor",
		},
		{
			tool: "factoring_list_remittances", key: "id",
			ours: ours.remittances, other: other.remittances,
			narrow:   map[string]any{"fileName": "funding-A3"},
			narrowed: ours.remittances[2:], money: "totalCashMinor",
		},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			converted, err := agentquerytoolservice.ConvertedListTool(c.tool, reader, engine)
			require.NoError(t, err)

			ids, pages := walkByKey(t, ours.ctx, converted, actor, nil, c.key)
			assert.Greater(t, pages, 1, "the walk must cross a page")
			assert.ElementsMatch(t, c.ours, ids)
			for _, id := range c.other {
				assert.NotContains(t, ids, id)
			}

			narrowed, _ := walkByKey(t, ours.ctx, converted, actor, c.narrow, c.key)
			assert.ElementsMatch(t, c.narrowed, narrowed)

			if c.money != "" {
				data, queryErr := converted.Query(ours.ctx, &agentports.QueryToolParams{
					OrganizationID: actor.OrganizationID, BusinessUnitID: actor.BusinessUnitID,
					Actor: actor, Timezone: "America/Chicago", Params: map[string]any{},
					DataAccessCeiling: permission.SensitivityInternal,
					RunID:             pulid.MustNew("ar_"),
				})
				require.NoError(t, queryErr)
				page := asMap(t, data)
				assert.Contains(t, page["withheldByAccess"], c.money)
				for _, item := range page["items"].([]any) {
					assert.NotContains(t, item.(map[string]any), c.money)
				}
			}

			hidden, err := agentquerytoolservice.ConvertedListTool(c.tool, public, engine)
			require.NoError(t, err)
			page := callTool(t, ours.ctx, hidden, actor, map[string]any{})
			assert.NotEmpty(t, page["withheldByAccess"])
			for _, item := range page["items"].([]any) {
				assert.NotContains(t, item.(map[string]any), "status")
			}
		})
	}

	t.Run("a notice's query matches who it was sent to", func(t *testing.T) {
		converted, err := agentquerytoolservice.ConvertedListTool(
			"factoring_list_noas", reader, engine)
		require.NoError(t, err)
		matched, _ := walkByKey(t, ours.ctx, converted, actor,
			map[string]any{"query": "ap@A1"}, "id")
		assert.ElementsMatch(t, ours.noas[:1], matched,
			"the search names the sent_to column, so the query reads")
	})

	t.Run("factoring_list_agreements", func(t *testing.T) {
		converted, err := agentquerytoolservice.ConvertedListTool(
			"factoring_list_agreements", reader, engine)
		require.NoError(t, err)
		page := callTool(t, ours.ctx, converted, actor, map[string]any{
			"factoringCompanyId": ours.factor,
		})
		ids := make([]string, 0, 1)
		for _, item := range page["items"].([]any) {
			row := item.(map[string]any)
			ids = append(ids, row["id"].(string))
			assert.Equal(t, true, row["inForce"])
		}
		assert.Equal(t, ours.agreements, ids)

		theirFactor := callTool(t, ours.ctx, converted, actor, map[string]any{
			"factoringCompanyId": other.factor,
		})
		assert.Empty(t, theirFactor["items"])
	})

	gets := []struct {
		tool        string
		ours, other string
		call        func(id string) map[string]any
	}{
		{
			tool: "factoring_get_submission", ours: ours.submissions[0],
			other: other.submissions[0],
			call:  func(id string) map[string]any { return map[string]any{"submissionId": id} },
		},
		{
			tool: "factoring_get_remittance", ours: ours.remittances[0],
			other: other.remittances[0],
			call:  func(id string) map[string]any { return map[string]any{"remittanceId": id} },
		},
	}
	for _, g := range gets {
		t.Run(g.tool, func(t *testing.T) {
			get, err := agentquerytoolservice.ConvertedGetTool(g.tool, reader, engine)
			require.NoError(t, err)
			record := callTool(t, ours.ctx, get, actor, g.call(g.ours))
			assert.Equal(t, g.ours, record["id"])

			_, err = get.Query(ours.ctx, &agentports.QueryToolParams{
				OrganizationID: actor.OrganizationID, BusinessUnitID: actor.BusinessUnitID,
				Actor: actor, Timezone: "America/Chicago", Params: g.call(g.other),
				RunID: pulid.MustNew("ar_"),
			})
			require.Error(t, err, "another tenant's record is not found")
		})
	}

	t.Run("a remittance's lines narrow by match status", func(t *testing.T) {
		get, err := agentquerytoolservice.ConvertedGetTool(
			"factoring_get_remittance", reader, engine)
		require.NoError(t, err)
		record := callTool(t, ours.ctx, get, actor, map[string]any{
			"remittanceId": ours.remittances[0], "matchStatuses": []any{"Error"},
		})
		lines, _ := record["lines"].([]any)
		require.Len(t, lines, 1)
		line := lines[0].(map[string]any)
		assert.Equal(t, "Error", line["matchStatus"])
		assert.Contains(t, record["withheldByAccess"], "lines.cells",
			"what the factor printed carries its amounts, so it stays Restricted")
	})

	t.Run("factoring_get_readiness", func(t *testing.T) {
		get, err := agentquerytoolservice.ConvertedGetTool(
			"factoring_get_readiness", reader, engine)
		require.NoError(t, err)
		record := callTool(t, ours.ctx, get, actor, map[string]any{
			"invoiceId": ours.invoices[0],
		})
		assert.Equal(t, ours.invoices[0], record["invoiceId"])
		assert.NotEmpty(t, record["verdict"])

		_, err = get.Query(ours.ctx, &agentports.QueryToolParams{
			OrganizationID: actor.OrganizationID, BusinessUnitID: actor.BusinessUnitID,
			Actor: actor, Timezone: "America/Chicago",
			Params: map[string]any{"invoiceId": other.invoices[0]},
			RunID:  pulid.MustNew("ar_"),
		})
		require.Error(t, err, "another tenant's invoice is not found")
	})

	t.Run("factoring_get_reserve_balance", func(t *testing.T) {
		get, err := agentquerytoolservice.ConvertedGetTool(
			"factoring_get_reserve_balance", reader, engine)
		require.NoError(t, err)
		record := callTool(t, ours.ctx, get, actor, map[string]any{
			"factoringCompanyId": ours.factor,
		})
		assert.Equal(t, ours.factor, record["factoringCompanyId"])
		assert.Contains(t, record["withheldByAccess"], "reserveMinor")
	})
}
