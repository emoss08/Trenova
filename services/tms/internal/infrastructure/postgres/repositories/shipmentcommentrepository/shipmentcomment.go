package shipmentcommentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	scm := buncolgen.ShipmentCommentMentionColumns
	cursorSort := []pagination.CursorSortField{
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
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
		Where(sc.ShipmentID.Eq(), req.ShipmentID).
		Apply(buncolgen.ShipmentCommentApplyTenant(req.Filter.TenantInfo)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*shipment.ShipmentComment]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(items *[]*shipment.ShipmentComment) *bun.SelectQuery {
			return db.NewSelect().
				Model(items).
				ColumnExpr(buncolgen.ShipmentCommentTable.All()).
				Where(sc.ShipmentID.Eq(), req.ShipmentID).
				Apply(buncolgen.ShipmentCommentApplyTenant(req.Filter.TenantInfo)).
				Relation(buncolgen.ShipmentCommentRelations.User).
				Relation(buncolgen.ShipmentCommentRelations.MentionedUsers, func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Order(scm.CreatedAt.OrderDesc())
				}).
				Relation(buncolgen.Rel(
					buncolgen.ShipmentCommentRelations.MentionedUsers,
					buncolgen.ShipmentCommentMentionRelations.MentionedUser,
				))
		},
		Apply: func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
			if req.Cursor.After != "" {
				q = q.WhereGroup(" AND ", func(cq *bun.SelectQuery) *bun.SelectQuery {
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
		Count(ctx)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetShipmentCommentByIDRequest,
) (*shipment.ShipmentComment, error) {
	sc := buncolgen.ShipmentCommentColumns
	scm := buncolgen.ShipmentCommentMentionColumns
	db := r.db.DBForContext(ctx)
	entity := new(shipment.ShipmentComment)

	if err := db.NewSelect().
		Model(entity).
		Where(sc.ID.Eq(), req.CommentID).
		Where(sc.ShipmentID.Eq(), req.ShipmentID).
		Apply(buncolgen.ShipmentCommentApplyTenant(req.TenantInfo)).
		Relation(buncolgen.ShipmentCommentRelations.User).
		Relation(buncolgen.ShipmentCommentRelations.MentionedUsers, func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order(scm.CreatedAt.OrderAsc())
		}).
		Relation(buncolgen.Rel(
			buncolgen.ShipmentCommentRelations.MentionedUsers,
			buncolgen.ShipmentCommentMentionRelations.MentionedUser,
		)).
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
			Set(sc.Type.Set(), entity.Type).
			Set(sc.Visibility.Set(), entity.Visibility).
			Set(sc.Priority.Set(), entity.Priority).
			Set(sc.Source.Set(), entity.Source).
			Set(sc.Metadata.Set(), entity.Metadata).
			Set(sc.Comment.Set(), entity.Comment).
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
