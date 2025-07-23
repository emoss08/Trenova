// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/common"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ShipmentTypeRepositoryParams defines dependencies required for initializing the ShipmentTypeRepository.
// This includes database connection, logger, and shipment type repository.
type ShipmentTypeRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// shipmentTypeRepository implements the ShipmentTypeRepository interface
// and provides methods to manage shipment types, including CRUD operations,
// status updates, and retrieval by ID.
type shipmentTypeRepository struct {
	*common.BaseRepository
}

// NewShipmentTypeRepository initializes a new shipment type repository with its dependencies.
//
// Parameters:
//   - p: ShipmentTypeRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.ShipmentTypeRepository: A ready-to-use shipment type repository instance.
func NewShipmentTypeRepository(p ShipmentTypeRepositoryParams) repositories.ShipmentTypeRepository {
	log := p.Logger.With().
		Str("repository", "shipmenttype").
		Logger()

	return &shipmentTypeRepository{
		BaseRepository: &common.BaseRepository{
			DB:         p.DB,
			Logger:     &log,
			TableName:  "shipment_types",
			EntityName: "Shipment Type",
		},
	}
}

// filterQuery applies filters and pagination to the shipment type query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - req: ListShipmentTypeRequest containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (str *shipmentTypeRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListShipmentTypeRequest,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: shipmenttype.ShipmentTypeQuery.Alias,
		Filter:     req.Filter,
	})

	if req.Status != "" {
		status, err := domain.StatusFromString(req.Status)
		if err != nil {
			str.Logger.Error().Err(err).Str("status", req.Status).Msg("invalid status")
			return q
		}

		q = shipmenttype.ShipmentTypeQuery.Where.StatusEQ(q, status)
	}

	if req.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			req.Filter.Query,
			(*shipmenttype.ShipmentType)(nil),
		)
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

// List retrieves shipment types based on filtering and pagination options.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: LimitOffsetQueryOptions containing filtering and pagination parameters.
//
// Returns:
//   - *ports.ListResult[*shipmenttype.ShipmentType]: List of shipment types and total count.
//   - error: If any database operation fails.
func (str *shipmentTypeRepository) List(
	ctx context.Context,
	req *repositories.ListShipmentTypeRequest,
) (*ports.ListResult[*shipmenttype.ShipmentType], error) {
	dba, log, err := str.SetupReadOnly(ctx, "List",
		"buID", req.Filter.TenantOpts.BuID.String(),
		"userID", req.Filter.TenantOpts.UserID.String(),
	)
	if err != nil {
		return nil, err
	}

	entities := make([]*shipmenttype.ShipmentType, 0)

	q := dba.NewSelect().Model(&entities)
	q = str.filterQuery(q, req)

	// Order by status and created at
	q = common.ApplyDefaultListOrdering(
		q,
		shipmenttype.ShipmentTypeQuery.Alias,
		shipmenttype.ShipmentTypeQuery.OrderBy.Default()...)

	result, err := common.ExecuteListQuery[*shipmenttype.ShipmentType](ctx, q)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan shipment types")
		return nil, err
	}

	return result, nil
}

// GetByID retrieves a shipment type by its unique ID, including optional expanded details.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: GetShipmentTypeByIDOptions containing ID and expansion preferences.
//
// Returns:
//   - *shipmenttype.ShipmentType: The retrieved shipment type entity.
//   - error: If the shipment type is not found or query fails.
func (str *shipmentTypeRepository) GetByID(
	ctx context.Context,
	opts repositories.GetShipmentTypeByIDOptions,
) (*shipmenttype.ShipmentType, error) {
	dba, log, err := str.SetupReadOnly(ctx, "GetByID",
		"shipmentTypeID", opts.ID.String(),
	)
	if err != nil {
		return nil, err
	}

	entity, err := shipmenttype.NewShipmentTypeQuery(dba).
		WhereIDEQ(opts.ID).
		WhereTenant(opts.OrgID, opts.BuID).
		One(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment type")
		return nil, common.HandleNotFoundError(err, str.EntityName)
	}

	return entity, nil
}

// Create inserts a new shipment type into the database.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - st: The shipment type entity to be created.
//
// Returns:
//   - *shipmenttype.ShipmentType: The created shipment type entity.
//   - error: If the creation fails.
func (str *shipmentTypeRepository) Create(
	ctx context.Context,
	st *shipmenttype.ShipmentType,
) (*shipmenttype.ShipmentType, error) {
	dba, log, err := str.SetupWriteOnly(ctx, "Create",
		"orgID", st.OrganizationID.String(),
		"buID", st.BusinessUnitID.String(),
	)
	if err != nil {
		return nil, err
	}

	if _, err = dba.NewInsert().Model(st).Returning("*").Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("shipmentType", st).
			Msg("failed to insert shipment type")
		return nil, err
	}

	return st, nil
}

// Update updates an existing shipment type in the database.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - st: The shipment type entity to be updated.
//
// Returns:
//   - *shipmenttype.ShipmentType: The updated shipment type entity.
//   - error: If the update fails.
func (str *shipmentTypeRepository) Update(
	ctx context.Context,
	st *shipmenttype.ShipmentType,
) (*shipmenttype.ShipmentType, error) {
	// Update needs read-write access for transactions with optimistic locking
	_, log, err := str.SetupReadWrite(ctx, "Update",
		"id", st.GetID(),
		"version", st.Version,
	)
	if err != nil {
		return nil, err
	}

	result, err := common.RunInTransactionWithResult(ctx, str.DB,
		func(c context.Context, tx bun.Tx) (*shipmenttype.ShipmentType, error) {
			if err := common.OptimisticUpdateWithAlias(c, tx, st, str.EntityName, shipmenttype.ShipmentTypeQuery.Alias); err != nil {
				return nil, err
			}
			return st, nil
		})

	if err != nil {
		log.Error().Err(err).Interface("shipmentType", st).Msg("failed to update shipment type")
		return nil, common.WrapDatabaseError(err, "update shipment type")
	}

	return result, nil
}
