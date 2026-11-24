package driversettlementworkspaceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type heldStore struct {
	driversettlementports.DriverSettlementRepository
	settlements map[pulid.ID]*driversettlement.Settlement
	saved       int
}

func (s *heldStore) GetByID(
	_ context.Context,
	req driversettlementports.GetDriverSettlementByIDRequest,
) (*driversettlement.Settlement, error) {
	copied := *s.settlements[req.ID]

	return &copied, nil
}

func (s *heldStore) Update(
	_ context.Context,
	entity *driversettlement.Settlement,
) (*driversettlement.Settlement, error) {
	s.saved++

	return entity, nil
}

func TestBulkActionsRefuseAHeldSettlementByNameAndReason(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	held := &driversettlement.Settlement{
		ID:               pulid.MustNew("dstl_"),
		OrganizationID:   tenant.OrgID,
		BusinessUnitID:   tenant.BuID,
		SettlementNumber: "DS-311",
		Status:           driversettlement.StatusApproved,
		CurrencyCode:     "USD",
	}
	require.NoError(t, held.PlaceHold(&driversettlement.HoldInput{
		Reason: driversettlement.HoldReasonMissingPaperwork,
		UserID: pulid.MustNew("usr_"),
		At:     1,
	}))
	store := &heldStore{settlements: map[pulid.ID]*driversettlement.Settlement{held.ID: held}}
	settlements := driversettlementservice.New(driversettlementservice.Params{
		Logger:         zap.NewNop(),
		DB:             dbtest.NopConnection{},
		SettlementRepo: store,
		Realtime:       &mocks.NoopRealtimeService{},
	})
	svc := New(Params{Settlements: settlements, SettlementRepo: store})

	for _, action := range []BulkActionType{BulkActionApprove, BulkActionPost} {
		payload, err := svc.BulkAction(t.Context(), &BulkActionRequest{
			TenantInfo:    tenant,
			SettlementIDs: []pulid.ID{held.ID},
			Action:        action,
		}, &permissionports.RequestActor{UserID: pulid.MustNew("usr_")})
		require.NoError(t, err, action)

		assert.Equal(t, 0, payload.SuccessCount, action)
		assert.Equal(t, 1, payload.FailureCount, action)
		require.Len(t, payload.Results, 1)
		assert.False(t, payload.Results[0].Success)
		assert.Equal(t,
			"Settlement DS-311 is on hold for missing paperwork; release the hold first",
			payload.Results[0].Error, action)
	}
	assert.Zero(t, store.saved, "a held settlement is never saved by a bulk action")
}
