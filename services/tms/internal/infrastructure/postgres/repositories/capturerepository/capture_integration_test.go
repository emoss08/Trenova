//go:build integration

package capturerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCaptureRepositoriesAgainstPostgres runs every capture repository
// against the real schema: the page upsert on (batch, sequence), the version
// checks, JSONB and array columns, the default-profile swap, tenant scoping of
// cover sheets, and the cascade from a batch to its pages.
func TestCaptureRepositoriesAgainstPostgres(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	p := Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}
	orgID, buID, userID := data.Organization.ID, data.BusinessUnit.ID, data.User.ID
	ti := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	devices := NewDeviceRepository(p)
	dev, err := devices.Create(ctx, &capture.CaptureDevice{OrganizationID: orgID, BusinessUnitID: buID, UserID: userID,
		Name: "D", MachineName: "M", AgentVersion: "1.0.0", Architecture: capture.ArchitectureX64,
		RefreshTokenHash: "r" + pulid.MustNew("x_").String(), AccessTokenHash: "a" + pulid.MustNew("x_").String(), AccessTokenExpiresAt: 10})
	require.NoError(t, err)
	got, err := devices.GetByAccessTokenHash(ctx, dev.AccessTokenHash)
	require.NoError(t, err)
	got.Sources = []capture.SourceInfo{{Name: "fi-8170", Protocol: capture.ProtocolTWAIN, Bitness: 32, Resolutions: []int{300}}}
	_, err = devices.Update(ctx, got)
	require.NoError(t, err)
	_, err = devices.Update(ctx, dev)
	assert.True(t, errortypes.IsVersionMismatchError(err), "stale version refused")
	require.NoError(t, devices.Touch(ctx, repositories.TouchCaptureDeviceRequest{ID: dev.ID, TenantInfo: ti, SeenAt: 99, IP: "1.2.3.4"}))
	_, err = devices.GetByAccessTokenHash(ctx, "missing")
	assert.True(t, errortypes.IsNotFoundError(err))
	list, err := devices.List(ctx, &repositories.ListCaptureDevicesRequest{Filter: &pagination.QueryOptions{TenantInfo: ti}, UserID: userID})
	require.NoError(t, err)
	assert.Equal(t, 1, list.Total)
	reread, err := devices.GetByID(ctx, repositories.GetCaptureDeviceByIDRequest{ID: dev.ID, TenantInfo: ti})
	require.NoError(t, err)
	assert.Equal(t, "fi-8170", reread.Sources[0].Name)
	require.NotNil(t, reread.LastSeenAt)

	pairings := NewPairingRepository(p)
	_, err = pairings.Create(ctx, &capture.CapturePairing{DeviceCodeHash: "dc" + pulid.MustNew("x_").String(), UserCode: "BCDFGHJK",
		MachineName: "M", AgentVersion: "1", Architecture: capture.ArchitectureX64, ExpiresAt: 5})
	require.NoError(t, err)
	_, err = pairings.GetOpenByUserCode(ctx, "BCDFGHJK")
	require.NoError(t, err)
	n, err := pairings.ExpireStale(ctx, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, 1)
	_, err = pairings.GetOpenByUserCode(ctx, "BCDFGHJK")
	assert.True(t, errortypes.IsNotFoundError(err))

	profiles := NewProfileRepository(p)
	p1, err := profiles.Create(ctx, &capture.CaptureProfile{OrganizationID: orgID, BusinessUnitID: buID,
		Name: "A" + pulid.MustNew("x_").String(), IsDefault: true,
		SeparatorStrategies: []capture.SeparatorStrategy{capture.SeparatorPatchCode}})
	require.NoError(t, err)
	p2, err := profiles.Create(ctx, &capture.CaptureProfile{OrganizationID: orgID, BusinessUnitID: buID,
		Name: "B" + pulid.MustNew("x_").String(), IsDefault: true})
	require.NoError(t, err)
	def, err := profiles.GetDefault(ctx, ti)
	require.NoError(t, err)
	assert.Equal(t, p2.ID, def.ID, "a new default clears the old")
	old, err := profiles.GetByID(ctx, repositories.GetCaptureProfileByIDRequest{ID: p1.ID, TenantInfo: ti})
	require.NoError(t, err)
	assert.Equal(t, []capture.SeparatorStrategy{capture.SeparatorPatchCode}, old.SeparatorStrategies)
	old.IsDefault = true
	_, err = profiles.Update(ctx, old)
	require.NoError(t, err)
	plist, err := profiles.List(ctx, &repositories.ListCaptureProfilesRequest{Filter: &pagination.QueryOptions{TenantInfo: ti}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, plist.Total, 2)

	requests := NewRequestRepository(p)
	req, err := requests.Create(ctx, &capture.CaptureRequest{OrganizationID: orgID, BusinessUnitID: buID, UserID: userID,
		DeviceID: dev.ID, Mode: capture.RequestModeScan, TargetType: "shipment", TargetID: "shp_1", ProfileID: &p1.ID, ExpiresAt: 5})
	require.NoError(t, err)
	open, err := requests.ListOpen(ctx, repositories.ListOpenCaptureRequestsRequest{TenantInfo: ti, DeviceID: dev.ID})
	require.NoError(t, err)
	require.Len(t, open, 1)
	require.NotNil(t, open[0].CaptureProfile)
	exp, err := requests.ListExpired(ctx, 10, 10)
	require.NoError(t, err)
	assert.Len(t, exp, 1)
	fr, err := requests.ListForTarget(ctx, repositories.ListCaptureRequestsForTargetRequest{TenantInfo: ti, TargetType: "shipment", TargetID: "shp_1"})
	require.NoError(t, err)
	assert.Len(t, fr, 1)
	_, err = requests.GetByID(ctx, repositories.GetCaptureRequestByIDRequest{ID: req.ID, TenantInfo: ti})
	require.NoError(t, err)

	batches := NewBatchRepository(p)
	b, err := batches.Create(ctx, &capture.CaptureBatch{OrganizationID: orgID, BusinessUnitID: buID, UserID: userID,
		DeviceID: dev.ID, ClientKey: "k" + pulid.MustNew("x_").String(), Source: capture.SourceScan, RetainUntil: 1,
		Settings: capture.Settings{DPI: 300, Refused: []string{"CAP_DUPLEXENABLED"}}})
	require.NoError(t, err)
	_, err = batches.Create(ctx, &capture.CaptureBatch{OrganizationID: orgID, BusinessUnitID: buID, UserID: userID,
		DeviceID: dev.ID, ClientKey: b.ClientKey, Source: capture.SourceScan, RetainUntil: 1})
	require.Error(t, err)
	byKey, err := batches.GetByClientKey(ctx, repositories.GetCaptureBatchByClientKeyRequest{TenantInfo: ti, DeviceID: dev.ID, ClientKey: b.ClientKey})
	require.NoError(t, err)
	assert.Equal(t, b.ID, byKey.ID)

	pages := NewPageRepository(p)
	pg, inserted, err := pages.Insert(ctx, &capture.CapturePage{OrganizationID: orgID, BusinessUnitID: buID, BatchID: b.ID,
		Sequence: 1, StoragePath: "s", ChecksumSHA256: "c", ByteSize: 1, ContentType: "application/pdf",
		Markers: capture.PageMarkers{PatchCode: "T"}})
	require.NoError(t, err)
	assert.True(t, inserted)
	again, inserted, err := pages.Insert(ctx, &capture.CapturePage{OrganizationID: orgID, BusinessUnitID: buID, BatchID: b.ID,
		Sequence: 1, StoragePath: "s2", ChecksumSHA256: "c2", ByteSize: 1, ContentType: "application/pdf"})
	require.NoError(t, err)
	assert.False(t, inserted)
	assert.Equal(t, pg.ID, again.ID)
	assert.Equal(t, "T", again.Markers.PatchCode)
	require.NoError(t, batches.IncrementReceived(ctx, repositories.IncrementCaptureBatchPagesRequest{ID: b.ID, TenantInfo: ti}))
	score := 0.5
	pg.BlankScore = &score
	_, err = pages.Update(ctx, pg)
	require.NoError(t, err)

	items := NewItemRepository(p)
	require.NoError(t, items.ReplaceOpen(ctx, &repositories.ReplaceOpenCaptureItemsRequest{BatchID: b.ID, TenantInfo: ti,
		Items: []*capture.CaptureItem{{ID: pulid.MustNew("citm_"), OrganizationID: orgID, BusinessUnitID: buID,
			BatchID: b.ID, Position: 1, Status: capture.ItemProposed, PageIDs: []pulid.ID{pg.ID}}}}))
	its, err := items.ListByBatch(ctx, repositories.ListCaptureItemsRequest{BatchID: b.ID, TenantInfo: ti})
	require.NoError(t, err)
	require.Len(t, its, 1)
	assert.Equal(t, []pulid.ID{pg.ID}, its[0].PageIDs)
	its[0].Suggest(capture.Target{ResourceType: "shipment", ResourceID: &pg.ID}, capture.SuggestionClassifier, 0.8, "r")
	_, err = items.Update(ctx, its[0])
	require.NoError(t, err)

	full, err := batches.GetByID(ctx, &repositories.GetCaptureBatchByIDRequest{ID: b.ID, TenantInfo: ti, IncludePages: true, IncludeItems: true})
	require.NoError(t, err)
	assert.Len(t, full.Pages, 1)
	assert.Len(t, full.Items, 1)
	assert.Equal(t, 1, full.ReceivedPageCount)
	assert.Equal(t, []string{"CAP_DUPLEXENABLED"}, full.Settings.Refused)
	full.Status = capture.BatchReady
	full.ReceivedPageCount = 999
	full.Pages, full.Items = nil, nil
	updated, err := batches.Update(ctx, full)
	require.NoError(t, err)
	assert.Equal(t, 1, updated.ReceivedPageCount, "update never overwrites the concurrent page counter")

	due, err := batches.ListRetentionDue(ctx, repositories.ListRetentionDueCaptureBatchesRequest{Now: 10})
	require.NoError(t, err)
	assert.NotEmpty(t, due)
	stale, err := batches.ListStale(ctx, repositories.ListStaleCaptureBatchesRequest{Statuses: []capture.BatchStatus{capture.BatchReady}, UpdatedBefore: 1 << 40})
	require.NoError(t, err)
	assert.NotEmpty(t, stale)
	bl, err := batches.List(ctx, &repositories.ListCaptureBatchesRequest{Filter: &pagination.QueryOptions{TenantInfo: ti},
		Statuses: []capture.BatchStatus{capture.BatchReady}, UserID: userID})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, bl.Total, 1)

	sheets := NewCoverSheetRepository(p)
	cs := &capture.CaptureCoverSheet{OrganizationID: orgID, BusinessUnitID: buID, TokenHash: "h" + pulid.MustNew("x_").String(),
		IssuedByID: userID, ExpiresAt: 100}
	require.NoError(t, sheets.CreateMany(ctx, []*capture.CaptureCoverSheet{cs}))
	_, err = sheets.GetByTokenHash(ctx, repositories.GetCaptureCoverSheetByTokenRequest{TenantInfo: ti, TokenHash: cs.TokenHash})
	require.NoError(t, err)
	_, err = sheets.GetByTokenHash(ctx, repositories.GetCaptureCoverSheetByTokenRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: buID}, TokenHash: cs.TokenHash})
	assert.True(t, errortypes.IsNotFoundError(err), "another tenant cannot resolve the sheet")
	require.NoError(t, sheets.MarkUsed(ctx, repositories.MarkCaptureCoverSheetUsedRequest{ID: cs.ID, TenantInfo: ti, UsedAt: 3}))
	used, err := sheets.GetByID(ctx, repositories.GetCaptureCoverSheetByIDRequest{ID: cs.ID, TenantInfo: ti})
	require.NoError(t, err)
	assert.Equal(t, 1, used.UseCount)

	require.NoError(t, batches.Delete(ctx, repositories.DeleteCaptureBatchRequest{ID: b.ID, TenantInfo: ti}))
	left, err := pages.ListByBatch(ctx, repositories.ListCapturePagesRequest{BatchID: b.ID, TenantInfo: ti})
	require.NoError(t, err)
	assert.Empty(t, left, "deleting the batch takes its pages")
}
