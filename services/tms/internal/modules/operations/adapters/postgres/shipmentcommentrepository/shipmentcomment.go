package shipmentcommentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const commentEntity = "Shipment comment"

var commentErrors = dberror.Mapping{Entity: commentEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) shipmentports.ShipmentCommentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-comment-repository"),
	}
}

func (r *repository) ListByShipmentID(
	ctx context.Context,
	req *shipmentports.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
			replyMode := req.ParentCommentID != nil && !req.ParentCommentID.IsNil()

			direction := dbtype.SortDirectionDesc
			if replyMode {
				direction = dbtype.SortDirectionAsc
			}
			filter := &pagination.QueryOptions{
				Sort: []domaintypes.SortField{
					{Field: "createdAt", Direction: direction},
					{Field: "id", Direction: direction},
				},
			}
			if req.Filter != nil {
				filter.TenantInfo = req.Filter.TenantInfo
				filter.Pagination = req.Filter.Pagination
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				shipmentdb.ShipmentComments.Where(listConditions(req)...),
				&listquery.Request[shipment.ShipmentComment]{
					Filter: filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*shipment.ShipmentComment],
					) *keel.Query[*shipment.ShipmentComment] {
						q = withCommentRelations(q)
						if replyMode {
							return q
						}
						return withReplyCount(q)
					},
				},
			)
			if err != nil {
				return nil, commentErrors.Map(err, req.ShipmentID.String())
			}

			return result, nil
		},
	)
}

func listConditions(req *shipmentports.ListShipmentCommentsRequest) []keel.Pred {
	sc := shipmentdb.ShipmentComments
	f := req.Filters
	replyMode := req.ParentCommentID != nil && !req.ParentCommentID.IsNil()

	preds := []keel.Pred{sc.ShipmentID.Eq(req.ShipmentID)}
	if replyMode {
		preds = append(preds, sc.ParentCommentID.Eq(*req.ParentCommentID))
	} else {
		preds = append(preds, sc.ParentCommentID.IsNull())
		if f.HasAny() {
			preds = append(preds, sc.DeletedAt.IsNull())
		}
	}

	if len(f.Types) > 0 {
		preds = append(preds, sc.Type.In(f.Types...))
	}
	if len(f.Priorities) > 0 {
		preds = append(preds, sc.Priority.In(f.Priorities...))
	}
	if len(f.AuthorIDs) > 0 {
		preds = append(preds, sc.UserID.In(f.AuthorIDs...))
	}
	if f.MentionsUserID != nil && !f.MentionsUserID.IsNil() {
		m := sc.MentionedUsers()
		preds = append(preds, m.Any(m.MentionedUserID.Eq(*f.MentionsUserID)))
	}
	if f.PinnedOnly {
		preds = append(preds, sc.PinnedAt.IsNotNull())
	}
	if f.UnresolvedOnly {
		preds = append(preds, sc.ResolvedAt.IsNull())
	}
	if f.Search != "" {
		preds = append(preds, sc.SearchVector.Match(f.Search, keel.SearchConfig("english")))
	}

	return preds
}

func withReplyCount(q *keel.Query[*shipment.ShipmentComment]) *keel.Query[*shipment.ShipmentComment] {
	sc := shipmentdb.ShipmentComments
	replies := sc.As("scr")

	return q.Compute(sc.ReplyCount, keel.Scalar(replies.Pick(keel.Count()).Where(
		replies.ParentCommentID.EqCol(sc.ID),
		replies.ShipmentID.EqCol(sc.ShipmentID),
	)))
}

func withCommentRelations(
	q *keel.Query[*shipment.ShipmentComment],
) *keel.Query[*shipment.ShipmentComment] {
	sc := shipmentdb.ShipmentComments
	mentions := sc.MentionedUsers()
	acks := sc.Acknowledgments()

	return q.Load(
		sc.User(),
		sc.PinnedBy(),
		sc.ResolvedBy(),
		mentions.Order(mentions.CreatedAt.Asc()).Load(mentions.MentionedUser()),
		acks.Order(acks.AcknowledgedAt.Asc()).Load(acks.User()),
	)
}

func (r *repository) GetCountByShipmentID(
	ctx context.Context,
	req *shipmentports.GetShipmentCommentCountRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		sc := shipmentdb.ShipmentComments
		n, err := sc.Where(sc.ShipmentID.Eq(req.ShipmentID), sc.DeletedAt.IsNull()).
			Count(ctx, r.db.Keel(ctx))
		return int(n), err
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *shipmentports.GetShipmentCommentByIDRequest,
) (*shipment.ShipmentComment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentComment, error) {
		sc := shipmentdb.ShipmentComments
		q := sc.Where(sc.ID.Eq(req.CommentID), sc.ShipmentID.Eq(req.ShipmentID))
		entity, err := withReplyCount(withCommentRelations(q)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, commentErrors.Map(err, req.CommentID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentComment, error) {
		if err := shipmentdb.ShipmentComments.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, fmt.Errorf(
				"insert shipment comment: %w",
				commentErrors.Map(err, entity.ID.String()),
			)
		}

		if err := r.insertMentions(ctx, entity); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &shipmentports.GetShipmentCommentByIDRequest{
			CommentID:  entity.ID,
			ShipmentID: entity.ShipmentID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentComment, error) {
		sc := shipmentdb.ShipmentComments
		n, err := sc.Where(
			sc.ID.Eq(entity.ID),
			sc.ShipmentID.Eq(entity.ShipmentID),
			sc.Version.Eq(entity.Version),
			sc.DeletedAt.IsNull(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			sc.Type.To(entity.Type),
			sc.Visibility.To(entity.Visibility),
			sc.Priority.To(entity.Priority),
			sc.Source.To(entity.Source),
			sc.Metadata.To(entity.Metadata),
			sc.Comment.To(entity.Comment),
			sc.Body.To(entity.Body),
			sc.RequiresAcknowledgment.To(entity.RequiresAcknowledgment),
			sc.EditedAt.ToExpr(keel.Val(entity.EditedAt)),
		)
		if err != nil {
			return nil, fmt.Errorf("update shipment comment: %w", err)
		}
		if err = dberror.CheckAffected(n, commentEntity, entity.ID.String()); err != nil {
			return nil, err
		}

		if err = r.deleteMentions(ctx, entity.ID); err != nil {
			return nil, err
		}
		if err = r.insertMentions(ctx, entity); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &shipmentports.GetShipmentCommentByIDRequest{
			CommentID:  entity.ID,
			ShipmentID: entity.ShipmentID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req *shipmentports.DeleteShipmentCommentQuery,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		sc := shipmentdb.ShipmentComments
		n, err := sc.Where(sc.ID.Eq(req.CommentID), sc.ShipmentID.Eq(req.ShipmentID)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("delete shipment comment: %w", err)
		}

		return dberror.CheckAffected(n, "ShipmentComment", req.CommentID.String())
	})
}

func (r *repository) SoftDelete(
	ctx context.Context,
	req *shipmentports.SoftDeleteShipmentCommentRequest,
) (*shipment.ShipmentComment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentComment, error) {
		sc := shipmentdb.ShipmentComments
		n, err := sc.Where(
			sc.ID.Eq(req.CommentID),
			sc.ShipmentID.Eq(req.ShipmentID),
			sc.DeletedAt.IsNull(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			sc.DeletedAt.To(timeutils.NowUnix()),
			sc.DeletedByID.To(req.ActorID),
			sc.Comment.To(""),
			sc.Body.ToNull(),
			sc.PinnedAt.ToNull(),
			sc.PinnedByID.ToNull(),
			sc.RequiresAcknowledgment.To(false),
		)
		if err != nil {
			return nil, fmt.Errorf("soft delete shipment comment: %w", err)
		}
		if err = dberror.CheckAffected(n, commentEntity, req.CommentID.String()); err != nil {
			return nil, err
		}

		if err = r.deleteMentions(ctx, req.CommentID); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &shipmentports.GetShipmentCommentByIDRequest{
			CommentID:  req.CommentID,
			ShipmentID: req.ShipmentID,
			TenantInfo: req.TenantInfo,
		})
	})
}

func (r *repository) SetPinned(
	ctx context.Context,
	req *shipmentports.SetShipmentCommentPinnedRequest,
) (*shipment.ShipmentComment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentComment, error) {
		sc := shipmentdb.ShipmentComments
		sets := []keel.Assignment{sc.PinnedAt.ToNull(), sc.PinnedByID.ToNull()}
		if req.Pinned {
			sets = []keel.Assignment{
				sc.PinnedAt.To(timeutils.NowUnix()),
				sc.PinnedByID.To(req.ActorID),
			}
		}

		n, err := sc.Where(
			sc.ID.Eq(req.CommentID),
			sc.ShipmentID.Eq(req.ShipmentID),
			sc.DeletedAt.IsNull(),
			sc.ParentCommentID.IsNull(),
		).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return nil, fmt.Errorf("set shipment comment pinned: %w", err)
		}
		if err = dberror.CheckAffected(n, commentEntity, req.CommentID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &shipmentports.GetShipmentCommentByIDRequest{
			CommentID:  req.CommentID,
			ShipmentID: req.ShipmentID,
			TenantInfo: req.TenantInfo,
		})
	})
}

func (r *repository) SetResolved(
	ctx context.Context,
	req *shipmentports.SetShipmentCommentResolvedRequest,
) (*shipment.ShipmentComment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentComment, error) {
		sc := shipmentdb.ShipmentComments
		sets := []keel.Assignment{sc.ResolvedAt.ToNull(), sc.ResolvedByID.ToNull()}
		if req.Resolved {
			sets = []keel.Assignment{
				sc.ResolvedAt.To(timeutils.NowUnix()),
				sc.ResolvedByID.To(req.ActorID),
			}
		}

		n, err := sc.Where(
			sc.ID.Eq(req.CommentID),
			sc.ShipmentID.Eq(req.ShipmentID),
			sc.DeletedAt.IsNull(),
		).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return nil, fmt.Errorf("set shipment comment resolved: %w", err)
		}
		if err = dberror.CheckAffected(n, commentEntity, req.CommentID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &shipmentports.GetShipmentCommentByIDRequest{
			CommentID:  req.CommentID,
			ShipmentID: req.ShipmentID,
			TenantInfo: req.TenantInfo,
		})
	})
}

func (r *repository) Acknowledge(
	ctx context.Context,
	req *shipmentports.AcknowledgeShipmentCommentRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		ack := &shipment.ShipmentCommentAcknowledgment{
			CommentID:      req.CommentID,
			BusinessUnitID: req.TenantInfo.BuID,
			OrganizationID: req.TenantInfo.OrgID,
			ShipmentID:     req.ShipmentID,
			UserID:         req.UserID,
		}

		a := shipmentdb.ShipmentCommentAcknowledgments
		n, err := a.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*shipment.ShipmentCommentAcknowledgment{ack},
			a.CommentID,
			a.UserID,
		)
		if err != nil {
			return false, fmt.Errorf("acknowledge shipment comment: %w", err)
		}

		return n > 0, nil
	})
}

func (r *repository) CountReplies(
	ctx context.Context,
	req *shipmentports.GetShipmentCommentByIDRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		sc := shipmentdb.ShipmentComments
		n, err := sc.Where(sc.ParentCommentID.Eq(req.CommentID), sc.ShipmentID.Eq(req.ShipmentID)).
			Count(ctx, r.db.Keel(ctx))
		return int(n), err
	})
}

func (r *repository) CountPinnedByShipmentID(
	ctx context.Context,
	req *shipmentports.GetShipmentCommentCountRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		sc := shipmentdb.ShipmentComments
		n, err := sc.Where(
			sc.ShipmentID.Eq(req.ShipmentID),
			sc.PinnedAt.IsNotNull(),
			sc.DeletedAt.IsNull(),
		).Count(ctx, r.db.Keel(ctx))
		return int(n), err
	})
}

func (r *repository) insertMentions(ctx context.Context, entity *shipment.ShipmentComment) error {
	if len(entity.MentionedUsers) == 0 {
		return nil
	}

	for _, mention := range entity.MentionedUsers {
		mention.CommentID = entity.ID
		mention.ShipmentID = entity.ShipmentID
		mention.OrganizationID = entity.OrganizationID
		mention.BusinessUnitID = entity.BusinessUnitID
	}

	if err := shipmentdb.ShipmentCommentMentions.CreateMany(
		ctx,
		r.db.Keel(ctx),
		entity.MentionedUsers,
	); err != nil {
		return fmt.Errorf("insert shipment comment mentions: %w", err)
	}

	return nil
}

func (r *repository) deleteMentions(ctx context.Context, commentID pulid.ID) error {
	m := shipmentdb.ShipmentCommentMentions
	if _, err := m.Where(m.CommentID.Eq(commentID)).Delete(ctx, r.db.Keel(ctx)); err != nil {
		return fmt.Errorf("delete shipment comment mentions: %w", err)
	}

	return nil
}

func tenantInfo(entity *shipment.ShipmentComment) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}
