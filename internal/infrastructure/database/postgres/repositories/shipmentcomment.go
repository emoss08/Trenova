/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
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

func (scr *shipmentCommentRepository) ListByShipmentID(
	ctx context.Context,
	req repositories.GetCommentsByShipmentIDRequest,
) (*ports.ListResult[*shipment.ShipmentComment], error) {
	dba, err := scr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := scr.l.With().
		Str("operation", "ListByShipmentID").
		Str("shipmentID", req.ShipmentID.String()).
		Str("orgID", req.Filter.TenantOpts.OrgID.String()).
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Logger()

	comments := make([]*shipment.ShipmentComment, 0)

	total, err := dba.NewSelect().
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
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get shipment comments")
			return nil, errors.NewNotFoundError(
				"Shipment Comments not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get shipment comments")
		return nil, err
	}

	return &ports.ListResult[*shipment.ShipmentComment]{
		Items: comments,
		Total: total,
	}, nil
}

func (scr *shipmentCommentRepository) Create(
	ctx context.Context,
	comment *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	dba, err := scr.db.WriteDB(ctx)
	if err != nil {
		return nil, err
	}

	log := scr.l.With().
		Str("operation", "Create").
		Str("orgID", comment.OrganizationID.String()).
		Str("buID", comment.BusinessUnitID.String()).
		Logger()

	if _, err = dba.NewInsert().Model(comment).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to create shipment comment")
		return nil, err
	}

	// * Loop over possible mentioned users and create a mention for each
	if len(comment.MentionedUsers) > 0 {
		for _, mentionedUser := range comment.MentionedUsers {
			mentionedUser.CommentID = comment.ID
			mentionedUser.BusinessUnitID = comment.BusinessUnitID
			mentionedUser.OrganizationID = comment.OrganizationID
			mentionedUser.ShipmentID = comment.ShipmentID

			if _, err := dba.NewInsert().Model(mentionedUser).Exec(ctx); err != nil {
				log.Error().Err(err).Msg("failed to insert shipment comment mention")
				return nil, err
			}

			comment.MentionedUsers = append(comment.MentionedUsers, mentionedUser)
		}
	}

	return comment, nil
}
