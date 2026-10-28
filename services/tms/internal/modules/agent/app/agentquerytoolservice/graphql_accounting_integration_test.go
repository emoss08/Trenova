//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/accountingsyncrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/accountingdriftservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const accountingSystem = "QuickBooksOnline"

// accountingTenant is one tenant's accounting sync rows: its connection,
// the documents sent, the differences found and the mappings to finish.
type accountingTenant struct {
	ctx      context.Context
	info     pagination.TenantInfo
	records  []string
	invoices []string
	drift    []string
	mappings []string
}

type accountingRepos struct {
	connections accountingconnectionports.AccountingConnectionRepository
	records     accountingsyncports.AccountingSyncRecordRepository
	findings    accountingdriftports.AccountingDriftFindingRepository
	mappings    accountingmappingports.AccountingMappingRepository
}

func writeAccountingRows(
	t *testing.T,
	repos *accountingRepos,
	tenant *agentreadtest.Tenant,
	label string,
) *accountingTenant {
	t.Helper()

	ctx := readscope.With(dbscope.WithTenant(t.Context(), tenant.Scope()), nil)
	info := pagination.TenantInfo{
		OrgID: tenant.Actor.OrganizationID, BuID: tenant.Actor.BusinessUnitID,
		UserID: tenant.Actor.UserID,
	}
	now := timeutils.NowUnix()
	conn := &accountingsync.AccountingConnection{
		OrganizationID:  info.OrgID,
		BusinessUnitID:  info.BuID,
		IntegrationType: integration.TypeQuickBooksOnline,
		ExternalRealmID: "realm-" + label + pulid.MustNew("x_").String()[2:10],
	}
	conn.Connect(info.UserID, accountingsync.TokenGrant{
		AccessTokenCiphertext:  "cipher-access",
		AccessTokenExpiresAt:   now + 3600,
		RefreshTokenCiphertext: "cipher-refresh",
		RefreshTokenExpiresAt:  now + 8_640_000,
	}, now)
	conn.ApplyCompanyFacts(&accountingsync.CompanyFacts{
		CompanyName: "Acme " + label, HomeCurrency: "USD",
	})
	conn.SetupStep = accountingsync.SetupStepComplete
	conn.SyncStartDate, conn.SyncEnabledAt = &now, &now
	created, err := repos.connections.Create(ctx, conn)
	require.NoError(t, err)

	out := &accountingTenant{ctx: ctx, info: info}
	var queued []*accountingsync.AccountingSyncRecord
	for idx, objectType := range []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice, accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectCustomerPayment,
	} {
		objectID := pulid.MustNew("inv_")
		queued = append(
			queued,
			accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
				TenantInfo:   info,
				ConnectionID: created.ID,
				Key: accountingsync.SyncRecordKey{
					ObjectType: objectType,
					ObjectID:   objectID,
					Operation:  accountingsync.SyncOperationCreate,
				},
				ObjectNumber: label + "-" + string(rune('1'+idx)),
				SourceEvent:  accountingsync.SyncSourceInvoicePosted,
				At:           now + int64(idx),
			}),
		)
		if objectType == accountingsync.SyncObjectInvoice {
			out.invoices = append(out.invoices, objectID.String())
		}
	}
	result, err := repos.records.Enqueue(ctx, queued)
	require.NoError(t, err)
	for _, record := range result.Inserted {
		out.records = append(out.records, record.ID.String())
	}

	trenova, provider := int64(10_000), int64(9_000)
	for idx := range 3 {
		finding, findErr := repos.findings.Create(ctx,
			accountingsync.NewAccountingDriftFinding(&accountingsync.DriftObservation{
				TenantInfo:    info,
				ConnectionID:  created.ID,
				ObjectType:    accountingsync.SyncObjectInvoice,
				ObjectID:      pulid.MustNew("inv_"),
				ObjectNumber:  label + "-INV-" + string(rune('1'+idx)),
				PartyName:     "Acme Foods " + label,
				Kind:          accountingsync.DriftAmountMismatch,
				CurrencyCode:  "USD",
				TrenovaMinor:  &trenova,
				ProviderMinor: &provider,
				At:            now + int64(idx),
			}))
		require.NoError(t, findErr)
		out.drift = append(out.drift, finding.ID.String())
	}

	var mappings []*accountingsync.AccountingMapping
	for _, key := range []string{"ARAccount", "RevenueAccount", "APAccount"} {
		mappings = append(mappings, &accountingsync.AccountingMapping{
			ID:             pulid.MustNew("acctm_"),
			OrganizationID: info.OrgID,
			BusinessUnitID: info.BuID,
			ConnectionID:   created.ID,
			TargetType:     accountingsync.MappingTargetType("AccountRole"),
			TrenovaKey:     key,
			TargetLabel:    key + " " + label,
			SearchLabel:    key,
			ProviderKind:   accountingsync.ReferenceKindAccount,
			State:          accountingsync.MappingStateUnmatched,
		})
	}
	_, err = repos.mappings.CreateMissing(ctx, mappings)
	require.NoError(t, err)
	for _, mapping := range mappings {
		out.mappings = append(out.mappings, mapping.ID.String())
	}

	return out
}

// Every accounting list reads, through the screen's resolvers under
// row-level security, the tenant's own rows and none of another's, pages
// through them, narrows by a typed filter, and withholds every field above a
// ceiling of Public.
func TestConvertedLists_AccountingReadsUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	repos := &accountingRepos{
		connections: accountingsyncrepository.NewConnectionRepository(
			accountingsyncrepository.ConnectionParams{DB: conn, Logger: logger}),
		records: accountingsyncrepository.NewSyncRecordRepository(
			accountingsyncrepository.SyncRecordParams{DB: conn, Logger: logger}),
		findings: accountingsyncrepository.NewDriftFindingRepository(
			accountingsyncrepository.DriftFindingParams{DB: conn, Logger: logger}),
		mappings: accountingsyncrepository.NewMappingRepository(
			accountingsyncrepository.MappingParams{DB: conn, Logger: logger}),
	}
	references := accountingsyncrepository.NewReferenceRepository(
		accountingsyncrepository.ReferenceParams{DB: conn, Logger: logger})
	resolvers := agentread.ResolverParams{
		AccountingDriftService: accountingdriftservice.New(accountingdriftservice.Params{
			Logger: logger, Connections: repos.connections, Findings: repos.findings,
			Records: repos.records,
		}),
		AccountingSyncService: accountingsyncservice.New(accountingsyncservice.Params{
			Logger: logger, Connections: repos.connections, Records: repos.records,
		}),
		AccountingMappingService: accountingmappingservice.New(accountingmappingservice.Params{
			Logger: logger, Connections: repos.connections, Mappings: repos.mappings,
			References: references,
		}),
	}
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
	ours := writeAccountingRows(t, repos, mine, "A")
	other := writeAccountingRows(t, repos, theirs, "B")
	actor := mine.Actor
	system := map[string]any{"system": accountingSystem}

	cases := []struct {
		tool        string
		ours, other []string
		narrow      map[string]any
		narrowed    []string
	}{
		{
			tool: "accounting_list_drift_findings", ours: ours.drift, other: other.drift,
			narrow:   map[string]any{"system": accountingSystem, "query": "A-INV-1"},
			narrowed: ours.drift[:1],
		},
		{
			tool: "accounting_list_sync_records", ours: ours.records, other: other.records,
			narrow: map[string]any{
				"system": accountingSystem, "documentType": []any{"CustomerPayment"},
			},
			narrowed: ours.records[2:],
		},
		{
			tool: "accounting_list_mapping_gaps", ours: ours.mappings, other: other.mappings,
			narrow:   map[string]any{"system": accountingSystem, "query": "RevenueAccount"},
			narrowed: ours.mappings[1:2],
		},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			converted, err := agentquerytoolservice.ConvertedListTool(c.tool, reader, engine)
			require.NoError(t, err)

			ids, pages := walk(t, ours.ctx, converted, actor, system)
			assert.Greater(t, pages, 1, "the walk must cross a page")
			assert.ElementsMatch(t, c.ours, ids)
			for _, id := range c.other {
				assert.NotContains(t, ids, id)
			}

			narrowed, _ := walk(t, ours.ctx, converted, actor, c.narrow)
			assert.ElementsMatch(t, c.narrowed, narrowed)

			hidden, err := agentquerytoolservice.ConvertedListTool(c.tool, public, engine)
			require.NoError(t, err)
			page := callTool(t, ours.ctx, hidden, actor, system)
			assert.NotEmpty(t, page["withheldByAccess"])
			for _, item := range page["items"].([]any) {
				row := item.(map[string]any)
				assert.NotEmpty(t, row["id"])
				assert.NotContains(t, row, "status")
			}
		})
	}

	t.Run("documents named by id read their states, and only the tenant's", func(t *testing.T) {
		converted, err := agentquerytoolservice.ConvertedListTool(
			"accounting_list_sync_records", reader, engine)
		require.NoError(t, err)
		page := callTool(t, ours.ctx, converted, actor, map[string]any{
			"documentIds": []any{ours.invoices[0], other.invoices[0]},
		})
		states, _ := page["states"].([]any)
		require.Len(t, states, 1, "another tenant's document reads as never sent")
		state := states[0].(map[string]any)
		assert.Equal(t, ours.invoices[0], state["objectId"])
		record, _ := state["record"].(map[string]any)
		assert.Equal(t, ours.records[0], record["id"])
	})
}
