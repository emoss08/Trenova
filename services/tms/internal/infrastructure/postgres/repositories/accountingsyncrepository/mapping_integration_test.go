//go:build integration

package accountingsyncrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type mappingFixture struct {
	ctx        context.Context
	db         *bun.DB
	conn       *postgres.Connection
	tenant     pagination.TenantInfo
	connection *accountingsync.AccountingConnection
	references repositories.AccountingReferenceObjectRepository
	mappings   repositories.AccountingMappingRepository
	userID     pulid.ID
}

func setupMappingFixture(t *testing.T) *mappingFixture {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	connections := NewConnectionRepository(ConnectionParams{DB: conn, Logger: zap.NewNop()})
	created, err := connections.Create(ctx, newConnection(tenant, data.User.ID, realm, timeutils.NowUnix()))
	require.NoError(t, err)

	return &mappingFixture{
		ctx:        ctx,
		db:         db,
		conn:       conn,
		tenant:     tenant,
		connection: created,
		references: NewReferenceRepository(ReferenceParams{DB: conn, Logger: zap.NewNop()}),
		mappings:   NewMappingRepository(MappingParams{DB: conn, Logger: zap.NewNop()}),
		userID:     data.User.ID,
	}
}

func account(externalID, name, accountType string) *accountingsync.AccountingReferenceObject {
	return &accountingsync.AccountingReferenceObject{
		Kind:        accountingsync.ReferenceKindAccount,
		ExternalID:  externalID,
		Name:        name,
		AccountType: accountType,
		Active:      true,
	}
}

func TestReferenceRepository_UpsertRefreshesAndMarksUnseenRecordsRemoved(t *testing.T) {
	f := setupMappingFixture(t)

	require.NoError(t, f.references.Upsert(f.ctx, &repositories.UpsertAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		SeenAt:       100,
		Objects: []*accountingsync.AccountingReferenceObject{
			account("84", "Accounts Receivable (A/R)", "Accounts Receivable"),
			account("79", "Freight Income", "Income"),
		},
	}))

	require.NoError(t, f.references.Upsert(f.ctx, &repositories.UpsertAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		SeenAt:       200,
		Objects: []*accountingsync.AccountingReferenceObject{
			account("84", "Accounts Receivable", "Accounts Receivable"),
		},
	}))

	removed, err := f.references.MarkRemovedUnseen(f.ctx, &repositories.MarkAccountingReferenceRemovedRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Kind:         accountingsync.ReferenceKindAccount,
		SeenBefore:   200,
		At:           201,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), removed)

	all, err := f.references.ListByKind(f.ctx, &repositories.ListAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Kind:         accountingsync.ReferenceKindAccount,
	})
	require.NoError(t, err)
	require.Len(t, all, 2)
	byID := map[string]*accountingsync.AccountingReferenceObject{}
	for _, obj := range all {
		byID[obj.ExternalID] = obj
	}
	assert.Equal(t, "Accounts Receivable", byID["84"].Name, "the second pull updated the name")
	assert.Equal(t, "accounts receivable", byID["84"].SearchName)
	assert.Nil(t, byID["84"].RemovedAt)
	require.NotNil(t, byID["79"].RemovedAt)
	assert.Equal(t, int64(201), *byID["79"].RemovedAt)

	usable, err := f.references.ListByKind(f.ctx, &repositories.ListAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Kind:         accountingsync.ReferenceKindAccount,
		UsableOnly:   true,
	})
	require.NoError(t, err)
	require.Len(t, usable, 1)
	assert.Equal(t, "84", usable[0].ExternalID)

	require.NoError(t, f.references.Upsert(f.ctx, &repositories.UpsertAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		SeenAt:       300,
		Objects:      []*accountingsync.AccountingReferenceObject{account("79", "Freight Income", "Income")},
	}))
	back, err := f.references.GetByExternalIDs(f.ctx, &repositories.GetAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Kind:         accountingsync.ReferenceKindAccount,
		ExternalIDs:  []string{"79"},
	})
	require.NoError(t, err)
	require.Len(t, back, 1)
	assert.Nil(t, back[0].RemovedAt, "a record seen again is no longer removed")
}

func TestReferenceRepository_SearchSkipsUnusableItemsAndRanksPrefixMatchesFirst(t *testing.T) {
	f := setupMappingFixture(t)

	items := []*accountingsync.AccountingReferenceObject{
		{Kind: accountingsync.ReferenceKindItem, ExternalID: "1", Name: "Detention", ItemType: "Service", Active: true},
		{Kind: accountingsync.ReferenceKindItem, ExternalID: "2", Name: "Truck detention charge", ItemType: "Service", Active: true},
		{Kind: accountingsync.ReferenceKindItem, ExternalID: "3", Name: "Detention group", ItemType: "Category", Active: true},
		{Kind: accountingsync.ReferenceKindItem, ExternalID: "4", Name: "Detention old", ItemType: "Service", Active: false},
		{Kind: accountingsync.ReferenceKindItem, ExternalID: "5", Name: "Lumper", ItemType: "Service", Active: true},
	}
	require.NoError(t, f.references.Upsert(f.ctx, &repositories.UpsertAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		SeenAt:       100,
		Objects:      items,
	}))

	found, err := f.references.Search(f.ctx, &repositories.SearchAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Kind:         accountingsync.ReferenceKindItem,
		Query:        "Detention",
		UsableOnly:   true,
	})
	require.NoError(t, err)
	require.Len(t, found, 2)
	assert.Equal(t, "1", found[0].ExternalID)
	assert.Equal(t, "2", found[1].ExternalID)

	withInactive, err := f.references.Search(f.ctx, &repositories.SearchAccountingReferenceObjectsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Kind:         accountingsync.ReferenceKindItem,
		Query:        "detention",
	})
	require.NoError(t, err)
	assert.Len(t, withInactive, 4)
}

func roleRow(f *mappingFixture, key string) *accountingsync.AccountingMapping {
	return &accountingsync.AccountingMapping{
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		ConnectionID:   f.connection.ID,
		TargetType:     accountingsync.TargetAccountRole,
		TrenovaKey:     key,
		TargetLabel:    key,
		State:          accountingsync.MappingStateUnmatched,
	}
}

func TestMappingRepository_CreateMissingIsIdempotentPerTarget(t *testing.T) {
	f := setupMappingFixture(t)

	created, err := f.mappings.CreateMissing(f.ctx, []*accountingsync.AccountingMapping{
		roleRow(f, accountingsync.AccountRoleAR),
		roleRow(f, accountingsync.AccountRoleRevenue),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), created)

	created, err = f.mappings.CreateMissing(f.ctx, []*accountingsync.AccountingMapping{
		roleRow(f, accountingsync.AccountRoleAR),
		roleRow(f, accountingsync.AccountRoleDeposit),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), created, "the existing AR row is left alone")

	rows, err := f.mappings.ListByConnection(f.ctx, &repositories.ListAccountingMappingsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
	})
	require.NoError(t, err)
	assert.Len(t, rows, 3)
}

func TestMappingRepository_TheDatabaseEnforcesTheStateInvariant(t *testing.T) {
	f := setupMappingFixture(t)

	bad := roleRow(f, accountingsync.AccountRoleAR)
	bad.State = accountingsync.MappingStateProposed
	_, err := f.mappings.CreateMissing(f.ctx, []*accountingsync.AccountingMapping{bad})
	require.Error(t, err, "a proposed row must name a QuickBooks record")

	both := roleRow(f, accountingsync.AccountRoleAR)
	both.TrenovaObjectID = pulid.MustNew("cus_")
	_, err = f.mappings.CreateMissing(f.ctx, []*accountingsync.AccountingMapping{both})
	require.Error(t, err, "a row is keyed by a record or a value, never both")
}

func TestMappingRepository_ApplyScoringNeverOverwritesAConfirmedOrChangedRow(t *testing.T) {
	f := setupMappingFixture(t)

	_, err := f.mappings.CreateMissing(f.ctx, []*accountingsync.AccountingMapping{
		roleRow(f, accountingsync.AccountRoleAR),
		roleRow(f, accountingsync.AccountRoleRevenue),
		roleRow(f, accountingsync.AccountRoleDeposit),
	})
	require.NoError(t, err)
	rows, err := f.mappings.ListByConnection(f.ctx, &repositories.ListAccountingMappingsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
	})
	require.NoError(t, err)
	byKey := map[string]*accountingsync.AccountingMapping{}
	for _, row := range rows {
		byKey[row.TrenovaKey] = row
	}

	confirmedCopy := *byKey[accountingsync.AccountRoleAR]
	confirmedCopy.Confirm(&accountingsync.Choice{
		ExternalID: "84",
		Source:     accountingsync.MappingSourceManual,
		ActorID:    f.userID,
		At:         timeutils.NowUnix(),
	})
	_, err = f.mappings.Update(f.ctx, &confirmedCopy)
	require.NoError(t, err)

	staleCopy := *byKey[accountingsync.AccountRoleDeposit]
	staleCopy.TargetLabel = "Renamed elsewhere"
	_, err = f.mappings.Update(f.ctx, &staleCopy)
	require.NoError(t, err)

	for _, key := range []string{accountingsync.AccountRoleAR, accountingsync.AccountRoleRevenue, accountingsync.AccountRoleDeposit} {
		byKey[key].ApplyProposal(&accountingsync.Proposal{
			ExternalID: "12",
			Source:     accountingsync.MappingSourceSuggested,
			Confidence: 0.8,
			Reason:     "Similar name",
		})
	}
	applied, err := f.mappings.ApplyScoring(f.ctx, []*accountingsync.AccountingMapping{
		byKey[accountingsync.AccountRoleAR],
		byKey[accountingsync.AccountRoleRevenue],
		byKey[accountingsync.AccountRoleDeposit],
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), applied, "only the untouched revenue row takes the new score")

	ar, err := f.mappings.GetByTarget(f.ctx, &repositories.GetAccountingMappingByTargetRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		TargetType:   accountingsync.TargetAccountRole,
		TrenovaKey:   accountingsync.AccountRoleAR,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.MappingStateConfirmed, ar.State)
	assert.Equal(t, "84", ar.ExternalID)

	revenue, err := f.mappings.GetByTarget(f.ctx, &repositories.GetAccountingMappingByTargetRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		TargetType:   accountingsync.TargetAccountRole,
		TrenovaKey:   accountingsync.AccountRoleRevenue,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.MappingStateProposed, revenue.State)
	assert.Equal(t, "12", revenue.ExternalID)
	require.NotNil(t, revenue.Confidence)
	assert.InDelta(t, 0.8, *revenue.Confidence, 1e-9)
}

func TestMappingRepository_ListConnectionFiltersAndCounts(t *testing.T) {
	f := setupMappingFixture(t)

	customer := &accountingsync.AccountingMapping{
		OrganizationID:  f.tenant.OrgID,
		BusinessUnitID:  f.tenant.BuID,
		ConnectionID:    f.connection.ID,
		TargetType:      accountingsync.TargetCustomer,
		TrenovaObjectID: pulid.MustNew("cus_"),
		TargetLabel:     "Peak Distributing",
		State:           accountingsync.MappingStateUnmatched,
	}
	_, err := f.mappings.CreateMissing(f.ctx, []*accountingsync.AccountingMapping{
		roleRow(f, accountingsync.AccountRoleAR),
		roleRow(f, accountingsync.AccountRoleAP),
		customer,
	})
	require.NoError(t, err)

	page, err := f.mappings.ListConnection(f.ctx, &repositories.ListAccountingMappingsConnectionRequest{
		Filter:       &pagination.QueryOptions{TenantInfo: f.tenant},
		Cursor:       pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		ConnectionID: f.connection.ID,
		RequiredOnly: true,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, accountingsync.AccountRoleAR, page.Items[0].TrenovaKey)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 1, *page.TotalCount)

	searched, err := f.mappings.ListConnection(f.ctx, &repositories.ListAccountingMappingsConnectionRequest{
		Filter:       &pagination.QueryOptions{TenantInfo: f.tenant},
		Cursor:       pagination.CursorInfo{Limit: 10},
		ConnectionID: f.connection.ID,
		Search:       "peak",
	})
	require.NoError(t, err)
	require.Len(t, searched.Items, 1)
	assert.Equal(t, accountingsync.TargetCustomer, searched.Items[0].TargetType)
	assert.Nil(t, searched.TotalCount, "no count unless asked")

	counts, err := f.mappings.CountByState(f.ctx, f.tenant, f.connection.ID)
	require.NoError(t, err)
	total := 0
	for _, count := range counts {
		assert.Equal(t, accountingsync.MappingStateUnmatched, count.State)
		total += count.Count
	}
	assert.Equal(t, 3, total)
}

func TestMappingRepository_OneTenantCannotReadOrWriteAnothersMappings(t *testing.T) {
	f := setupMappingFixture(t)
	second := seedtest.SeedAdditionalTenant(t, f.ctx, f.db, "MP")
	secondTenant := pagination.TenantInfo{OrgID: second.Organization.ID, BuID: second.BusinessUnit.ID}

	_, err := f.mappings.CreateMissing(f.ctx, []*accountingsync.AccountingMapping{roleRow(f, accountingsync.AccountRoleAR)})
	require.NoError(t, err)
	rows, err := f.mappings.ListByConnection(f.ctx, &repositories.ListAccountingMappingsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)

	_, err = f.mappings.GetByID(f.ctx, repositories.GetAccountingMappingRequest{TenantInfo: secondTenant, ID: rows[0].ID})
	assert.True(t, errortypes.IsNotFoundError(err))

	foreign, err := f.mappings.ListByConnection(f.ctx, &repositories.ListAccountingMappingsRequest{
		TenantInfo:   secondTenant,
		ConnectionID: f.connection.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, foreign)

	hijack := *rows[0]
	hijack.OrganizationID = secondTenant.OrgID
	hijack.BusinessUnitID = secondTenant.BuID
	hijack.Confirm(&accountingsync.Choice{ExternalID: "84", Source: accountingsync.MappingSourceManual, At: 1})
	_, err = f.mappings.Update(f.ctx, &hijack)
	require.Error(t, err, "an update scoped to another tenant touches nothing")

	refs, err := f.references.ListByKind(f.ctx, &repositories.ListAccountingReferenceObjectsRequest{
		TenantInfo:   secondTenant,
		ConnectionID: f.connection.ID,
		Kind:         accountingsync.ReferenceKindAccount,
	})
	require.NoError(t, err)
	assert.Empty(t, refs)
}
