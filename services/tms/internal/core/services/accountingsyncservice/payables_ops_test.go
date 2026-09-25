package accountingsyncservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func driverEnqueueRequests(h *harness, workerID pulid.ID) []*services.AccountingSyncEnqueueRequest {
	accts := newPayableAccounts()
	settlement := h.driverSettlement(accts)
	return []*services.AccountingSyncEnqueueRequest{
		settlementRequest(h, accountingsync.SyncObjectDriverBill, accountingsync.SyncOperationCreate, settlement),
		settlementRequest(h, accountingsync.SyncObjectDriverBill, accountingsync.SyncOperationVoid, settlement),
		settlementRequest(h, accountingsync.SyncObjectDriverBillPay, accountingsync.SyncOperationCreate, settlement),
		services.VendorSyncRequest(h.tenant, accountingsync.SyncObjectDriverVendor, workerID, "Dana Ortiz", 2),
		{
			TenantInfo:  h.tenant,
			ObjectType:  accountingsync.SyncObjectDriverVendor,
			ObjectID:    workerID,
			Operation:   accountingsync.SyncOperationCreate,
			Revision:    1,
			SourceEvent: accountingsync.SyncSourceDependencyOf,
		},
	}
}

func TestDriverRecordsAreEnqueuedOnlyWhileTheDriverSettingIsOn(t *testing.T) {
	t.Parallel()

	t.Run("off", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		workerID := pulid.MustNew("wrk_")
		h.confirm(driverVendorTarget(workerID), "qb-vendor-dana")

		for _, req := range driverEnqueueRequests(h, workerID) {
			require.NoError(t, h.enqueuer.Enqueue(t.Context(), req))
		}

		assert.Empty(t, h.records.all(), "owner-operator records wait for the connection to opt in")
		assert.Empty(t, h.dispatcher.kicked())
	})

	t.Run("on", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.sendDriverSettlements(syncEnabledAt)
		workerID := pulid.MustNew("wrk_")
		h.confirm(driverVendorTarget(workerID), "qb-vendor-dana")
		requests := driverEnqueueRequests(h, workerID)

		for _, req := range requests {
			require.NoError(t, h.enqueuer.Enqueue(t.Context(), req))
		}

		require.Len(t, h.records.all(), len(requests))
		for _, req := range requests {
			record := h.only(t, req.ObjectType, req.ObjectID, req.Operation)
			assert.Equal(t, req.SourceEvent, record.SourceEvent)
		}
	})

	t.Run("turned off again", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.sendDriverSettlements(syncEnabledAt)
		h.updateConnection(func(conn *accountingsync.AccountingConnection) {
			conn.SetDriverSettlements(false, time.Now().Unix())
		})
		workerID := pulid.MustNew("wrk_")
		h.confirm(driverVendorTarget(workerID), "qb-vendor-dana")

		for _, req := range driverEnqueueRequests(h, workerID) {
			require.NoError(t, h.enqueuer.Enqueue(t.Context(), req))
		}

		assert.Empty(t, h.records.all())
	})
}

func TestCarrierRecordsDoNotNeedTheDriverSetting(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	h.confirm(carrierVendorTarget(settlement.PartyID), "qb-vendor-swift")

	h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement)
	h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationVoid, settlement)
	h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate, settlement)
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.VendorSyncRequest(
		h.tenant, accountingsync.SyncObjectCarrierVendor, settlement.PartyID, "Swift Haulers", 2,
	)))

	assert.Len(t, h.records.all(), 4)
}

func TestVendorUpdatesAreEnqueuedOnlyForConfirmedMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectType accountingsync.SyncObjectType
		target     func(pulid.ID) mappingTarget
		prefix     string
		state      accountingsync.MappingState
		mapped     bool
		want       int
	}{
		{name: "carrier with no mapping", objectType: accountingsync.SyncObjectCarrierVendor,
			target: carrierVendorTarget, prefix: "car_"},
		{name: "carrier with an unmatched mapping", objectType: accountingsync.SyncObjectCarrierVendor,
			target: carrierVendorTarget, prefix: "car_", mapped: true, state: accountingsync.MappingStateUnmatched},
		{name: "carrier with a proposed mapping", objectType: accountingsync.SyncObjectCarrierVendor,
			target: carrierVendorTarget, prefix: "car_", mapped: true, state: accountingsync.MappingStateProposed},
		{name: "carrier with a confirmed mapping", objectType: accountingsync.SyncObjectCarrierVendor,
			target: carrierVendorTarget, prefix: "car_", mapped: true, state: accountingsync.MappingStateConfirmed,
			want: 1},
		{name: "driver with no mapping", objectType: accountingsync.SyncObjectDriverVendor,
			target: driverVendorTarget, prefix: "wrk_"},
		{name: "driver with a confirmed mapping", objectType: accountingsync.SyncObjectDriverVendor,
			target: driverVendorTarget, prefix: "wrk_", mapped: true, state: accountingsync.MappingStateConfirmed,
			want: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			h.sendDriverSettlements(syncEnabledAt)
			partyID := pulid.MustNew(tc.prefix)
			if tc.mapped {
				h.mappings.set(h.conn, tc.target(partyID), tc.state, "qb-vendor")
			}

			require.NoError(t, h.enqueuer.Enqueue(t.Context(),
				services.VendorSyncRequest(h.tenant, tc.objectType, partyID, "Party", 3)))

			assert.Len(t, h.records.find(tc.objectType, partyID, accountingsync.SyncOperationUpdate), tc.want)
		})
	}
}

func TestSettlementPostedBeforeTheStartDateIsNotEnqueued(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	early := syncStartDate - 1
	settlement.PostedAt = &early

	require.NoError(t, h.enqueuer.Enqueue(t.Context(), settlementRequest(
		h, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, settlement,
	)))

	assert.Empty(t, h.records.all())
}

func TestSafetyNetCoversPayables(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sendDriverSettlements(syncEnabledAt)

	h.safetyNet(t)

	type kind struct {
		objectType accountingsync.SyncObjectType
		operation  accountingsync.SyncOperation
	}
	seen := map[kind]bool{}
	for _, call := range h.records.candidateCalls {
		seen[kind{call.ObjectType, call.Operation}] = true
	}
	for _, want := range []kind{
		{accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate},
		{accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationVoid},
		{accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate},
		{accountingsync.SyncObjectDriverBill, accountingsync.SyncOperationCreate},
		{accountingsync.SyncObjectDriverBill, accountingsync.SyncOperationVoid},
		{accountingsync.SyncObjectDriverBillPay, accountingsync.SyncOperationCreate},
	} {
		assert.True(t, seen[want], "%s %s is checked", want.objectType, want.operation)
	}
}

func TestSafetyNetSkipsDriverSettlementsWhileTheSettingIsOff(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	carrier := h.addCandidates(accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, 1,
		syncEnabledAt+10)
	driver := h.addCandidates(accountingsync.SyncObjectDriverBill, accountingsync.SyncOperationCreate, 1,
		syncEnabledAt+10)
	h.addCandidates(accountingsync.SyncObjectDriverBillPay, accountingsync.SyncOperationCreate, 1,
		syncEnabledAt+10)

	result := h.safetyNet(t)

	assert.Equal(t, 1, result.Queued)
	h.only(t, carrier[0].ObjectType, carrier[0].ObjectID, accountingsync.SyncOperationCreate)
	assert.Empty(t, h.records.find(driver[0].ObjectType, driver[0].ObjectID, accountingsync.SyncOperationCreate))
	for _, call := range h.records.candidateCalls {
		assert.False(t, call.ObjectType.NeedsDriverSettlements(),
			"%s is not even looked for while owner-operator settlements are off", call.ObjectType)
	}
}

func TestSafetyNetLooksForDriverSettlementsFromTheLaterOfTheTwoTimes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		driverOn     int64
		wantFrom     int64
		wantQueued   []int64
		wantUnqueued []int64
	}{
		{
			name:         "turned on after sync began",
			driverOn:     syncEnabledAt + 1000,
			wantFrom:     syncEnabledAt + 1000,
			wantQueued:   []int64{syncEnabledAt + 1500},
			wantUnqueued: []int64{syncEnabledAt + 500, syncEnabledAt - 10},
		},
		{
			name:         "turned on before sync began",
			driverOn:     syncEnabledAt - 1000,
			wantFrom:     syncEnabledAt,
			wantQueued:   []int64{syncEnabledAt + 500},
			wantUnqueued: []int64{syncEnabledAt - 500},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			h.sendDriverSettlements(tc.driverOn)
			queued := make([]pulid.ID, 0, len(tc.wantQueued))
			for _, at := range tc.wantQueued {
				queued = append(queued, h.addCandidates(accountingsync.SyncObjectDriverBill,
					accountingsync.SyncOperationCreate, 1, at)[0].ObjectID)
			}
			unqueued := make([]pulid.ID, 0, len(tc.wantUnqueued))
			for _, at := range tc.wantUnqueued {
				unqueued = append(unqueued, h.addCandidates(accountingsync.SyncObjectDriverBill,
					accountingsync.SyncOperationCreate, 1, at)[0].ObjectID)
			}
			carrier := h.addCandidates(accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, 1,
				syncEnabledAt+500)

			h.safetyNet(t)

			for _, call := range h.records.candidateCalls {
				require.NotNil(t, call.PostedFrom)
				if call.ObjectType.NeedsDriverSettlements() {
					assert.Equal(t, tc.wantFrom, *call.PostedFrom, "%s %s", call.ObjectType, call.Operation)
				} else {
					assert.Equal(t, syncEnabledAt, *call.PostedFrom, "%s %s", call.ObjectType, call.Operation)
				}
			}
			for _, id := range queued {
				assert.Len(t, h.records.find(accountingsync.SyncObjectDriverBill, id,
					accountingsync.SyncOperationCreate), 1)
			}
			for _, id := range unqueued {
				assert.Empty(t, h.records.find(accountingsync.SyncObjectDriverBill, id,
					accountingsync.SyncOperationCreate))
			}
			assert.Len(t, h.records.find(accountingsync.SyncObjectCarrierBill, carrier[0].ObjectID,
				accountingsync.SyncOperationCreate), 1, "carrier settlements keep the sync window")
		})
	}
}

func TestRequestBackfillReachesDriverSettlementsOnlyWhenTheSettingIsOn(t *testing.T) {
	t.Parallel()

	t.Run("off leaves them out", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		backfill, err := h.requestBackfill(t, nil)
		require.NoError(t, err)

		assert.Equal(t, []accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
			accountingsync.SyncObjectCreditMemo,
			accountingsync.SyncObjectCustomerPayment,
			accountingsync.SyncObjectCreditApplication,
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncObjectCarrierBillPay,
		}, backfill.ObjectTypes)
	})

	t.Run("on includes them", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.sendDriverSettlements(syncEnabledAt)

		backfill, err := h.requestBackfill(t, nil)
		require.NoError(t, err)

		assert.Equal(t, accountingsync.BackfillObjectTypes(), backfill.ObjectTypes)
	})

	for _, typ := range []accountingsync.SyncObjectType{
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay,
	} {
		t.Run("off rejects "+string(typ), func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)

			_, err := h.requestBackfill(t, func(r *services.RequestAccountingBackfillRequest) {
				r.ObjectTypes = []accountingsync.SyncObjectType{accountingsync.SyncObjectCarrierBill, typ}
			})

			requireValidationField(t, err, "objectTypes")
			assert.Contains(t, err.Error(), "turn them on first")
			assert.Empty(t, h.dispatcher.started())
		})
	}

	t.Run("vendors are not backfilled", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.sendDriverSettlements(syncEnabledAt)

		_, err := h.requestBackfill(t, func(r *services.RequestAccountingBackfillRequest) {
			r.ObjectTypes = []accountingsync.SyncObjectType{accountingsync.SyncObjectCarrierVendor}
		})

		requireValidationField(t, err, "objectTypes")
	})
}

func TestRunningBackfillSkipsDriverSettlementsOnceTheSettingIsTurnedOff(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sendDriverSettlements(syncEnabledAt)
	carrier := h.addCandidates(accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate, 1,
		syncStartDate+100)
	driver := h.addCandidates(accountingsync.SyncObjectDriverBill, accountingsync.SyncOperationCreate, 2,
		syncStartDate+100)
	driverPay := h.addCandidates(accountingsync.SyncObjectDriverBillPay, accountingsync.SyncOperationCreate, 1,
		syncStartDate+100)
	backfill := h.newBackfill(t,
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay,
	)

	first := h.backfillStep(t, backfill.ID)

	assert.Equal(t, 1, first.Enqueued)
	assert.False(t, first.Done)
	assert.Equal(t, accountingsync.SyncObjectDriverBill, h.backfills.get(backfill.ID).Cursor.ObjectType)
	h.only(t, accountingsync.SyncObjectCarrierBill, carrier[0].ObjectID, accountingsync.SyncOperationCreate)

	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.SetDriverSettlements(false, time.Now().Unix())
	})
	calls := len(h.records.candidateCalls)

	second := h.backfillStep(t, backfill.ID)

	assert.False(t, second.Done)
	assert.Zero(t, second.Enqueued)
	assert.Equal(t, accountingsync.SyncObjectDriverBillPay, h.backfills.get(backfill.ID).Cursor.ObjectType)

	third := h.backfillStep(t, backfill.ID)

	assert.True(t, third.Done)
	saved := h.backfills.get(backfill.ID)
	assert.Equal(t, accountingsync.BackfillStatusCompleted, saved.Status)
	assert.Equal(t, 1, saved.EnqueuedCount)
	assert.Len(t, h.records.candidateCalls, calls, "a skipped type is not even listed")
	for _, candidate := range append(driver, driverPay...) {
		assert.Empty(t, h.records.find(candidate.ObjectType, candidate.ObjectID, accountingsync.SyncOperationCreate))
	}
}

func TestRunningBackfillSkipsATrailingDriverTypeAndCompletes(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sendDriverSettlements(syncEnabledAt)
	backfill := h.newBackfill(t, accountingsync.SyncObjectDriverBillPay)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.SetDriverSettlements(false, time.Now().Unix())
	})

	result := h.backfillStep(t, backfill.ID)

	assert.True(t, result.Done)
	assert.Equal(t, accountingsync.BackfillStatusCompleted, h.backfills.get(backfill.ID).Status)
	assert.Empty(t, h.records.candidateCalls)
}

func (h *harness) updateSettings(
	t *testing.T,
	autoSync, drivers bool,
) (*accountingsync.AccountingConnection, error) {
	t.Helper()
	return h.svc.UpdateSettings(t.Context(), &services.UpdateAccountingSyncSettingsRequest{
		TenantInfo:        h.tenant,
		UserID:            h.userID,
		IntegrationType:   integration.TypeQuickBooksOnline,
		AutoSync:          autoSync,
		DriverSettlements: drivers,
	})
}

func TestUpdateSettingsNeedsSetupToBeComplete(t *testing.T) {
	t.Parallel()

	for _, step := range []accountingsync.SetupStep{
		accountingsync.SetupStepMappings,
		accountingsync.SetupStepStartDate,
	} {
		t.Run(string(step), func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			h.updateConnection(func(conn *accountingsync.AccountingConnection) { conn.SetupStep = step })

			_, err := h.updateSettings(t, false, true)

			require.Error(t, err)
			assert.True(t, errortypes.IsBusinessError(err))
			stored := h.connections.get(h.conn.ID)
			assert.Nil(t, stored.DriverSettlementsEnabledAt)
			assert.True(t, stored.AutoSync)
			assert.Zero(t, h.audit.count())
		})
	}
}

func TestUpdateSettingsTogglesTheDriverSettingAndAutoSync(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	longAgo := syncEnabledAt + 60

	on, err := h.updateSettings(t, false, true)
	require.NoError(t, err)

	assert.False(t, on.AutoSync)
	nearNow(t, 0, on.DriverSettlementsEnabledAt)
	stored := h.connections.get(h.conn.ID)
	assert.False(t, stored.AutoSync)
	nearNow(t, 0, stored.DriverSettlementsEnabledAt)
	assert.Equal(t, "Started sending owner-operator settlements to QuickBooks Online", h.audit.lastComment())

	h.sendDriverSettlements(longAgo)
	unchanged, err := h.updateSettings(t, true, true)
	require.NoError(t, err)

	assert.True(t, unchanged.AutoSync)
	require.NotNil(t, unchanged.DriverSettlementsEnabledAt)
	assert.Equal(t, longAgo, *unchanged.DriverSettlementsEnabledAt,
		"keeping the setting on keeps the time it was turned on")
	assert.Equal(t, "Changed how documents are sent to QuickBooks Online", h.audit.lastComment())

	off, err := h.updateSettings(t, true, false)
	require.NoError(t, err)

	assert.Nil(t, off.DriverSettlementsEnabledAt)
	assert.Nil(t, h.connections.get(h.conn.ID).DriverSettlementsEnabledAt)
	assert.Equal(t, "Stopped sending owner-operator settlements to QuickBooks Online", h.audit.lastComment())

	again, err := h.updateSettings(t, true, true)
	require.NoError(t, err)

	require.NotNil(t, again.DriverSettlementsEnabledAt)
	assert.NotEqual(t, longAgo, *again.DriverSettlementsEnabledAt, "turning it on again stamps a new time")
	nearNow(t, 0, again.DriverSettlementsEnabledAt)
	assert.Equal(t, 4, h.audit.count())
	assert.Equal(t, syncEnabledAt, *h.connections.get(h.conn.ID).SyncEnabledAt,
		"changing settings never moves when sync began")
}

func TestUpdateSettingsAutoSyncOnlyChange(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	updated, err := h.updateSettings(t, false, false)
	require.NoError(t, err)

	assert.False(t, updated.AutoSync)
	assert.Nil(t, updated.DriverSettlementsEnabledAt)
	assert.Equal(t, "Changed how documents are sent to QuickBooks Online", h.audit.lastComment())
}

func TestEnableSyncWithDriverSettlementsStampsBothTimesAndBackfillsThem(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()

	conn, err := h.svc.EnableSync(t.Context(), &services.EnableAccountingSyncRequest{
		TenantInfo:        h.tenant,
		UserID:            h.userID,
		IntegrationType:   integration.TypeQuickBooksOnline,
		StartDate:         syncStartDate,
		AutoSync:          true,
		DriverSettlements: true,
		Backfill:          true,
	})
	require.NoError(t, err)

	require.NotNil(t, conn.DriverSettlementsEnabledAt)
	require.NotNil(t, conn.SyncEnabledAt)
	assert.Equal(t, *conn.SyncEnabledAt, *conn.DriverSettlementsEnabledAt)
	started := h.dispatcher.started()
	require.Len(t, started, 1)
	assert.Equal(t, accountingsync.BackfillObjectTypes(), started[0].ObjectTypes)
}

func TestEnableSyncLeavesDriverSettlementsOffByDefault(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()

	conn, err := h.enableSync(t, syncStartDate, false)
	require.NoError(t, err)

	assert.Nil(t, conn.DriverSettlementsEnabledAt)
	assert.False(t, conn.SyncsDriverSettlements())
}
