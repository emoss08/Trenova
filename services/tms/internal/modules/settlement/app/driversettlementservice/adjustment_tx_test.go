package driversettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type txMarker struct{}

type recordingTransactor struct {
	dbtest.NopConnection
	rolledBack bool
	committed  bool
}

func (r *recordingTransactor) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	err := fn(context.WithValue(ctx, txMarker{}, true))
	r.rolledBack = err != nil
	r.committed = err == nil
	return err
}

func inTx(ctx context.Context) bool {
	marked, ok := ctx.Value(txMarker{}).(bool)
	return ok && marked
}

type lineStore struct {
	driversettlementports.DriverSettlementRepository
	current   *driversettlement.Settlement
	updateErr error
	readInTx  bool
	linesInTx bool
	saveInTx  bool
	replaced  int
}

func (s *lineStore) GetByID(
	ctx context.Context,
	_ driversettlementports.GetDriverSettlementByIDRequest,
) (*driversettlement.Settlement, error) {
	s.readInTx = inTx(ctx)
	copied := *s.current
	copied.Lines = append([]*driversettlement.SettlementLine(nil), s.current.Lines...)
	return &copied, nil
}

func (s *lineStore) ReplaceLines(ctx context.Context, _ *driversettlement.Settlement) error {
	s.linesInTx = inTx(ctx)
	s.replaced++
	return nil
}

func (s *lineStore) Update(
	ctx context.Context,
	entity *driversettlement.Settlement,
) (*driversettlement.Settlement, error) {
	s.saveInTx = inTx(ctx)
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	copied := *entity
	return &copied, nil
}

func draftWithAdjustment() (*driversettlement.Settlement, pulid.ID) {
	lineID := pulid.MustNew("dstll_")
	return &driversettlement.Settlement{
		ID:             pulid.MustNew("dstl_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         driversettlement.StatusDraft,
		Lines: []*driversettlement.SettlementLine{{
			ID:       lineID,
			Category: driversettlement.LineCategoryAdjustment,
		}},
	}, lineID
}

func adjustmentInput() *AdjustmentLineInput {
	return &AdjustmentLineInput{Description: "Tolls", AmountMinor: 2500}
}

func TestAddAdjustmentLineRollsBackTheLinesWhenTheVersionConflicts(t *testing.T) {
	t.Parallel()

	entity, _ := draftWithAdjustment()
	store := &lineStore{
		current:   entity,
		updateErr: errortypes.NewConflictError("The settlement changed; reload and try again"),
	}
	db := &recordingTransactor{}
	svc := &Service{db: db, settlementRepo: store}

	updated, err := svc.AddAdjustmentLine(
		t.Context(),
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.ID,
		adjustmentInput(),
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.True(t, store.readInTx, "the read shares the transaction")
	assert.True(t, store.linesInTx, "the lines share the transaction")
	assert.True(t, store.saveInTx, "the version check shares the transaction")
	assert.Equal(t, 1, store.replaced)
	assert.True(t, db.rolledBack, "a version conflict undoes the replaced lines")
	assert.False(t, db.committed)
}

func TestRemoveAdjustmentLineRollsBackTheLinesWhenTheVersionConflicts(t *testing.T) {
	t.Parallel()

	entity, lineID := draftWithAdjustment()
	store := &lineStore{
		current:   entity,
		updateErr: errortypes.NewConflictError("The settlement changed; reload and try again"),
	}
	db := &recordingTransactor{}
	svc := &Service{db: db, settlementRepo: store}

	updated, err := svc.RemoveAdjustmentLine(
		t.Context(),
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.ID,
		lineID,
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.True(t, store.readInTx)
	assert.True(t, store.linesInTx)
	assert.True(t, store.saveInTx)
	assert.True(t, db.rolledBack)
}

func TestAddAdjustmentLineCommitsTheLinesAndTheVersionTogether(t *testing.T) {
	t.Parallel()

	entity, _ := draftWithAdjustment()
	store := &lineStore{current: entity}
	db := &recordingTransactor{}
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()
	svc := &Service{
		l:              zap.NewNop(),
		db:             db,
		settlementRepo: store,
		auditService:   silentAudit{},
		realtime:       realtime,
	}

	updated, err := svc.AddAdjustmentLine(
		t.Context(),
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.ID,
		adjustmentInput(),
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Len(t, updated.Lines, 2)
	assert.True(t, db.committed)
	assert.Equal(t, 1, store.replaced)
}

func TestAddAdjustmentLineRefusesASettlementThatIsNotEditable(t *testing.T) {
	t.Parallel()

	entity, _ := draftWithAdjustment()
	entity.Status = driversettlement.StatusPosted
	store := &lineStore{current: entity}
	db := &recordingTransactor{}
	svc := &Service{db: db, settlementRepo: store}

	_, err := svc.AddAdjustmentLine(
		t.Context(),
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.ID,
		adjustmentInput(),
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only draft or pending settlements can be adjusted")
	assert.Zero(t, store.replaced)
}
