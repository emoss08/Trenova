// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// DedicatedLaneSuggestionRepositoryParams defines dependencies required for initializing the DedicatedLaneSuggestionRepository.
// This includes database connection and logger.
type DedicatedLaneSuggestionRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// dedicatedLaneSuggestionRepository implements the DedicatedLaneSuggestionRepository interface
// and provides methods to manage dedicated lane suggestion data, including CRUD operations.
type dedicatedLaneSuggestionRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewDedicatedLaneSuggestionRepository initializes a new instance of dedicatedLaneSuggestionRepository with its dependencies.
//
// Parameters:
//   - p: DedicatedLaneSuggestionRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.DedicatedLaneSuggestionRepository: A ready-to-use dedicated lane suggestion repository instance.
func NewDedicatedLaneSuggestionRepository(
	p DedicatedLaneSuggestionRepositoryParams,
) repositories.DedicatedLaneSuggestionRepository {
	log := p.Logger.With().
		Str("repository", "dedicated_lane_suggestion").
		Logger()

	return &dedicatedLaneSuggestionRepository{
		db: p.DB,
		l:  &log,
	}
}

func (dlsr *dedicatedLaneSuggestionRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDedicatedLaneSuggestionRequest,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "dls",
		Filter:     req.Filter,
	})

	// Filter by status if provided
	if req.Status != nil {
		q = q.Where("dls.status = ?", *req.Status)
	}

	// Filter by customer if provided
	if req.CustomerID != nil {
		q = q.Where("dls.customer_id = ?", *req.CustomerID)
	}

	// Include/exclude expired suggestions
	if !req.IncludeExpired {
		now := timeutils.NowUnix()
		q = q.Where(
			"dls.expires_at > ? OR dls.status != ?",
			now,
			dedicatedlane.SuggestionStatusPending,
		)
	}

	// Include/exclude processed suggestions
	if !req.IncludeProcessed {
		q = q.Where("dls.status = ?", dedicatedlane.SuggestionStatusPending)
	}

	// Load relationships
	relations := []string{
		"Customer",
		"OriginLocation",
		"OriginLocation.State",
		"DestinationLocation",
		"DestinationLocation.State",
	}

	for _, rel := range relations {
		q = q.Relation(rel)
	}

	// Order by confidence score and creation date for best suggestions first
	q = q.Order("dls.confidence_score DESC", "dls.created_at DESC")

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (dlsr *dedicatedLaneSuggestionRepository) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneSuggestionRequest,
) (*ports.ListResult[*dedicatedlane.DedicatedLaneSuggestion], error) {
	dba, err := dlsr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "list").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlsr.l.With().
		Str("op", "list").
		Interface("tenantOps", req.Filter.TenantOpts).
		Logger()

	entities := make([]*dedicatedlane.DedicatedLaneSuggestion, 0)

	q := dba.NewSelect().Model(&entities)
	q = dlsr.filterQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan and count dedicated lane suggestions")
		return nil, err
	}

	return &ports.ListResult[*dedicatedlane.DedicatedLaneSuggestion]{
		Total: total,
		Items: entities,
	}, nil
}

func (dlsr *dedicatedLaneSuggestionRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDedicatedLaneSuggestionByIDRequest,
) (*dedicatedlane.DedicatedLaneSuggestion, error) {
	dba, err := dlsr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "get_by_id").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlsr.l.With().
		Str("op", "get_by_id").
		Interface("req", req).
		Logger()

	entity := &dedicatedlane.DedicatedLaneSuggestion{}

	q := dba.NewSelect().Model(entity).
		Relation("Customer").
		Relation("OriginLocation").
		Relation("OriginLocation.State").
		Relation("DestinationLocation").
		Relation("DestinationLocation.State").
		Relation("ServiceType").
		Relation("ShipmentType").
		Relation("TractorType").
		Relation("TrailerType").
		Relation("CreatedDedicatedLane").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("dls.organization_id = ?", req.OrgID).
				Where("dls.business_unit_id = ?", req.BuID).
				Where("dls.id = ?", req.ID)
		})

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, oops.In("dedicated_lane_suggestion_repository").
				With("op", "get_by_id").
				Time(time.Now()).
				Wrapf(err, "no dedicated lane suggestion found")
		}

		log.Error().Err(err).Msg("failed to get dedicated lane suggestion")
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "get_by_id").
			Time(time.Now()).
			Wrapf(err, "get dedicated lane suggestion")
	}

	return entity, nil
}

func (dlsr *dedicatedLaneSuggestionRepository) Create(
	ctx context.Context,
	suggestion *dedicatedlane.DedicatedLaneSuggestion,
) (*dedicatedlane.DedicatedLaneSuggestion, error) {
	dba, err := dlsr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlsr.l.With().
		Str("op", "create").
		Str("suggestionId", suggestion.ID.String()).
		Logger()

	if _, err = dba.NewInsert().Model(suggestion).Returning("*").Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to create dedicated lane suggestion")
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "create dedicated lane suggestion")
	}

	return suggestion, nil
}

func (dlsr *dedicatedLaneSuggestionRepository) Update(
	ctx context.Context,
	suggestion *dedicatedlane.DedicatedLaneSuggestion,
) (*dedicatedlane.DedicatedLaneSuggestion, error) {
	dba, err := dlsr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "update").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlsr.l.With().
		Str("op", "update").
		Str("suggestionId", suggestion.ID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := suggestion.Version

		suggestion.Version++

		results, rErr := tx.NewUpdate().
			Model(suggestion).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("dls.id = ?", suggestion.ID).
					Where("dls.organization_id = ?", suggestion.OrganizationID).
					Where("dls.business_unit_id = ?", suggestion.BusinessUnitID).
					Where("dls.version = ?", ov)
			}).
			OmitZero().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update dedicated lane suggestion")
			return eris.Wrap(rErr, "update dedicated lane suggestion")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Dedicated Lane Suggestion (%s) has either been updated or deleted since the last request.",
					suggestion.ID,
				),
			)
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to update dedicated lane suggestion")
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "update").
			Time(time.Now()).
			Wrapf(err, "update dedicated lane suggestion")
	}

	return suggestion, nil
}

func (dlsr *dedicatedLaneSuggestionRepository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateSuggestionStatusRequest,
) (*dedicatedlane.DedicatedLaneSuggestion, error) {
	dba, err := dlsr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "update_status").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlsr.l.With().
		Str("op", "update_status").
		Str("suggestionId", req.SuggestionID.String()).
		Str("status", string(req.Status)).
		Logger()

	suggestion := &dedicatedlane.DedicatedLaneSuggestion{}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// First get the current suggestion
		if err = tx.NewSelect().Model(suggestion).
			Where("dls.id = ?", req.SuggestionID).
			Scan(c); err != nil {
			return eris.Wrap(err, "get suggestion for status update")
		}

		// Update the status and related fields
		suggestion.Status = req.Status
		suggestion.ProcessedByID = req.ProcessedByID
		suggestion.ProcessedAt = req.ProcessedAt
		suggestion.Version++

		if req.RejectReason != nil {
			if suggestion.PatternDetails == nil {
				suggestion.PatternDetails = make(map[string]any)
			}
			suggestion.PatternDetails["rejectReason"] = *req.RejectReason
			suggestion.PatternDetails["rejectedAt"] = timeutils.NowUnix()
		}

		_, rErr := tx.NewUpdate().
			Model(suggestion).
			Where("dls.id = ?", req.SuggestionID).
			OmitZero().
			Returning("*").
			Exec(c)
		if rErr != nil {
			return eris.Wrap(rErr, "update suggestion status")
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to update suggestion status")
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "update_status").
			Time(time.Now()).
			Wrapf(err, "update suggestion status")
	}

	return suggestion, nil
}

func (dlsr *dedicatedLaneSuggestionRepository) Delete(
	ctx context.Context,
	id pulid.ID,
	orgID pulid.ID,
	buID pulid.ID,
) error {
	dba, err := dlsr.db.DB(ctx)
	if err != nil {
		return oops.In("dedicated_lane_suggestion_repository").
			With("op", "delete").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlsr.l.With().
		Str("op", "delete").
		Str("suggestionId", id.String()).
		Logger()

	result, err := dba.NewDelete().
		Model((*dedicatedlane.DedicatedLaneSuggestion)(nil)).
		Where("dls.id = ?", id).
		Where("dls.organization_id = ?", orgID).
		Where("dls.business_unit_id = ?", buID).
		Exec(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to delete dedicated lane suggestion")
		return oops.In("dedicated_lane_suggestion_repository").
			With("op", "delete").
			Time(time.Now()).
			Wrapf(err, "delete dedicated lane suggestion")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return oops.In("dedicated_lane_suggestion_repository").
			With("op", "delete").
			Time(time.Now()).
			Wrapf(err, "get rows affected")
	}

	if rows == 0 {
		return oops.In("dedicated_lane_suggestion_repository").
			With("op", "delete").
			Time(time.Now()).
			Errorf("no dedicated lane suggestion found with id %s", id.String())
	}

	return nil
}

func (dlsr *dedicatedLaneSuggestionRepository) ExpireOldSuggestions(
	ctx context.Context,
	orgID pulid.ID,
	buID pulid.ID,
) (int64, error) {
	dba, err := dlsr.db.DB(ctx)
	if err != nil {
		return 0, oops.In("dedicated_lane_suggestion_repository").
			With("op", "expire_old_suggestions").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlsr.l.With().
		Str("op", "expire_old_suggestions").
		Str("orgId", orgID.String()).
		Logger()

	now := timeutils.NowUnix()

	result, err := dba.NewUpdate().
		Model((*dedicatedlane.DedicatedLaneSuggestion)(nil)).
		Set("status = ?", dedicatedlane.SuggestionStatusExpired).
		Set("updated_at = ?", now).
		Where("dls.organization_id = ?", orgID).
		Where("dls.business_unit_id = ?", buID).
		Where("dls.status = ?", dedicatedlane.SuggestionStatusPending).
		Where("dls.expires_at <= ?", now).
		Exec(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to expire old suggestions")
		return 0, oops.In("dedicated_lane_suggestion_repository").
			With("op", "expire_old_suggestions").
			Time(time.Now()).
			Wrapf(err, "expire old suggestions")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return 0, oops.In("dedicated_lane_suggestion_repository").
			With("op", "expire_old_suggestions").
			Time(time.Now()).
			Wrapf(err, "get rows affected")
	}

	log.Info().Int64("expiredCount", rowsAffected).Msg("expired old suggestions")

	return rowsAffected, nil
}

func (dlsr *dedicatedLaneSuggestionRepository) CheckForDuplicatePattern(
	ctx context.Context,
	req *repositories.FindDedicatedLaneByShipmentRequest,
) (*dedicatedlane.DedicatedLaneSuggestion, error) {
	dba, err := dlsr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "check_duplicate_pattern").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	suggestion := new(dedicatedlane.DedicatedLaneSuggestion)

	query := dba.NewSelect().Model(suggestion).
		Distinct(). // Distinct is used to ensure that we only get one suggestion back
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = sq.
				Where("dls.organization_id = ?", req.OrganizationID).
				Where("dls.business_unit_id = ?", req.BusinessUnitID).
				Where("dls.customer_id = ?", req.CustomerID).
				Where("dls.origin_location_id = ?", req.OriginLocationID).
				Where("dls.destination_location_id = ?", req.DestinationLocationID).
				Where("dls.status = ?", dedicatedlane.SuggestionStatusPending)

			// Check equipment type IDs to match the unique constraint
			if req.ServiceTypeID != nil && !req.ServiceTypeID.IsNil() {
				sq = sq.Where("dls.service_type_id = ?", *req.ServiceTypeID)
			} else {
				sq = sq.Where("dls.service_type_id IS NULL")
			}

			if req.ShipmentTypeID != nil && !req.ShipmentTypeID.IsNil() {
				sq = sq.Where("dls.shipment_type_id = ?", *req.ShipmentTypeID)
			} else {
				sq = sq.Where("dls.shipment_type_id IS NULL")
			}

			if req.TrailerTypeID != nil && !req.TrailerTypeID.IsNil() {
				sq = sq.Where("dls.trailer_type_id = ?", *req.TrailerTypeID)
			} else {
				sq = sq.Where("dls.trailer_type_id IS NULL")
			}

			if req.TractorTypeID != nil && !req.TractorTypeID.IsNil() {
				sq = sq.Where("dls.tractor_type_id = ?", *req.TractorTypeID)
			} else {
				sq = sq.Where("dls.tractor_type_id IS NULL")
			}

			return sq
		})

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("no duplicate found for this organization")
		}

		return nil, oops.In("dedicated_lane_suggestion_repository").
			With("op", "check_duplicate_pattern").
			Time(time.Now()).
			Wrapf(err, "check duplicate pattern")
	}

	return suggestion, nil
}
