//go:build integration

package shipmentcommentrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx      context.Context
	repo     shipmentports.ShipmentCommentRepository
	tenant   pagination.TenantInfo
	entity   *shipment.Shipment
	otherCtx context.Context
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:          "BOL-CMT",
		ProNumber:    "PRO-CMT",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
	})

	return &harness{
		ctx: data.Ctx(ctx),
		repo: shipmentcommentrepository.New(shipmentcommentrepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		tenant:   tenant,
		entity:   graph.Shipment,
		otherCtx: seedtest.SeedAdditionalTenant(t, ctx, db, "OT").Ctx(ctx),
	}
}

func (h *harness) comment(text string, parent *pulid.ID) *shipment.ShipmentComment {
	return &shipment.ShipmentComment{
		ShipmentID:      h.entity.ID,
		OrganizationID:  h.tenant.OrgID,
		BusinessUnitID:  h.tenant.BuID,
		UserID:          h.tenant.UserID,
		ParentCommentID: parent,
		Comment:         text,
		MentionedUsers: []*shipment.ShipmentCommentMention{
			{MentionedUserID: h.tenant.UserID},
		},
	}
}

func (h *harness) list(
	t *testing.T,
	ctx context.Context,
	parent *pulid.ID,
) []*shipment.ShipmentComment {
	t.Helper()

	result, err := h.repo.ListByShipmentID(ctx, &shipmentports.ListShipmentCommentsRequest{
		Filter:          &pagination.QueryOptions{TenantInfo: h.tenant},
		Cursor:          pagination.CursorInfo{IncludeTotalCount: true},
		ShipmentID:      h.entity.ID,
		ParentCommentID: parent,
	})
	require.NoError(t, err)
	return result.Items
}

func TestCreate_StoresTheCommentWithDefaultsMentionsAndReplies(t *testing.T) {
	h := newHarness(t)

	root, err := h.repo.Create(h.ctx, h.comment("Driver is running late", nil))
	require.NoError(t, err)
	assert.Equal(t, shipment.CommentTypeInternal, root.Type)
	assert.Equal(t, shipment.CommentPriorityNormal, root.Priority)
	require.Len(t, root.MentionedUsers, 1)
	require.NotNil(t, root.MentionedUsers[0].MentionedUser)

	_, err = h.repo.Create(h.ctx, h.comment("Updated ETA", &root.ID))
	require.NoError(t, err)

	roots := h.list(t, h.ctx, nil)
	require.Len(t, roots, 1)
	assert.Equal(t, int64(1), roots[0].ReplyCount)
	assert.Len(t, h.list(t, h.ctx, &root.ID), 1)
	assert.Empty(t, h.list(t, h.otherCtx, nil), "another tenant never sees the thread")

	count, err := h.repo.GetCountByShipmentID(h.ctx, &shipmentports.GetShipmentCommentCountRequest{
		TenantInfo: h.tenant,
		ShipmentID: h.entity.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestCreate_UnderASystemScopeReReadsTheCommentInItsOwnTenant(t *testing.T) {
	h := newHarness(t)
	systemCtx := dbscope.WithSystem(h.ctx, "test a system-scoped shipment comment")

	root, err := h.repo.Create(systemCtx, h.comment("Synced from the linked organization", nil))
	require.NoError(t, err)
	assert.Equal(t, h.tenant.OrgID, root.OrganizationID)
	require.Len(t, root.MentionedUsers, 1)

	reply, err := h.repo.Create(systemCtx, h.comment("Reply", &root.ID))
	require.NoError(t, err)
	assert.Equal(t, h.tenant.OrgID, reply.OrganizationID)

	reread, err := h.repo.GetByID(systemCtx, &shipmentports.GetShipmentCommentByIDRequest{
		CommentID:  root.ID,
		ShipmentID: h.entity.ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), reread.ReplyCount)

	stranger := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: h.tenant.BuID}
	_, err = h.repo.GetByID(systemCtx, &shipmentports.GetShipmentCommentByIDRequest{
		CommentID:  root.ID,
		ShipmentID: h.entity.ID,
		TenantInfo: stranger,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "a system read is still held to the named tenant")
}

func TestPinAcknowledgeAndSoftDelete(t *testing.T) {
	h := newHarness(t)
	root, err := h.repo.Create(h.ctx, h.comment("Check the seal", nil))
	require.NoError(t, err)

	pinned, err := h.repo.SetPinned(h.ctx, &shipmentports.SetShipmentCommentPinnedRequest{
		TenantInfo: h.tenant,
		ShipmentID: h.entity.ID,
		CommentID:  root.ID,
		ActorID:    h.tenant.UserID,
		Pinned:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, pinned.PinnedAt)

	ack := &shipmentports.AcknowledgeShipmentCommentRequest{
		TenantInfo: h.tenant,
		ShipmentID: h.entity.ID,
		CommentID:  root.ID,
		UserID:     h.tenant.UserID,
	}
	first, err := h.repo.Acknowledge(h.ctx, ack)
	require.NoError(t, err)
	assert.True(t, first)
	second, err := h.repo.Acknowledge(h.ctx, ack)
	require.NoError(t, err)
	assert.False(t, second, "a second acknowledgment is skipped")

	deleted, err := h.repo.SoftDelete(h.ctx, &shipmentports.SoftDeleteShipmentCommentRequest{
		TenantInfo: h.tenant,
		ShipmentID: h.entity.ID,
		CommentID:  root.ID,
		ActorID:    h.tenant.UserID,
	})
	require.NoError(t, err)
	require.NotNil(t, deleted.DeletedAt)
	assert.Empty(t, deleted.Comment)
	assert.Nil(t, deleted.PinnedAt)
	assert.Empty(t, deleted.MentionedUsers)
}
