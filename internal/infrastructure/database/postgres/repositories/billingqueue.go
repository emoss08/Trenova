package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// BillingQueueRepositoryParams defines dependencies required for initializing the BillingQueueRepository.
// This includes database connection and logger.
type BillingQueueRepositoryParams struct {
	fx.In

	DB           db.Connection
	Logger       *logger.Logger
	ShipmentRepo repositories.ShipmentRepository
}

// billingQueueRepository implements the BillingQueueRepository interface
// and provides methods to manage billing queue data, including CRUD operations.
type billingQueueRepository struct {
	db           db.Connection
	l            *zerolog.Logger
	shipmentRepo repositories.ShipmentRepository
}

// NewBillingQueueRepository initalizes a new instance of billingQueueRepository with its dependencies.
//
// Parameters:
//   - p: BillingQueueRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.BillingQueueRepository: A ready-to-use billing queue repository instance.
func NewBillingQueueRepository(p BillingQueueRepositoryParams) repositories.BillingQueueRepository {
	log := p.Logger.With().
		Str("repository", "billing_queue").
		Logger()

	return &billingQueueRepository{
		db:           p.DB,
		l:            &log,
		shipmentRepo: p.ShipmentRepo,
	}
}

// addOptions expands the query with related entities based on BillingQueueFilterOptions.
// This allows eager loading of related data like shipment.
//
// Parameters:
//   - q: The base select query.
//   - opts: Options to determine which related data to include.
//
// Returns:
//   - *bun.SelectQuery: The updated query with the necessary relations.
func (br *billingQueueRepository) addOptions(
	q *bun.SelectQuery,
	opts *repositories.BillingQueueFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeShipmentDetails {
		q = q.Relation("Shipment")
	}

	// * Filter by status
	if opts.Status != "" {
		status, err := billingqueue.QueueStatusFromString(opts.Status)
		if err != nil {
			br.l.Error().Err(err).Msg("failed to convert status to enum")
			return q
		}
		q.Where("status = ?", status)
	}

	// * Filter by bill type
	if opts.BillType != "" {
		billType, err := billingqueue.QueueTypeFromString(opts.BillType)
		if err != nil {
			br.l.Error().Err(err).Msg("failed to convert bill type to enum")
			return q
		}
		q.Where("bill_type = ?", billType)
	}

	return q
}

// filterQuery applies filters and pagination to the billing queue query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - req: ListBillingQueueRequest containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (br *billingQueueRepository) filterQuery(
	q *bun.SelectQuery,
	opts *repositories.ListBillingQueueRequest,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "bqi",
		Filter:     opts.Filter,
	})

	q = br.addOptions(q, &opts.FilterOptions)

	return q
}

// List retrieves billing queue items based on filtering and pagination options.
// It returns a list of billing queue items along with the total count.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: ListBillingQueueRequest for filtering and pagination.
//
// Returns:
//   - *ports.ListResult[*billingqueue.QueueItem]: List of billing queue items and total count.
//   - error: If any database operation fails.
func (br *billingQueueRepository) List(
	ctx context.Context,
	req *repositories.ListBillingQueueRequest,
) (*ports.ListResult[*billingqueue.QueueItem], error) {
	dba, err := br.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("billing_queue_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := br.l.With().
		Str("method", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("userID", req.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*billingqueue.QueueItem, 0)

	q := dba.NewSelect().Model(&entities)
	q = br.filterQuery(q, req)

	total, err := q.ScanAndCount(ctx, &entities)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan and count billing queue items")
		return nil, oops.
			In("billing_queue_repository").
			Tags("crud", "list").
			Time(time.Now()).
			Wrapf(err, "scan and count billing queue items")
	}

	return &ports.ListResult[*billingqueue.QueueItem]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves a billing queue item by its unique ID.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: GetBillingQueueItemRequest containing ID and expansion preferences.
//
// Returns:
//   - *billingqueue.QueueItem: The retrieved billing queue item entity.
//   - error: If the billing queue item is not found or query fails.
func (br *billingQueueRepository) GetByID(
	ctx context.Context,
	req *repositories.GetBillingQueueItemRequest,
) (*billingqueue.QueueItem, error) {
	dba, err := br.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("billing_queue_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := br.l.With().
		Str("operation", "GetByID").
		Str("billingQueueItemID", req.BillingQueueItemID.String()).
		Logger()

	entity := new(billingqueue.QueueItem)

	q := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("bqi.id = ?", req.BillingQueueItemID).
				Where("bqi.organization_id = ?", req.OrgID).
				Where("bqi.business_unit_id = ?", req.BuID)
		})

	q = br.addOptions(q, &req.FilterOptions)

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get billing queue item")
			return nil, errors.NewNotFoundError("Billing Queue Item not within your organization")
		}

		log.Error().Err(err).Msg("failed to get billing queue item")
		return nil, oops.
			In("billing_queue_repository").
			Tags("crud", "get").
			Time(time.Now()).
			Wrapf(err, "get billing queue item by id")
	}

	return entity, nil
}

// Create inserts a new billing queue item into the database.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - qi: The billing queue item entity to be created.
//
// Returns:
//   - *billingqueue.QueueItem: The created billing queue item.
//   - error: If insertion or related operations fail.
func (br *billingQueueRepository) Create(
	ctx context.Context,
	qi *billingqueue.QueueItem,
) (*billingqueue.QueueItem, error) {
	dba, err := br.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("billing_queue_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := br.l.With().
		Str("operation", "Create").
		Str("orgID", qi.OrganizationID.String()).
		Str("buID", qi.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(qi).Exec(c); iErr != nil {
			log.Error().Err(iErr).Msg("failed to insert billing queue item")
			return iErr
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create billing queue item")
		return nil, oops.
			In("billing_queue_repository").
			Tags("crud", "create").
			Time(time.Now()).
			Wrapf(err, "create billing queue item")
	}

	return qi, nil
}

// Update modifies an existing billing queue item.
// It uses optimistic locking to avoid concurrent modification issues.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - qi: The billing queue item entity with updated fields.
//
// Returns:
//   - *billingqueue.QueueItem: The updated billing queue item.
//   - error: If the update fails or version conflicts occur.
func (br *billingQueueRepository) Update(
	ctx context.Context,
	qi *billingqueue.QueueItem,
) (*billingqueue.QueueItem, error) {
	dba, err := br.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("billing_queue_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := br.l.With().
		Str("operation", "Update").
		Str("orgID", qi.OrganizationID.String()).
		Str("buID", qi.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := qi.Version

		qi.Version++

		results, rErr := tx.NewUpdate().
			Model(qi).
			WherePK().
			OmitZero().
			Where("bqi.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update billing queue item")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Billing Queue Item (%s) has either been updated or deleted since the last request.",
					qi.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update billing queue item")
		return nil, oops.
			In("billing_queue_repository").
			Tags("crud", "update").
			Time(time.Now()).
			Wrapf(err, "update billing queue item")
	}

	return qi, nil
}

// BulkTransfer transfers all shipments that are ready to be billed to the billing queue.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: BulkTransferRequest containing organization and business unit IDs.
//
// Returns:
//   - error: If any database operation fails.
func (br *billingQueueRepository) BulkTransfer(
	ctx context.Context,
	_ *repositories.BulkTransferRequest,
) error {
	dba, err := br.db.DB(ctx)
	if err != nil {
		return oops.
			In("billing_queue_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	shipments, err := br.shipmentRepo.List(ctx, &repositories.ListShipmentOptions{
		ShipmentOptions: repositories.ShipmentOptions{Status: string(shipment.StatusReadyToBill)},
	})
	if err != nil {
		return oops.
			In("billing_queue_repository").
			Tags("crud", "bulk_transfer").
			Time(time.Now()).
			Wrapf(err, "list shipments")
	}

	billingQueueItems := make([]*billingqueue.QueueItem, 0, shipments.Total)
	for _, shipment := range shipments.Items {
		billingQueueItems = append(billingQueueItems, &billingqueue.QueueItem{
			OrganizationID: shipment.OrganizationID,
			BusinessUnitID: shipment.BusinessUnitID,
			ShipmentID:     shipment.ID,
		})
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		_, iErr := tx.NewInsert().Model(&billingQueueItems).Exec(c)
		if iErr != nil {
			return iErr
		}

		return nil
	})
	if err != nil {
		return oops.
			In("billing_queue_repository").
			Tags("crud", "transfer_shipments").
			Time(time.Now()).
			Wrapf(err, "transfer shipments")
	}

	return nil
}
