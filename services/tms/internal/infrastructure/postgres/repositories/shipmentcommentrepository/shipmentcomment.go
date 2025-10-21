package shipmentcommentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func NewRepository(p Params) repositories.ShipmentCommentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipmentcomment-repository"),
	}
}

func (r *repository) ListByShipmentID(
	ctx context.Context,
	req repositories.GetCommentsByShipmentIDRequest,
) (*pagination.ListResult[*shipment.ShipmentComment], error) {
	log := r.l.With(
		zap.String("operation", "ListByShipmentID"),
		zap.String("shipmentID", req.ShipmentID.String()),
		zap.String("orgID", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buID", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	comments := make([]*shipment.ShipmentComment, 0, req.Filter.Limit)

	total, err := db.NewSelect().
		Model(&comments).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sc.shipment_id = ?", req.ShipmentID).
				Where("sc.organization_id = ?", req.Filter.TenantOpts.OrgID).
				Where("sc.business_unit_id = ?", req.Filter.TenantOpts.BuID)
		}).
		Relation("User").
		Relation("MentionedUsers").
		Relation("MentionedUsers.MentionedUser").
		Order("sc.created_at ASC").
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan shipment comments", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Shipment Comment")
	}

	return &pagination.ListResult[*shipment.ShipmentComment]{
		Items: comments,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetCommentByIDRequest,
) (*shipment.ShipmentComment, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("commentID", req.CommentID.String()),
		zap.String("orgID", req.OrgID.String()),
		zap.String("buID", req.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	comment := new(shipment.ShipmentComment)

	if err = db.NewSelect().Model(comment).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sc.id = ?", req.CommentID).
				Where("sc.organization_id = ?", req.OrgID).
				Where("sc.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx); err != nil {
		log.Error("failed to scan shipment comment", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Shipment Comment")
	}

	return comment, nil
}

func (r *repository) GetCountByShipmentID(
	ctx context.Context,
	req repositories.GetShipmentCommentCountRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "GetCountByShipmentID"),
		zap.String("shipmentID", req.ShipmentID.String()),
		zap.String("orgID", req.OrgID.String()),
		zap.String("buID", req.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return 0, err
	}

	count, err := db.NewSelect().Model((*shipment.ShipmentComment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sc.shipment_id = ?", req.ShipmentID).
				Where("sc.organization_id = ?", req.OrgID).
				Where("sc.business_unit_id = ?", req.BuID)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to get shipment comment count", zap.Error(err))
		return 0, dberror.HandleNotFoundError(err, "Shipment Comment")
	}

	return count, nil
}

func (r *repository) Create(
	ctx context.Context,
	comment *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("orgID", comment.OrganizationID.String()),
		zap.String("buID", comment.BusinessUnitID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(comment).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create shipment comment", zap.Error(err))
		return nil, err
	}

	if len(comment.MentionedUsers) > 0 {
		for _, mentionedUser := range comment.MentionedUsers {
			mentionedUser.CommentID = comment.ID
			mentionedUser.BusinessUnitID = comment.BusinessUnitID
			mentionedUser.OrganizationID = comment.OrganizationID
			mentionedUser.ShipmentID = comment.ShipmentID

			if _, err = db.NewInsert().Model(mentionedUser).
				Returning("*").
				Exec(ctx); err != nil {
				log.Error("failed to insert shipment comment mention", zap.Error(err))
				return nil, err
			}
		}
	}

	createdComment := new(shipment.ShipmentComment)
	if err = db.NewSelect().
		Model(createdComment).
		Where("sc.id = ?", comment.ID).
		Relation("User").
		Relation("MentionedUsers").
		Relation("MentionedUsers.MentionedUser").
		Scan(ctx); err != nil {
		log.Error("failed to fetch created comment with relations", zap.Error(err))
		return nil, err
	}

	return createdComment, nil
}

func (r *repository) Update(
	ctx context.Context,
	comment *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", comment.OrganizationID.String()),
		zap.String("buID", comment.BusinessUnitID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		comment.Version++

		res, rErr := tx.NewUpdate().Model(comment).
			With("_data", tx.NewValues(comment)).
			TableExpr("_data").
			Set("shipment_id = _data.shipment_id").
			Set("user_id = _data.user_id").
			Set("comment = _data.comment").
			Set("comment_type = _data.comment_type").
			Set("metadata = _data.metadata").
			Set("version = _data.version + 1").
			WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
				return sq.
					Where("sc.id = _data.id").
					Where("sc.version = _data.version - 1").
					Where("sc.organization_id = _data.organization_id").
					Where("sc.business_unit_id = _data.business_unit_id")
			}).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to update shipment comment", zap.Error(rErr))
			return err
		}

		roErr := dberror.CheckRowsAffected(res, "Shipment Comment", comment.ID.String())
		if roErr != nil {
			return roErr
		}

		if _, err = tx.NewDelete().
			Model((*shipment.ShipmentCommentMention)(nil)).
			Where("comment_id = ?", comment.ID).
			Exec(c); err != nil {
			log.Error("failed to delete existing shipment comment mentions", zap.Error(err))
			return err
		}

		if len(comment.MentionedUsers) > 0 {
			for _, mentionedUser := range comment.MentionedUsers {
				mentionedUser.CommentID = comment.ID
				mentionedUser.BusinessUnitID = comment.BusinessUnitID
				mentionedUser.OrganizationID = comment.OrganizationID
				mentionedUser.ShipmentID = comment.ShipmentID

				if _, err = tx.NewInsert().Model(mentionedUser).Exec(c); err != nil {
					log.Error("failed to insert shipment comment mention", zap.Error(err))
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteCommentRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("orgID", req.OrgID.String()),
		zap.String("buID", req.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	if _, err = db.NewDelete().
		Model((*shipment.ShipmentComment)(nil)).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.
				Where("sc.id = ?", req.CommentID).
				Where("sc.organization_id = ?", req.OrgID).
				Where("sc.business_unit_id = ?", req.BuID).
				Where("sc.shipment_id = ?", req.ShipmentID).
				Where("sc.user_id = ?", req.UserID)
		}).
		Exec(ctx); err != nil {
		log.Error("failed to delete shipment comment", zap.Error(err))
		return err
	}

	return nil
}
