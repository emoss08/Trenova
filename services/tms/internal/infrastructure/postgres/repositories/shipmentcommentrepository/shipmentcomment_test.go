package shipmentcommentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)

	bunDB := bun.NewDB(db, pgdialect.New())

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, mock
}

func TestListByShipmentID_ReturnsCommentsAndCount(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")
	userID := pulid.MustNew("usr_")

	mock.ExpectQuery(`SELECT count\(\*\) FROM "shipment_comments" AS "sc".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .* FROM "shipment_comments" AS "sc".*ORDER BY "sc"\."created_at" DESC.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "user_id", "comment", "version", "created_at", "updated_at",
			"user__id", "user__business_unit_id", "user__current_organization_id", "user__status", "user__name", "user__username", "user__time_format", "user__password", "user__email_address", "user__profile_pic_url", "user__thumbnail_url", "user__timezone", "user__is_locked", "user__must_change_password", "user__is_platform_admin", "user__version", "user__created_at", "user__updated_at", "user__last_login_at",
		}).AddRow(commentID, buID, orgID, shipmentID, userID, "hello", 0, 1, 1, userID, buID, orgID, "Active", "Alice", "alice", "12-hour", "secret", "a@example.com", "", "", "UTC", false, false, false, 0, 1, 1, nil))
	mock.ExpectQuery(`SELECT .* FROM "shipment_comment_mentions" AS "scm".*LEFT JOIN "users" AS "mentioned_user".*WHERE .*"scm"\."comment_id" IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "comment_id", "mentioned_user_id", "organization_id", "business_unit_id", "shipment_id", "created_at",
			"mentioned_user__id", "mentioned_user__business_unit_id", "mentioned_user__current_organization_id", "mentioned_user__status", "mentioned_user__name", "mentioned_user__username", "mentioned_user__time_format", "mentioned_user__password", "mentioned_user__email_address", "mentioned_user__profile_pic_url", "mentioned_user__thumbnail_url", "mentioned_user__timezone", "mentioned_user__is_locked", "mentioned_user__must_change_password", "mentioned_user__is_platform_admin", "mentioned_user__version", "mentioned_user__created_at", "mentioned_user__updated_at", "mentioned_user__last_login_at",
		}))

	result, err := repo.ListByShipmentID(t.Context(), &repositories.ListShipmentCommentsRequest{
		ShipmentID: shipmentID,
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
			Pagination: pagination.Info{Limit: 20},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, commentID, result.Items[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_InsertsCommentAndMentions(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	entity := &shipment.ShipmentComment{
		ID:             pulid.MustNew("shc_"),
		ShipmentID:     pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		Comment:        "hello",
		MentionedUsers: []*shipment.ShipmentCommentMention{
			{MentionedUserID: pulid.MustNew("usr_")},
		},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "shipment_comments".*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(1))
	mock.ExpectExec(`INSERT INTO "shipment_comment_mentions".*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT .* FROM "shipment_comments" AS "sc".*sc\.id = .*sc\.shipment_id = .*sc\.organization_id = .*sc\.business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "user_id", "comment", "version", "created_at", "updated_at",
			"user__id", "user__business_unit_id", "user__current_organization_id", "user__status", "user__name", "user__username", "user__time_format", "user__password", "user__email_address", "user__profile_pic_url", "user__thumbnail_url", "user__timezone", "user__is_locked", "user__must_change_password", "user__is_platform_admin", "user__version", "user__created_at", "user__updated_at", "user__last_login_at",
		}).AddRow(entity.ID, entity.BusinessUnitID, entity.OrganizationID, entity.ShipmentID, entity.UserID, entity.Comment, 0, 1, 1, entity.UserID, entity.BusinessUnitID, entity.OrganizationID, "Active", "Alice", "alice", "12-hour", "secret", "a@example.com", "", "", "UTC", false, false, false, 0, 1, 1, nil))
	mock.ExpectQuery(`SELECT .* FROM "shipment_comment_mentions" AS "scm".*LEFT JOIN "users" AS "mentioned_user".*WHERE .*"scm"\."comment_id" IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "comment_id", "mentioned_user_id", "organization_id", "business_unit_id", "shipment_id", "created_at",
			"mentioned_user__id", "mentioned_user__business_unit_id", "mentioned_user__current_organization_id", "mentioned_user__status", "mentioned_user__name", "mentioned_user__username", "mentioned_user__time_format", "mentioned_user__password", "mentioned_user__email_address", "mentioned_user__profile_pic_url", "mentioned_user__thumbnail_url", "mentioned_user__timezone", "mentioned_user__is_locked", "mentioned_user__must_change_password", "mentioned_user__is_platform_admin", "mentioned_user__version", "mentioned_user__created_at", "mentioned_user__updated_at", "mentioned_user__last_login_at",
		}).AddRow(entity.MentionedUsers[0].ID, entity.ID, entity.MentionedUsers[0].MentionedUserID, entity.OrganizationID, entity.BusinessUnitID, entity.ShipmentID, 1, entity.MentionedUsers[0].MentionedUserID, entity.BusinessUnitID, entity.OrganizationID, "Active", "Bob", "bob", "12-hour", "secret", "b@example.com", "", "", "UTC", false, false, false, 0, 1, 1, nil))

	created, err := repo.Create(t.Context(), entity)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, created.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_RemovesComment(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	commentID := pulid.MustNew("shc_")
	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mock.ExpectExec(`DELETE FROM "shipment_comments" AS "sc".*organization_id = .*business_unit_id = .*id = .*shipment_id = .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(t.Context(), &repositories.DeleteShipmentCommentRequest{
		CommentID:  commentID,
		ShipmentID: shipmentID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
