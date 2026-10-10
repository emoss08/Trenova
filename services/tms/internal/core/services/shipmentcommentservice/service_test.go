package shipmentcommentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/agentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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

	repo := shipmentportsmock.NewMockShipmentCommentRepository(t)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := agentportsmock.NewMockRealtimeService(t)
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	docRepo.EXPECT().
		GetByResourceIDs(mock.Anything, mock.Anything).
		Return([]*document.Document{}, nil).
		Maybe()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		DocumentRepo: docRepo,
		AuditService: audit,
		EventService: noopShipmentEventService{},
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
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		})).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()

	userRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req tenantports.GetUsersByIDsRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID &&
				len(req.UserIDs) == 2
		})).
		Return([]*tenant.User{{ID: mentionA, Name: "Alice"}, {ID: mentionB, Name: "Bob"}}, nil).
		Once()

	repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.UserID == testutil.TestUserID &&
				comment.Comment == "hello world" &&
				len(comment.MentionedUsers) == 2
		})).
		RunAndReturn(func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
			comment.ID = pulid.MustNew("shc_")
			return comment, nil
		}).
		Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything).
		Run(func(params *auditports.LogActionParams, _ ...auditports.LogOption) {
			assert.Equal(t, permission.ResourceShipmentComment, params.Resource)
			assert.Equal(t, testutil.TestOrgID, params.OrganizationID)
		}).
		Return(nil).Once()

	realtime.EXPECT().
		PublishResourceInvalidation(
			mock.Anything,
			mock.MatchedBy(func(req *agentports.PublishResourceInvalidationRequest) bool {
				return req.Resource == permission.ResourceShipmentComment.String() &&
					req.Action == "created" &&
					req.RecordID == shipmentID &&
					req.OrganizationID == testutil.TestOrgID &&
					req.BusinessUnitID == testutil.TestBuID &&
					req.ActorUserID == testutil.TestUserID
			}),
		).
		Return(nil).Once()

	created, err := svc.Create(t.Context(), entity, testActor())

	require.NoError(t, err)
	assert.Equal(t, "hello world", created.Comment)
	assert.Len(t, created.MentionedUserIDs, 2)
	assert.Equal(t, testutil.TestUserID, created.UserID)
}

func TestCreate_RejectsUnknownMentionedUsers(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentCommentRepository(t)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := agentportsmock.NewMockRealtimeService(t)
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	docRepo.EXPECT().
		GetByResourceIDs(mock.Anything, mock.Anything).
		Return([]*document.Document{}, nil).
		Maybe()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		DocumentRepo: docRepo,
		AuditService: audit,
		EventService: noopShipmentEventService{},
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

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()
	userRepo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return([]*tenant.User{}, nil).Once()

	created, err := svc.Create(t.Context(), entity, testActor())

	require.Error(t, err)
	assert.Nil(t, created)
	assert.True(t, errortypes.IsError(err))
}

func TestCreate_RequiresUserActor(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         shipmentportsmock.NewMockShipmentCommentRepository(t),
		ShipmentRepo: shipmentportsmock.NewMockShipmentRepository(t),
		UserRepo:     tenantportsmock.NewMockUserRepository(t),
		DocumentRepo: documentportsmock.NewMockDocumentRepository(t),
		AuditService: auditportsmock.NewMockAuditService(t),
		EventService: noopShipmentEventService{},
		Realtime:     agentportsmock.NewMockRealtimeService(t),
	})

	created, err := svc.Create(t.Context(), &shipment.ShipmentComment{
		ShipmentID:     pulid.MustNew("shp_"),
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Comment:        "hello",
	}, nil)

	require.Error(t, err)
	assert.Nil(t, created)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestCreateSystem_CreatesTenantScopedCommentWithoutUser(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentCommentRepository(t)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := agentportsmock.NewMockRealtimeService(t)
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	docRepo.EXPECT().
		GetByResourceIDs(mock.Anything, mock.Anything).
		Return([]*document.Document{}, nil).
		Maybe()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		DocumentRepo: docRepo,
		AuditService: audit,
		EventService: noopShipmentEventService{},
		Realtime:     realtime,
	})

	shipmentID := pulid.MustNew("shp_")
	metadata := map[string]any{
		"source": "auto-delay",
		"code":   "late-stop",
	}

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID &&
				req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		})).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()

	repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.UserID.IsNil() &&
				comment.Source == shipment.CommentSourceSystem &&
				comment.OrganizationID == testutil.TestOrgID &&
				comment.BusinessUnitID == testutil.TestBuID &&
				comment.ShipmentID == shipmentID &&
				comment.Comment == "internal note" &&
				comment.Metadata["source"] == "auto-delay"
		})).
		RunAndReturn(func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
			comment.ID = pulid.MustNew("shc_")
			return comment, nil
		}).
		Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything).
		Run(func(params *auditports.LogActionParams, _ ...auditports.LogOption) {
			assert.Equal(t, permission.ResourceShipmentComment, params.Resource)
			assert.Equal(t, permissionports.PrincipalTypeSystem, params.PrincipalType)
			assert.Equal(t, permissionports.SystemPrincipalID, params.PrincipalID)
			assert.True(t, params.UserID.IsNil())
			assert.Equal(t, testutil.TestOrgID, params.OrganizationID)
		}).
		Return(nil).
		Once()

	realtime.EXPECT().
		PublishResourceInvalidation(
			mock.Anything,
			mock.MatchedBy(func(req *agentports.PublishResourceInvalidationRequest) bool {
				return req.Resource == permission.ResourceShipmentComment.String() &&
					req.Action == "created" &&
					req.RecordID == shipmentID &&
					req.ActorType == permissionports.PrincipalTypeSystem &&
					req.ActorID == permissionports.SystemPrincipalID &&
					req.ActorUserID.IsNil()
			}),
		).
		Return(nil).
		Once()

	created, err := svc.CreateSystem(t.Context(), &shipmentports.CreateSystemShipmentCommentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: testutil.TestOrgID,
			BuID:  testutil.TestBuID,
		},
		ShipmentID: shipmentID,
		Comment:    " internal note ",
		Metadata:   metadata,
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.True(t, created.UserID.IsNil())
	assert.Equal(t, shipment.CommentSourceSystem, created.Source)
	assert.Equal(t, "auto-delay", created.Metadata["source"])
}

func TestUpdate_RequiresOwnerSetsEditedAtAndPublishes(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentCommentRepository(t)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := agentportsmock.NewMockRealtimeService(t)
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	docRepo.EXPECT().
		GetByResourceIDs(mock.Anything, mock.Anything).
		Return([]*document.Document{}, nil).
		Maybe()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		DocumentRepo: docRepo,
		AuditService: audit,
		EventService: noopShipmentEventService{},
		Realtime:     realtime,
	})

	commentID := pulid.MustNew("shc_")
	shipmentID := pulid.MustNew("shp_")
	mentionID := pulid.MustNew("usr_")

	repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentCommentByIDRequest) bool {
			return req.CommentID == commentID
		})).
		Return(&shipment.ShipmentComment{
			ID:             commentID,
			ShipmentID:     shipmentID,
			UserID:         testutil.TestUserID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Comment:        "before",
			Version:        3,
			CreatedAt:      100,
		}, nil).
		Once()

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()
	userRepo.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return([]*tenant.User{{ID: mentionID, Name: "Alice"}}, nil).
		Once()

	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.Version == 3 && comment.EditedAt != nil &&
				len(comment.MentionedUsers) == 1
		})).
		RunAndReturn(func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
			comment.Version = 4
			return comment, nil
		}).
		Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything).
		Run(func(params *auditports.LogActionParams, _ ...auditports.LogOption) {
			assert.Equal(t, permission.ResourceShipmentComment, params.Resource)
			assert.Equal(t, commentID.String(), params.ResourceID)
		}).
		Return(nil).Once()

	realtime.EXPECT().
		PublishResourceInvalidation(
			mock.Anything,
			mock.MatchedBy(func(req *agentports.PublishResourceInvalidationRequest) bool {
				return req.Resource == permission.ResourceShipmentComment.String() &&
					req.Action == "updated" &&
					req.RecordID == shipmentID
			}),
		).
		Return(nil).
		Once()

	updated, err := svc.Update(t.Context(), &shipmentports.UpdateShipmentCommentRequest{
		Entity: &shipment.ShipmentComment{
			ID:               commentID,
			ShipmentID:       shipmentID,
			OrganizationID:   testutil.TestOrgID,
			BusinessUnitID:   testutil.TestBuID,
			Comment:          "after",
			Version:          3,
			MentionedUserIDs: []pulid.ID{mentionID},
		},
	}, testActor())

	require.NoError(t, err)
	require.NotNil(t, updated.EditedAt)
	assert.Equal(t, int64(4), updated.Version)
}

func TestDelete_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentCommentRepository(t)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := agentportsmock.NewMockRealtimeService(t)
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	docRepo.EXPECT().
		GetByResourceIDs(mock.Anything, mock.Anything).
		Return([]*document.Document{}, nil).
		Maybe()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ShipmentRepo: shipmentRepo,
		UserRepo:     userRepo,
		DocumentRepo: docRepo,
		AuditService: audit,
		EventService: noopShipmentEventService{},
		Realtime:     realtime,
	})

	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.ShipmentComment{
			ID:             pulid.MustNew("shc_"),
			ShipmentID:     pulid.MustNew("shp_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
		}, nil).
		Once()

	err := svc.Delete(t.Context(), &shipmentports.DeleteShipmentCommentRequest{
		ShipmentID: pulid.MustNew("shp_"),
		CommentID:  pulid.MustNew("shc_"),
		TenantInfo: pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID},
	}, testActor())

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func testActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    testutil.TestUserID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}
}
