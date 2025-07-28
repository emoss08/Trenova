/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ShipmentCommentRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type shipmentCommentRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewShipmentCommentRepository(
	p ShipmentCommentRepositoryParams,
) repositories.ShipmentCommentRepository {
	log := p.Logger.With().
		Str("repository", "shipmentcomment").
		Logger()

	return &shipmentCommentRepository{
		db: p.DB,
		l:  &log,
	}
}

func (scr *shipmentCommentRepository) GetByID(
	ctx context.Context,
	req repositories.GetCommentByIDRequest,
) (*shipment.ShipmentComment, error) {
	dba, err := scr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := scr.l.With().
		Str("operation", "GetByID").
		Str("commentID", req.CommentID.String()).
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	entity, err := shipment.NewShipmentCommentQuery(dba).
		WhereGroup(" AND ", func(scqb *shipment.ShipmentCommentQueryBuilder) *shipment.ShipmentCommentQueryBuilder {
			return scqb.
				WhereIDEQ(req.CommentID).
				WhereTenant(req.OrgID, req.BuID)
		}).
		First(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Shipment Comment not found")
		}

		log.Error().Err(err).Msg("failed to get shipment comment")
		return nil, err
	}

	return entity, nil
}

func (scr *shipmentCommentRepository) GetByShipmentID(
	ctx context.Context,
	req repositories.GetCommentsByShipmentIDRequest,
) ([]*shipment.ShipmentComment, error) {
	dba, err := scr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := scr.l.With().
		Str("operation", "GetByShipmentID").
		Str("shipmentID", req.ShipmentID.String()).
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	comments := make([]*shipment.ShipmentComment, 0)

	err = dba.NewSelect().
		Model(&comments).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sc.shipment_id = ?", req.ShipmentID).
				Where("sc.organization_id = ?", req.OrgID).
				Where("sc.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get shipment comments")
			return nil, errors.NewNotFoundError(
				"Shipment Comments not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get shipment comments")
		return nil, err
	}

	return comments, nil
}

func (scr *shipmentCommentRepository) BulkInsert(
	ctx context.Context,
	comments []*shipment.ShipmentComment,
) ([]*shipment.ShipmentComment, error) {
	dba, err := scr.db.WriteDB(ctx)
	if err != nil {
		return nil, err
	}

	log := scr.l.With().
		Str("operation", "BulkInsert").
		Interface("comments", comments).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&comments).Exec(c); err != nil {
			log.Error().Err(err).Msg("failed to bulk insert comments")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk insert comments")
		return nil, err
	}

	return comments, nil
}

func (scr *shipmentCommentRepository) HandleCommentOperations(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	isCreate bool,
) error {
	log := scr.l.With().
		Str("operation", "HandleCommentOperations").
		Str("shipmentID", shp.ID.String()).
		Logger()

	data, err := scr.prepareCommentsData(ctx, shp, isCreate)
	if err != nil {
		log.Error().Err(err).Msg("failed to prepare comments data")
		return err
	}

	if err = scr.processNewComments(ctx, tx, data.newComments); err != nil {
		return err
	}

	if err = scr.processUpdateComments(ctx, tx, data.updateComments); err != nil {
		return err
	}

	if !isCreate {
		if err = scr.checkAndHandleCommentDeletions(ctx, tx, shp.ID, data); err != nil {
			return err
		}
	}

	log.Debug().Int("new_comments", len(data.newComments)).
		Int("updated_comments", len(data.updateComments)).
		Msg("comment operations completed")

	return nil
}

type commentsOperationData struct {
	newComments         []*shipment.ShipmentComment
	updateComments      []*shipment.ShipmentComment
	existingCommentsMap map[pulid.ID]*shipment.ShipmentComment
	updatedCommentIDs   map[pulid.ID]struct{}
	commentToDelete     []*shipment.ShipmentComment
	existingComments    []*shipment.ShipmentComment
}

func (scr *shipmentCommentRepository) prepareCommentsData(
	ctx context.Context,
	shp *shipment.Shipment,
	isCreate bool,
) (*commentsOperationData, error) {
	log := scr.l.With().
		Str("operation", "prepareCommentsData").
		Str("shipmentID", shp.ID.String()).
		Logger()

	data := &commentsOperationData{
		newComments:         make([]*shipment.ShipmentComment, 0),
		updateComments:      make([]*shipment.ShipmentComment, 0),
		existingCommentsMap: make(map[pulid.ID]*shipment.ShipmentComment),
		updatedCommentIDs:   make(map[pulid.ID]struct{}),
		commentToDelete:     make([]*shipment.ShipmentComment, 0),
		existingComments:    make([]*shipment.ShipmentComment, 0),
	}

	if !isCreate {
		var err error
		data.existingComments, err = scr.GetByShipmentID(
			ctx,
			repositories.GetCommentsByShipmentIDRequest{
				ShipmentID: shp.ID,
				OrgID:      shp.OrganizationID,
				BuID:       shp.BusinessUnitID,
			},
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to get existing comments")
			return nil, err
		}

		for _, comment := range data.existingComments {
			log.Debug().Interface("comment", comment).Msg("existing comment")
			data.existingCommentsMap[comment.ID] = comment
		}
	}

	scr.categorizeComments(shp, data, isCreate)
	return data, nil
}

func (scr *shipmentCommentRepository) categorizeComments(
	shp *shipment.Shipment,
	data *commentsOperationData,
	isCreate bool,
) {
	for _, comment := range shp.Comments {
		comment.ShipmentID = shp.ID
		comment.OrganizationID = shp.OrganizationID
		comment.BusinessUnitID = shp.BusinessUnitID

		if isCreate || comment.ID.IsNil() {
			comment.ID = pulid.MustNew("sc_")
			data.newComments = append(data.newComments, comment)
		} else {
			if existing, ok := data.existingCommentsMap[comment.ID]; ok {
				comment.Version = existing.Version + 1
				data.updateComments = append(data.updateComments, comment)
				data.updatedCommentIDs[comment.ID] = struct{}{}
			}
		}
	}
}

func (scr *shipmentCommentRepository) processNewComments(
	ctx context.Context,
	tx bun.IDB,
	newComments []*shipment.ShipmentComment,
) error {
	if len(newComments) == 0 {
		return nil
	}

	log := scr.l.With().
		Str("operation", "processNewComments").
		Logger()

	if _, err := tx.NewInsert().Model(&newComments).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to bulk insert new comments")
		return err
	}

	return nil
}

func (scr *shipmentCommentRepository) processUpdateComments(
	ctx context.Context,
	tx bun.IDB,
	updateComments []*shipment.ShipmentComment,
) error {
	if len(updateComments) == 0 {
		return nil
	}

	log := scr.l.With().
		Str("operation", "processUpdateComments").
		Logger()

	for idx, comment := range updateComments {
		if err := scr.handleUpdate(ctx, tx, comment, idx); err != nil {
			log.Error().Err(err).Msg("failed to handle bulk update of comments")
			return err
		}
	}

	return nil
}

func (scr *shipmentCommentRepository) handleUpdate(
	ctx context.Context,
	tx bun.IDB,
	comment *shipment.ShipmentComment,
	idx int,
) error {
	log := scr.l.With().
		Str("operation", "handleUpdate").
		Int("idx", idx).
		Interface("comment", comment).
		Logger()

	values := tx.NewValues(comment)

	res, err := tx.NewUpdate().With("_data", values).
		Model(comment).
		TableExpr("_data").
		Set("shipment_id = _data.shipment_id").
		Set("organization_id = _data.organization_id").
		Set("business_unit_id = _data.business_unit_id").
		Set("comment = _data.comment").
		Set("is_high_priority = _data.is_high_priority").
		Set("version = _data.version").
		Where("sc.id = _data.id").
		Where("sc.version = _data.version - 1").
		Where("sc.organization_id = _data.organization_id").
		Where("sc.business_unit_id = _data.business_unit_id").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk update comments")
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected for bulk update of comments")
		return err
	}

	if rowsAffected == 0 {
		return errors.NewValidationError(
			fmt.Sprintf("comment[%d].version", idx),
			errors.ErrVersionMismatch,
			fmt.Sprintf(
				"Version mismatch. The comment (%s) has been updated since your last request.",
				comment.ID,
			),
		)
	}

	log.Debug().Int("count", int(rowsAffected)).Msg("bulk updated comments")

	return nil
}

func (scr *shipmentCommentRepository) checkAndHandleCommentDeletions(
	ctx context.Context,
	tx bun.IDB,
	shipmentID pulid.ID,
	data *commentsOperationData,
) error {
	log := scr.l.With().
		Str("operation", "checkAndHandleCommentDeletions").
		Str("shipmentID", shipmentID.String()).
		Logger()

	deletionRequired := false
	for commentID := range data.existingCommentsMap {
		if _, ok := data.updatedCommentIDs[commentID]; !ok {
			deletionRequired = true
			break
		}
	}

	if !deletionRequired {
		if err := scr.handleCommentDeletions(ctx, tx, &repositories.HandleCommentDeletionsRequest{
			ExistingCommentMap: data.existingCommentsMap,
			UpdatedCommentIDs:  data.updatedCommentIDs,
			CommentToDelete:    data.commentToDelete,
		}); err != nil {
			log.Error().Err(err).Msg("failed to handle comment deletions")
			return err
		}
	}

	return nil
}

func (scr *shipmentCommentRepository) handleCommentDeletions(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.HandleCommentDeletionsRequest,
) error {
	commentIDsToDelete := make([]pulid.ID, 0)
	for commentID, comment := range req.ExistingCommentMap {
		if _, ok := req.UpdatedCommentIDs[commentID]; !ok {
			commentIDsToDelete = append(commentIDsToDelete, commentID)
			req.CommentToDelete = append(req.CommentToDelete, comment)
		}
	}

	if len(commentIDsToDelete) > 0 {
		if err := scr.deleteComments(ctx, tx, commentIDsToDelete); err != nil {
			return err
		}
	}

	return nil
}

func (scr *shipmentCommentRepository) deleteComments(
	ctx context.Context,
	tx bun.IDB,
	commentIDs []pulid.ID,
) error {
	log := scr.l.With().
		Str("operation", "deleteComments").
		Interface("commentIDs", commentIDs).
		Logger()

	result, err := tx.NewDelete().
		Model((*shipment.ShipmentComment)(nil)).
		Where("id IN (?)", bun.In(commentIDs)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete comments")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected for comment deletion")
		return err
	}

	log.Info().Int64("deletedCommentCount", rowsAffected).
		Interface("commentIDs", commentIDs).
		Msg("successfully deleted comments")

	return nil
}
