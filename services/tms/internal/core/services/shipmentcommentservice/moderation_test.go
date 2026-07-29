package shipmentcommentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
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

type commentServiceMocks struct {
	repo         *mocks.MockShipmentCommentRepository
	shipmentRepo *mocks.MockShipmentRepository
	userRepo     *mocks.MockUserRepository
	docRepo      *mocks.MockDocumentRepository
	audit        *mocks.MockAuditService
	realtime     *mocks.MockRealtimeService
}

func newCommentService(t *testing.T) (servicesport.ShipmentCommentService, commentServiceMocks) {
	t.Helper()

	m := commentServiceMocks{
		repo:         mocks.NewMockShipmentCommentRepository(t),
		shipmentRepo: mocks.NewMockShipmentRepository(t),
		userRepo:     mocks.NewMockUserRepository(t),
		docRepo:      mocks.NewMockDocumentRepository(t),
		audit:        mocks.NewMockAuditService(t),
		realtime:     mocks.NewMockRealtimeService(t),
	}
	m.docRepo.EXPECT().
		GetByResourceIDs(mock.Anything, mock.Anything).
		Return([]*document.Document{}, nil).
		Maybe()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         m.repo,
		ShipmentRepo: m.shipmentRepo,
		UserRepo:     m.userRepo,
		DocumentRepo: m.docRepo,
		AuditService: m.audit,
		EventService: noopShipmentEventService{},
		Realtime:     m.realtime,
	})

	return svc, m
}

func tenantScope() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}
}

func TestCreate_RejectsReplyToReply(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	parentID := pulid.MustNew("shc_")
	grandparentID := pulid.MustNew("shc_")

	m.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentCommentByIDRequest) bool {
			return req.CommentID == parentID
		})).
		Return(&shipment.ShipmentComment{
			ID:              parentID,
			ShipmentID:      shipmentID,
			ParentCommentID: &grandparentID,
			OrganizationID:  testutil.TestOrgID,
			BusinessUnitID:  testutil.TestBuID,
		}, nil).
		Once()

	created, err := svc.Create(t.Context(), &shipment.ShipmentComment{
		ShipmentID:      shipmentID,
		OrganizationID:  testutil.TestOrgID,
		BusinessUnitID:  testutil.TestBuID,
		ParentCommentID: &parentID,
		Comment:         "nested reply",
	}, testActor())

	require.Error(t, err)
	assert.Nil(t, created)
}

func TestCreate_ReplyForcesRequiresAcknowledgmentOff(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	parentID := pulid.MustNew("shc_")
	parentAuthor := pulid.MustNew("usr_")

	m.repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.ShipmentComment{
			ID:             parentID,
			ShipmentID:     shipmentID,
			UserID:         parentAuthor,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
		}, nil).
		Once()
	m.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID, ProNumber: "S-1"}, nil).
		Once()
	m.repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.ParentCommentID != nil &&
				*comment.ParentCommentID == parentID &&
				!comment.RequiresAcknowledgment
		})).
		RunAndReturn(func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
			comment.ID = pulid.MustNew("shc_")
			return comment, nil
		}).
		Once()
	m.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	m.realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.Anything).
		Return(nil).
		Once()

	created, err := svc.Create(t.Context(), &shipment.ShipmentComment{
		ShipmentID:             shipmentID,
		OrganizationID:         testutil.TestOrgID,
		BusinessUnitID:         testutil.TestBuID,
		ParentCommentID:        &parentID,
		Comment:                "a reply",
		Priority:               shipment.CommentPriorityUrgent,
		RequiresAcknowledgment: true,
	}, testActor())

	require.NoError(t, err)
	assert.False(t, created.RequiresAcknowledgment)
}

func TestCreate_WithBodyDerivesPlainTextAndMentions(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	mentionID := pulid.MustNew("usr_")

	body := map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": "Ping "},
					map[string]any{
						"type": "mention",
						"attrs": map[string]any{
							"id":    mentionID.String(),
							"label": "Jane",
						},
					},
				},
			},
		},
	}

	m.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID, ProNumber: "S-1"}, nil).
		Once()
	m.userRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.GetUsersByIDsRequest) bool {
			return len(req.UserIDs) == 1 && req.UserIDs[0] == mentionID
		})).
		Return([]*tenant.User{{ID: mentionID, Name: "Jane"}}, nil).
		Once()
	m.userRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: testutil.TestUserID, Name: "Author"}, nil).
		Maybe()
	m.repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.Comment == "Ping @Jane" && comment.Body != nil
		})).
		RunAndReturn(func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
			comment.ID = pulid.MustNew("shc_")
			return comment, nil
		}).
		Once()
	m.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	m.realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.Anything).
		Return(nil).
		Once()

	created, err := svc.Create(t.Context(), &shipment.ShipmentComment{
		ShipmentID:     shipmentID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Body:           body,
	}, testActor())

	require.NoError(t, err)
	assert.Equal(t, "Ping @Jane", created.Comment)
	assert.Len(t, created.MentionedUserIDs, 1)
}

func TestCreate_RejectsInvalidBody(t *testing.T) {
	t.Parallel()

	svc, _ := newCommentService(t)

	created, err := svc.Create(t.Context(), &shipment.ShipmentComment{
		ShipmentID:     pulid.MustNew("shp_"),
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Body: map[string]any{
			"type": "doc",
			"content": []any{
				map[string]any{"type": "script"},
			},
		},
	}, testActor())

	require.Error(t, err)
	assert.Nil(t, created)
}

func TestCreate_RejectsForeignAttachmentParentage(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	otherShipmentID := pulid.MustNew("shp_")
	docID := pulid.MustNew("doc_")

	m.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()
	m.docRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.BulkDeleteDocumentRequest) bool {
			return len(req.IDs) == 1 && req.IDs[0] == docID
		})).
		Return([]*document.Document{{
			ID:           docID,
			ResourceType: "shipment",
			ResourceID:   otherShipmentID.String(),
		}}, nil).
		Once()

	created, err := svc.Create(t.Context(), &shipment.ShipmentComment{
		ShipmentID:            shipmentID,
		OrganizationID:        testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Comment:               "with attachment",
		AttachmentDocumentIDs: []pulid.ID{docID},
	}, testActor())

	require.Error(t, err)
	assert.Nil(t, created)
}

func TestDelete_TombstonesWhenCommentHasReplies(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")

	original := &shipment.ShipmentComment{
		ID:             commentID,
		ShipmentID:     shipmentID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Comment:        "to delete",
	}

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(original, nil).Once()
	m.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()
	m.repo.EXPECT().CountReplies(mock.Anything, mock.Anything).Return(2, nil).Once()

	now := int64(1000)
	m.repo.EXPECT().
		SoftDelete(mock.Anything, mock.MatchedBy(func(req *repositories.SoftDeleteShipmentCommentRequest) bool {
			return req.CommentID == commentID && req.ActorID == testutil.TestUserID
		})).
		Return(&shipment.ShipmentComment{
			ID:             commentID,
			ShipmentID:     shipmentID,
			UserID:         testutil.TestUserID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			DeletedAt:      &now,
		}, nil).
		Once()
	m.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	m.realtime.EXPECT().
		PublishResourceInvalidation(
			mock.Anything,
			mock.MatchedBy(func(req *servicesport.PublishResourceInvalidationRequest) bool {
				return req.Action == "updated"
			}),
		).
		Return(nil).
		Once()

	err := svc.Delete(t.Context(), &servicesport.DeleteShipmentCommentRequest{
		ShipmentID: shipmentID,
		CommentID:  commentID,
		TenantInfo: tenantScope(),
	}, testActor())

	require.NoError(t, err)
}

func TestDelete_HardDeletesWithoutReplies(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:             commentID,
		ShipmentID:     shipmentID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}, nil).Once()
	m.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()
	m.repo.EXPECT().CountReplies(mock.Anything, mock.Anything).Return(0, nil).Once()
	m.repo.EXPECT().
		Delete(mock.Anything, mock.MatchedBy(func(req *repositories.DeleteShipmentCommentRequest) bool {
			return req.CommentID == commentID
		})).
		Return(nil).
		Once()
	m.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	m.realtime.EXPECT().
		PublishResourceInvalidation(
			mock.Anything,
			mock.MatchedBy(func(req *servicesport.PublishResourceInvalidationRequest) bool {
				return req.Action == "deleted"
			}),
		).
		Return(nil).
		Once()

	err := svc.Delete(t.Context(), &servicesport.DeleteShipmentCommentRequest{
		ShipmentID: shipmentID,
		CommentID:  commentID,
		TenantInfo: tenantScope(),
	}, testActor())

	require.NoError(t, err)
}

func TestDelete_ModeratorCanDeleteOthersComment(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")
	otherUser := pulid.MustNew("usr_")

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:             commentID,
		ShipmentID:     shipmentID,
		UserID:         otherUser,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}, nil).Once()
	m.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()
	m.repo.EXPECT().CountReplies(mock.Anything, mock.Anything).Return(0, nil).Once()
	m.repo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil).Once()
	m.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	m.realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.Anything).
		Return(nil).
		Once()

	err := svc.Delete(t.Context(), &servicesport.DeleteShipmentCommentRequest{
		ShipmentID:  shipmentID,
		CommentID:   commentID,
		TenantInfo:  tenantScope(),
		AsModerator: true,
	}, testActor())

	require.NoError(t, err)
}

func TestUpdate_ModeratorPreservesOriginalAuthor(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")
	originalAuthor := pulid.MustNew("usr_")

	m.repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.ShipmentComment{
			ID:             commentID,
			ShipmentID:     shipmentID,
			UserID:         originalAuthor,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Comment:        "before",
			Version:        1,
		}, nil).
		Once()
	m.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()
	m.repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.UserID == originalAuthor && comment.Version == 1
		})).
		RunAndReturn(func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
			comment.Version = 2
			return comment, nil
		}).
		Once()
	m.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	m.realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.Anything).
		Return(nil).
		Once()

	updated, err := svc.Update(t.Context(), &servicesport.UpdateShipmentCommentRequest{
		Entity: &shipment.ShipmentComment{
			ID:             commentID,
			ShipmentID:     shipmentID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Comment:        "after",
			Version:        1,
		},
		AsModerator: true,
	}, testActor())

	require.NoError(t, err)
	assert.Equal(t, originalAuthor, updated.UserID)
}

func TestUpdate_NonOwnerWithoutModeratorIsRejected(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:             pulid.MustNew("shc_"),
		ShipmentID:     pulid.MustNew("shp_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}, nil).Once()

	updated, err := svc.Update(t.Context(), &servicesport.UpdateShipmentCommentRequest{
		Entity: &shipment.ShipmentComment{
			ID:             pulid.MustNew("shc_"),
			ShipmentID:     pulid.MustNew("shp_"),
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Comment:        "after",
		},
	}, testActor())

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestPin_EnforcesPinnedCap(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:             commentID,
		ShipmentID:     shipmentID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}, nil).Once()
	m.repo.EXPECT().
		CountPinnedByShipmentID(mock.Anything, mock.Anything).
		Return(shipment.MaxPinnedComments, nil).
		Once()

	pinned, err := svc.Pin(t.Context(), &servicesport.ToggleShipmentCommentRequest{
		TenantInfo: tenantScope(),
		ShipmentID: shipmentID,
		CommentID:  commentID,
	}, testActor())

	require.Error(t, err)
	assert.Nil(t, pinned)
}

func TestPin_RejectsReplies(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")
	parentID := pulid.MustNew("shc_")

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:              commentID,
		ShipmentID:      shipmentID,
		ParentCommentID: &parentID,
		UserID:          testutil.TestUserID,
		OrganizationID:  testutil.TestOrgID,
		BusinessUnitID:  testutil.TestBuID,
	}, nil).Once()

	pinned, err := svc.Pin(t.Context(), &servicesport.ToggleShipmentCommentRequest{
		TenantInfo: tenantScope(),
		ShipmentID: shipmentID,
		CommentID:  commentID,
	}, testActor())

	require.Error(t, err)
	assert.Nil(t, pinned)
}

func TestPin_PublishesPinnedAction(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")
	now := int64(2000)

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:             commentID,
		ShipmentID:     shipmentID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}, nil).Once()
	m.repo.EXPECT().CountPinnedByShipmentID(mock.Anything, mock.Anything).Return(0, nil).Once()
	m.repo.EXPECT().
		SetPinned(mock.Anything, mock.MatchedBy(func(req *repositories.SetShipmentCommentPinnedRequest) bool {
			return req.Pinned && req.ActorID == testutil.TestUserID
		})).
		Return(&shipment.ShipmentComment{
			ID:             commentID,
			ShipmentID:     shipmentID,
			UserID:         testutil.TestUserID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			PinnedAt:       &now,
		}, nil).
		Once()
	m.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	m.realtime.EXPECT().
		PublishResourceInvalidation(
			mock.Anything,
			mock.MatchedBy(func(req *servicesport.PublishResourceInvalidationRequest) bool {
				return req.Action == "pinned"
			}),
		).
		Return(nil).
		Once()

	pinned, err := svc.Pin(t.Context(), &servicesport.ToggleShipmentCommentRequest{
		TenantInfo: tenantScope(),
		ShipmentID: shipmentID,
		CommentID:  commentID,
	}, testActor())

	require.NoError(t, err)
	require.NotNil(t, pinned.PinnedAt)
}

func TestAcknowledge_RequiresAcknowledgmentFlag(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:             pulid.MustNew("shc_"),
		ShipmentID:     pulid.MustNew("shp_"),
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}, nil).Once()

	acked, err := svc.Acknowledge(t.Context(), &servicesport.ToggleShipmentCommentRequest{
		TenantInfo: tenantScope(),
		ShipmentID: pulid.MustNew("shp_"),
		CommentID:  pulid.MustNew("shc_"),
	}, testActor())

	require.Error(t, err)
	assert.Nil(t, acked)
}

func TestAcknowledge_IdempotentSkipsSideEffects(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")

	target := &shipment.ShipmentComment{
		ID:                     commentID,
		ShipmentID:             shipmentID,
		UserID:                 testutil.TestUserID,
		OrganizationID:         testutil.TestOrgID,
		BusinessUnitID:         testutil.TestBuID,
		RequiresAcknowledgment: true,
	}

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(target, nil).Twice()
	m.repo.EXPECT().
		Acknowledge(mock.Anything, mock.MatchedBy(func(req *repositories.AcknowledgeShipmentCommentRequest) bool {
			return req.UserID == testutil.TestUserID
		})).
		Return(false, nil).
		Once()

	acked, err := svc.Acknowledge(t.Context(), &servicesport.ToggleShipmentCommentRequest{
		TenantInfo: tenantScope(),
		ShipmentID: shipmentID,
		CommentID:  commentID,
	}, testActor())

	require.NoError(t, err)
	require.NotNil(t, acked)
}

func TestResolve_PublishesResolvedAction(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")
	now := int64(3000)

	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:             commentID,
		ShipmentID:     shipmentID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}, nil).Once()
	m.repo.EXPECT().
		SetResolved(mock.Anything, mock.MatchedBy(func(req *repositories.SetShipmentCommentResolvedRequest) bool {
			return req.Resolved && req.ActorID == testutil.TestUserID
		})).
		Return(&shipment.ShipmentComment{
			ID:             commentID,
			ShipmentID:     shipmentID,
			UserID:         testutil.TestUserID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			ResolvedAt:     &now,
		}, nil).
		Once()
	m.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	m.realtime.EXPECT().
		PublishResourceInvalidation(
			mock.Anything,
			mock.MatchedBy(func(req *servicesport.PublishResourceInvalidationRequest) bool {
				return req.Action == "resolved"
			}),
		).
		Return(nil).
		Once()

	resolved, err := svc.Resolve(t.Context(), &servicesport.ToggleShipmentCommentRequest{
		TenantInfo: tenantScope(),
		ShipmentID: shipmentID,
		CommentID:  commentID,
	}, testActor())

	require.NoError(t, err)
	require.NotNil(t, resolved.ResolvedAt)
}

func TestDelete_RejectsAlreadyDeletedComment(t *testing.T) {
	t.Parallel()

	svc, m := newCommentService(t)

	now := int64(4000)
	m.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.ShipmentComment{
		ID:             pulid.MustNew("shc_"),
		ShipmentID:     pulid.MustNew("shp_"),
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		DeletedAt:      &now,
	}, nil).Once()

	err := svc.Delete(t.Context(), &servicesport.DeleteShipmentCommentRequest{
		ShipmentID: pulid.MustNew("shp_"),
		CommentID:  pulid.MustNew("shc_"),
		TenantInfo: tenantScope(),
	}, testActor())

	require.Error(t, err)
}
