package shipmentcommentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreate_NormalizesMentionsAuditsAndPublishes(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentCommentRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		AuditService: audit,
		Realtime:     realtime,
	})

	shipmentID := pulid.MustNew("shp_")
	mentionA := pulid.MustNew("usr_")
	mentionB := pulid.MustNew("usr_")

	entity := &shipment.ShipmentComment{
		ShipmentID:       shipmentID,
		OrganizationID:   testutil.TestOrgID,
		BusinessUnitID:   testutil.TestBuID,
		Comment:          "  hello world  ",
		MentionedUserIDs: []pulid.ID{mentionA, mentionB, mentionA},
	}

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.TenantInfo.OrgID == testutil.TestOrgID && req.TenantInfo.BuID == testutil.TestBuID
		})).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()

	userRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.GetUsersByIDsRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID &&
				len(req.UserIDs) == 2
		})).
		Return([]*tenant.User{{ID: mentionA, Name: "Alice"}, {ID: mentionB, Name: "Bob"}}, nil).
		Once()

	repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.UserID == testutil.TestUserID && comment.Comment == "hello world" && len(comment.MentionedUsers) == 2
		})).
		RunAndReturn(func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
			comment.ID = pulid.MustNew("shc_")
			return comment, nil
		}).
		Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything).
		Run(func(params *servicesport.LogActionParams, _ ...servicesport.LogOption) {
			assert.Equal(t, permission.ResourceShipmentComment, params.Resource)
			assert.Equal(t, testutil.TestOrgID, params.OrganizationID)
		}).
		Return(nil).Once()

	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *servicesport.PublishResourceInvalidationRequest) bool {
			return req.Resource == permission.ResourceShipmentComment.String() &&
				req.Action == "created" &&
				req.RecordID == shipmentID &&
				req.OrganizationID == testutil.TestOrgID &&
				req.BusinessUnitID == testutil.TestBuID &&
				req.ActorUserID == testutil.TestUserID
		})).
		Return(nil).Once()

	created, err := svc.Create(t.Context(), entity, testActor())

	require.NoError(t, err)
	assert.Equal(t, "hello world", created.Comment)
	assert.Len(t, created.MentionedUserIDs, 2)
	assert.Equal(t, testutil.TestUserID, created.UserID)
}

func TestCreate_RejectsUnknownMentionedUsers(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentCommentRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		AuditService: audit,
		Realtime:     realtime,
	})

	shipmentID := pulid.MustNew("shp_")
	entity := &shipment.ShipmentComment{
		ShipmentID:       shipmentID,
		OrganizationID:   testutil.TestOrgID,
		BusinessUnitID:   testutil.TestBuID,
		Comment:          "hello",
		MentionedUserIDs: []pulid.ID{pulid.MustNew("usr_")},
	}

	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.Shipment{ID: shipmentID}, nil).Once()
	userRepo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return([]*tenant.User{}, nil).Once()

	created, err := svc.Create(t.Context(), entity, testActor())

	require.Error(t, err)
	assert.Nil(t, created)
	assert.True(t, errortypes.IsError(err))
}

func TestUpdate_RequiresOwnerSetsEditedAtAndPublishes(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentCommentRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		AuditService: audit,
		Realtime:     realtime,
	})

	commentID := pulid.MustNew("shc_")
	shipmentID := pulid.MustNew("shp_")
	mentionID := pulid.MustNew("usr_")

	repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentCommentByIDRequest) bool {
			return req.CommentID == commentID
		})).
		Return(&shipment.ShipmentComment{ID: commentID, ShipmentID: shipmentID, UserID: testutil.TestUserID, OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID, Comment: "before", Version: 3, CreatedAt: 100}, nil).
		Once()

	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.Shipment{ID: shipmentID}, nil).Once()
	userRepo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return([]*tenant.User{{ID: mentionID, Name: "Alice"}}, nil).Once()

	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.Version == 3 && comment.EditedAt != nil && len(comment.MentionedUsers) == 1
		})).
		RunAndReturn(func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
			comment.Version = 4
			return comment, nil
		}).
		Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything).
		Run(func(params *servicesport.LogActionParams, _ ...servicesport.LogOption) {
			assert.Equal(t, permission.ResourceShipmentComment, params.Resource)
			assert.Equal(t, commentID.String(), params.ResourceID)
		}).
		Return(nil).Once()

	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *servicesport.PublishResourceInvalidationRequest) bool {
		return req.Resource == permission.ResourceShipmentComment.String() && req.Action == "updated" && req.RecordID == shipmentID
	})).Return(nil).Once()

	updated, err := svc.Update(t.Context(), &shipment.ShipmentComment{
		ID:               commentID,
		ShipmentID:       shipmentID,
		OrganizationID:   testutil.TestOrgID,
		BusinessUnitID:   testutil.TestBuID,
		Comment:          "after",
		MentionedUserIDs: []pulid.ID{mentionID},
	}, testActor())

	require.NoError(t, err)
	require.NotNil(t, updated.EditedAt)
	assert.Equal(t, int64(4), updated.Version)
}

func TestDelete_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentCommentRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		AuditService: audit,
		Realtime:     realtime,
	})

	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{ID: pulid.MustNew("shc_"), ShipmentID: pulid.MustNew("shp_"), UserID: pulid.MustNew("usr_"), OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID}, nil).Once()

	err := svc.Delete(t.Context(), &repositories.DeleteShipmentCommentRequest{
		ShipmentID: pulid.MustNew("shp_"),
		CommentID:  pulid.MustNew("shc_"),
		TenantInfo: pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID},
	}, testActor())

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func testActor() *servicesport.RequestActor {
	return &servicesport.RequestActor{
		PrincipalType:  servicesport.PrincipalTypeUser,
		PrincipalID:    testutil.TestUserID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}
}
