package shipmentcommentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.ShipmentCommentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-comment-repository"),
	}
}

func (r *repository) ListByShipmentID(
	ctx context.Context,
	req *repositories.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	sc := buncolgen.ShipmentCommentColumns
	replyMode := req.ParentCommentID != nil && !req.ParentCommentID.IsNil()

	direction := "desc"
	if replyMode {
		direction = "asc"
	}
	cursorSort := []pagination.CursorSortField{
		{Field: "createdAt", Direction: direction},
		{Field: "id", Direction: direction},
	}
	cursorColumns := []pagination.CursorValueColumn{
		{SQLExpression: sc.CreatedAt.Qualified(), Alias: "__cursor_value_0"},
		{SQLExpression: sc.ID.Qualified(), Alias: "__cursor_value_1"},
	}
	db := r.db.DBForContext(ctx)

	if req.Cursor.After != "" {
		if err := pagination.ValidateCursorSort(req.Cursor.Cursor, cursorSort); err != nil {
			return nil, errortypes.NewValidationError(
				"after",
				errortypes.ErrInvalid,
				"Cursor sort does not match request sort",
			)
		}
	}

	total, err := db.NewSelect().
		Model((*shipment.ShipmentComment)(nil)).
		Apply(r.applyListConditions(req)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*shipment.ShipmentComment]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(items *[]*shipment.ShipmentComment) *bun.SelectQuery {
			q := db.NewSelect().
				Model(items).
				ColumnExpr(buncolgen.ShipmentCommentTable.All()).
				Apply(r.applyListConditions(req)).
				Apply(applyCommentRelations)
			if !replyMode {
				q = q.ColumnExpr(replyCountExpr())
			}
			return q
		},
		Apply: func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
			if req.Cursor.After != "" {
				q = q.WhereGroup(" AND ", func(cq *bun.SelectQuery) *bun.SelectQuery {
					if replyMode {
						return cq.
							Where(sc.CreatedAt.Gt(), req.Cursor.Cursor.CreatedAt).
							WhereOr(
								sc.CreatedAt.Eq()+" AND "+sc.ID.Gt(),
								req.Cursor.Cursor.CreatedAt,
								req.Cursor.Cursor.ID,
							)
					}
					return cq.
						Where(sc.CreatedAt.Lt(), req.Cursor.Cursor.CreatedAt).
						WhereOr(
							sc.CreatedAt.Eq()+" AND "+sc.ID.Lt(),
							req.Cursor.Cursor.CreatedAt,
							req.Cursor.Cursor.ID,
						)
				})
			}
			req.Filter.CursorSort = cursorSort
			req.Filter.CursorColumns = cursorColumns
			if replyMode {
				return q.
					Order(sc.CreatedAt.OrderAsc()).
					Order(sc.ID.OrderAsc()), nil
			}
			return q.
				Order(sc.CreatedAt.OrderDesc()).
				Order(sc.ID.OrderDesc()), nil
		},
	})
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment comment")
	}

	return result, nil
}

func (r *repository) applyListConditions(
	req *repositories.ListShipmentCommentsRequest,
) func(*bun.SelectQuery) *bun.SelectQuery {
	sc := buncolgen.ShipmentCommentColumns
	scm := buncolgen.ShipmentCommentMentionColumns
	f := req.Filters
	replyMode := req.ParentCommentID != nil && !req.ParentCommentID.IsNil()

	return func(q *bun.SelectQuery) *bun.SelectQuery {
		q = q.
			Where(sc.ShipmentID.Eq(), req.ShipmentID).
			Apply(buncolgen.ShipmentCommentApplyTenant(req.Filter.TenantInfo))

		if replyMode {
			q = q.Where(sc.ParentCommentID.Eq(), *req.ParentCommentID)
		} else {
			q = q.Where(sc.ParentCommentID.IsNull())
			if f.HasAny() {
				q = q.Where(sc.DeletedAt.IsNull())
			}
		}

		if len(f.Types) > 0 {
			q = q.Where(sc.Type.In(), bun.List(f.Types))
		}
		if len(f.Priorities) > 0 {
			q = q.Where(sc.Priority.In(), bun.List(f.Priorities))
		}
		if len(f.AuthorIDs) > 0 {
			q = q.Where(sc.UserID.In(), bun.List(f.AuthorIDs))
		}
		if f.MentionsUserID != nil && !f.MentionsUserID.IsNil() {
			mentionExists := fmt.Sprintf(
				"EXISTS (SELECT 1 FROM %s %s WHERE %s AND %s)",
				buncolgen.ShipmentCommentMentionTable.Name,
				buncolgen.ShipmentCommentMentionTable.Alias,
				scm.CommentID.EqColumn(sc.ID),
				scm.MentionedUserID.Eq(),
			)
			q = q.Where(mentionExists, *f.MentionsUserID)
		}
		if f.PinnedOnly {
			q = q.Where(sc.PinnedAt.IsNotNull())
		}
		if f.UnresolvedOnly {
			q = q.Where(sc.ResolvedAt.IsNull())
		}
		if f.Search != "" {
			if dbdialect.FromBun(q.DB()).Supports(dbdialect.CapFullTextSearch) {
				q = q.Where(
					sc.SearchVector.Expr("{} @@ websearch_to_tsquery('english', ?)"),
					f.Search,
				)
			} else {
				q = q.Where(sc.Comment.Expr("{} LIKE ?"), "%"+f.Search+"%")
			}
		}

		return q
	}
}

func replyCountExpr() string {
	sc := buncolgen.ShipmentCommentColumns
	scr := struct {
		ParentCommentID buncolgen.Column
		OrganizationID  buncolgen.Column
		BusinessUnitID  buncolgen.Column
		ShipmentID      buncolgen.Column
	}{
		ParentCommentID: sc.ParentCommentID.WithAlias("scr"),
		OrganizationID:  sc.OrganizationID.WithAlias("scr"),
		BusinessUnitID:  sc.BusinessUnitID.WithAlias("scr"),
		ShipmentID:      sc.ShipmentID.WithAlias("scr"),
	}

	return fmt.Sprintf(
		"(SELECT count(*) FROM %s scr WHERE %s AND %s AND %s AND %s) AS reply_count",
		buncolgen.ShipmentCommentTable.Name,
		scr.ParentCommentID.EqColumn(sc.ID),
		scr.OrganizationID.EqColumn(sc.OrganizationID),
		scr.BusinessUnitID.EqColumn(sc.BusinessUnitID),
		scr.ShipmentID.EqColumn(sc.ShipmentID),
	)
}

func applyCommentRelations(q *bun.SelectQuery) *bun.SelectQuery {
	scm := buncolgen.ShipmentCommentMentionColumns
	sca := buncolgen.ShipmentCommentAcknowledgmentColumns
	rel := buncolgen.ShipmentCommentRelations

	return q.
		Relation(rel.User).
		Relation(rel.PinnedBy).
		Relation(rel.ResolvedBy).
		Relation(rel.MentionedUsers, func(mq *bun.SelectQuery) *bun.SelectQuery {
			return mq.Order(scm.CreatedAt.OrderAsc())
		}).
		Relation(buncolgen.Rel(
			rel.MentionedUsers,
			buncolgen.ShipmentCommentMentionRelations.MentionedUser,
		)).
		Relation(rel.Acknowledgments, func(aq *bun.SelectQuery) *bun.SelectQuery {
			return aq.Order(sca.AcknowledgedAt.OrderAsc())
		}).
		Relation(buncolgen.Rel(
			rel.Acknowledgments,
			buncolgen.ShipmentCommentAcknowledgmentRelations.User,
		))
}

func (r *repository) GetCountByShipmentID(
	ctx context.Context,
	req *repositories.GetShipmentCommentCountRequest,
) (int, error) {
	sc := buncolgen.ShipmentCommentColumns
	db := r.db.DBForContext(ctx)

	return db.NewSelect().
		Model((*shipment.ShipmentComment)(nil)).
		Where(sc.ShipmentID.Eq(), req.ShipmentID).
		Where(sc.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(sc.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(sc.DeletedAt.IsNull()).
		Count(ctx)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetShipmentCommentByIDRequest,
) (*shipment.ShipmentComment, error) {
	sc := buncolgen.ShipmentCommentColumns
	db := r.db.DBForContext(ctx)
	entity := new(shipment.ShipmentComment)

	if err := db.NewSelect().
		Model(entity).
		ColumnExpr(buncolgen.ShipmentCommentTable.All()).
		ColumnExpr(replyCountExpr()).
		Where(sc.ID.Eq(), req.CommentID).
		Where(sc.ShipmentID.Eq(), req.ShipmentID).
		Apply(buncolgen.ShipmentCommentApplyTenant(req.TenantInfo)).
		Apply(applyCommentRelations).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment comment")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	if err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(entity).Exec(txCtx); err != nil {
			return fmt.Errorf("insert shipment comment: %w", err)
		}

		if err := r.replaceMentions(txCtx, tx, entity); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  entity.ID,
		ShipmentID: entity.ShipmentID,
		TenantInfo: tenantInfo(entity),
	})
}

//nolint:govet // existing scoped variable reuse is local and behavior-preserving
func (r *repository) Update(
	ctx context.Context,
	entity *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	sc := buncolgen.ShipmentCommentColumns
	if err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model(entity).
			Where(sc.ID.Eq(), entity.ID).
			Where(sc.ShipmentID.Eq(), entity.ShipmentID).
			Where(sc.OrganizationID.Eq(), entity.OrganizationID).
			Where(sc.BusinessUnitID.Eq(), entity.BusinessUnitID).
			Where(sc.Version.Eq(), entity.Version).
			Where(sc.DeletedAt.IsNull()).
			Set(sc.Type.Set(), entity.Type).
			Set(sc.Visibility.Set(), entity.Visibility).
			Set(sc.Priority.Set(), entity.Priority).
			Set(sc.Source.Set(), entity.Source).
			Set(sc.Metadata.Set(), entity.Metadata).
			Set(sc.Comment.Set(), entity.Comment).
			Set(sc.Body.Set(), entity.Body).
			Set(sc.RequiresAcknowledgment.Set(), entity.RequiresAcknowledgment).
			Set(sc.EditedAt.Set(), entity.EditedAt).
			Set(sc.Version.Inc(1)).
			Exec(txCtx)
		if err != nil {
			return fmt.Errorf("update shipment comment: %w", err)
		}
		if err := dberror.CheckRowsAffected(
			result,
			"Shipment comment",
			entity.ID.String(),
		); err != nil {
			return err
		}

		if err := r.deleteMentions(
			txCtx,
			tx,
			entity.ID,
			entity.ShipmentID,
			entity.OrganizationID,
			entity.BusinessUnitID,
		); err != nil {
			return err
		}
		if err := r.replaceMentions(txCtx, tx, entity); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  entity.ID,
		ShipmentID: entity.ShipmentID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteShipmentCommentRequest,
) error {
	sc := buncolgen.ShipmentCommentColumns

	db := r.db.DBForContext(ctx)
	result, err := db.NewDelete().
		Model((*shipment.ShipmentComment)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ShipmentCommentScopeTenantDelete(dq, req.TenantInfo).
				Where(sc.ID.Eq(), req.CommentID).
				Where(sc.ShipmentID.Eq(), req.ShipmentID)
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete shipment comment: %w", err)
	}

	return dberror.CheckRowsAffected(result, "ShipmentComment", req.CommentID.String())
}

func (r *repository) SoftDelete(
	ctx context.Context,
	req *repositories.SoftDeleteShipmentCommentRequest,
) (*shipment.ShipmentComment, error) {
	sc := buncolgen.ShipmentCommentColumns
	now := timeutils.NowUnix()

	if err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model((*shipment.ShipmentComment)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ShipmentCommentScopeTenantUpdate(uq, req.TenantInfo).
					Where(sc.ID.Eq(), req.CommentID).
					Where(sc.ShipmentID.Eq(), req.ShipmentID).
					Where(sc.DeletedAt.IsNull())
			}).
			Set(sc.DeletedAt.Set(), now).
			Set(sc.DeletedByID.Set(), req.ActorID).
			Set(sc.Comment.Set(), "").
			Set(sc.Body.SetNull()).
			Set(sc.PinnedAt.SetNull()).
			Set(sc.PinnedByID.SetNull()).
			Set(sc.RequiresAcknowledgment.Set(), false).
			Set(sc.UpdatedAt.Set(), now).
			Set(sc.Version.Inc(1)).
			Exec(txCtx)
		if err != nil {
			return fmt.Errorf("soft delete shipment comment: %w", err)
		}
		if err = dberror.CheckRowsAffected(
			result,
			"Shipment comment",
			req.CommentID.String(),
		); err != nil {
			return err
		}

		return r.deleteMentions(
			txCtx,
			tx,
			req.CommentID,
			req.ShipmentID,
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
		)
	}); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  req.CommentID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
}

func (r *repository) SetPinned(
	ctx context.Context,
	req *repositories.SetShipmentCommentPinnedRequest,
) (*shipment.ShipmentComment, error) {
	sc := buncolgen.ShipmentCommentColumns
	now := timeutils.NowUnix()
	db := r.db.DBForContext(ctx)

	q := db.NewUpdate().
		Model((*shipment.ShipmentComment)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ShipmentCommentScopeTenantUpdate(uq, req.TenantInfo).
				Where(sc.ID.Eq(), req.CommentID).
				Where(sc.ShipmentID.Eq(), req.ShipmentID).
				Where(sc.DeletedAt.IsNull()).
				Where(sc.ParentCommentID.IsNull())
		}).
		Set(sc.UpdatedAt.Set(), now).
		Set(sc.Version.Inc(1))

	if req.Pinned {
		q = q.
			Set(sc.PinnedAt.Set(), now).
			Set(sc.PinnedByID.Set(), req.ActorID)
	} else {
		q = q.
			Set(sc.PinnedAt.SetNull()).
			Set(sc.PinnedByID.SetNull())
	}

	result, err := q.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("set shipment comment pinned: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		result,
		"Shipment comment",
		req.CommentID.String(),
	); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  req.CommentID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
}

func (r *repository) SetResolved(
	ctx context.Context,
	req *repositories.SetShipmentCommentResolvedRequest,
) (*shipment.ShipmentComment, error) {
	sc := buncolgen.ShipmentCommentColumns
	now := timeutils.NowUnix()
	db := r.db.DBForContext(ctx)

	q := db.NewUpdate().
		Model((*shipment.ShipmentComment)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ShipmentCommentScopeTenantUpdate(uq, req.TenantInfo).
				Where(sc.ID.Eq(), req.CommentID).
				Where(sc.ShipmentID.Eq(), req.ShipmentID).
				Where(sc.DeletedAt.IsNull())
		}).
		Set(sc.UpdatedAt.Set(), now).
		Set(sc.Version.Inc(1))

	if req.Resolved {
		q = q.
			Set(sc.ResolvedAt.Set(), now).
			Set(sc.ResolvedByID.Set(), req.ActorID)
	} else {
		q = q.
			Set(sc.ResolvedAt.SetNull()).
			Set(sc.ResolvedByID.SetNull())
	}

	result, err := q.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("set shipment comment resolved: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		result,
		"Shipment comment",
		req.CommentID.String(),
	); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  req.CommentID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
}

func (r *repository) Acknowledge(
	ctx context.Context,
	req *repositories.AcknowledgeShipmentCommentRequest,
) (bool, error) {
	db := r.db.DBForContext(ctx)

	ack := &shipment.ShipmentCommentAcknowledgment{
		CommentID:      req.CommentID,
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		ShipmentID:     req.ShipmentID,
		UserID:         req.UserID,
	}

	result, err := db.NewInsert().
		Model(ack).
		On("CONFLICT (comment_id, user_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("acknowledge shipment comment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("acknowledge shipment comment rows affected: %w", err)
	}

	return rows > 0, nil
}

func (r *repository) CountReplies(
	ctx context.Context,
	req *repositories.GetShipmentCommentByIDRequest,
) (int, error) {
	sc := buncolgen.ShipmentCommentColumns
	db := r.db.DBForContext(ctx)

	return db.NewSelect().
		Model((*shipment.ShipmentComment)(nil)).
		Where(sc.ParentCommentID.Eq(), req.CommentID).
		Where(sc.ShipmentID.Eq(), req.ShipmentID).
		Apply(buncolgen.ShipmentCommentApplyTenant(req.TenantInfo)).
		Count(ctx)
}

func (r *repository) CountPinnedByShipmentID(
	ctx context.Context,
	req *repositories.GetShipmentCommentCountRequest,
) (int, error) {
	sc := buncolgen.ShipmentCommentColumns
	db := r.db.DBForContext(ctx)

	return db.NewSelect().
		Model((*shipment.ShipmentComment)(nil)).
		Where(sc.ShipmentID.Eq(), req.ShipmentID).
		Where(sc.PinnedAt.IsNotNull()).
		Where(sc.DeletedAt.IsNull()).
		Apply(buncolgen.ShipmentCommentApplyTenant(req.TenantInfo)).
		Count(ctx)
}

func (r *repository) replaceMentions(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.ShipmentComment,
) error {
	if len(entity.MentionedUsers) == 0 {
		return nil
	}

	for _, mention := range entity.MentionedUsers {
		mention.CommentID = entity.ID
		mention.ShipmentID = entity.ShipmentID
		mention.OrganizationID = entity.OrganizationID
		mention.BusinessUnitID = entity.BusinessUnitID
	}

	if _, err := tx.NewInsert().Model(&entity.MentionedUsers).Exec(ctx); err != nil {
		return fmt.Errorf("insert shipment comment mentions: %w", err)
	}

	return nil
}

func (r *repository) deleteMentions(
	ctx context.Context,
	tx bun.IDB,
	commentID, shipmentID, orgID, buID pulid.ID,
) error {
	scm := buncolgen.ShipmentCommentMentionColumns
	if _, err := tx.NewDelete().
		Model((*shipment.ShipmentCommentMention)(nil)).
		Where(scm.CommentID.Eq(), commentID).
		Where(scm.ShipmentID.Eq(), shipmentID).
		Where(scm.OrganizationID.Eq(), orgID).
		Where(scm.BusinessUnitID.Eq(), buID).
		Exec(ctx); err != nil {
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
