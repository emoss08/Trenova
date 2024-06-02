package queries

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/rate"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

type RateQueryService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

func NewRateQueryService(c *ent.Client, l *zerolog.Logger) *RateQueryService {
	return &RateQueryService{
		Client: c,
		Logger: l,
	}
}

type GetRatesParams struct {
	Limit    int
	Offset   int
	OrgID    uuid.UUID
	BuID     uuid.UUID
	Statuses []rate.Status
}

// GetRates retrieves a list of rates for a given organization and business unit.
// It returns a slice of Rate entities, the total number of rate records, and an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - params: GetRatesParams struct containing the parameters for the query.
//
// Returns:
//   - []*ent.Rate: A slice of Rate entities.
//   - int: The total number of rate records.
//   - error: An error object that indicates why the retrieval failed, nil if no error occurred.
func (rq *RateQueryService) GetRates(ctx context.Context, params GetRatesParams) ([]*ent.Rate, int, error) {
	count, err := rq.Client.Rate.Query().Where(
		rate.HasOrganizationWith(
			organization.IDEQ(params.OrgID),
			organization.BusinessUnitIDEQ(params.BuID),
		),
		rate.StatusIn(params.Statuses...),
	).Count(ctx)
	if err != nil {
		rq.Logger.Err(err).Msg("RateQueryService: Error getting rate count")
		return nil, 0, err
	}

	entities, err := rq.Client.Rate.Query().
		Limit(params.Limit).
		Offset(params.Offset).
		WithCustomer().
		WithCommodity().
		WithApprovedBy().
		Where(
			rate.HasOrganizationWith(
				organization.IDEQ(params.OrgID),
				organization.BusinessUnitIDEQ(params.BuID),
			),
			rate.StatusIn(params.Statuses...),
		).All(ctx)
	if err != nil {
		rq.Logger.Err(err).Msg("RateQueryService: Error getting rates")
		return nil, 0, err
	}

	return entities, count, nil
}

// CreateRateEntity creates a Rate entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *ent.Rate: The Rate request containing the details of the Rate to be created.
//
// Returns:
//   - *ent.Rate: A pointer to the newly created Rate entity.
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
func (rq *RateQueryService) CreateRateEntity(ctx context.Context, tx *ent.Tx, entity *ent.Rate) (*ent.Rate, error) {
	createdEntity, err := tx.Rate.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCustomerID(entity.CustomerID).
		SetEffectiveDate(entity.EffectiveDate).
		SetExpirationDate(entity.ExpirationDate).
		SetNillableCommodityID(entity.CommodityID).
		SetNillableShipmentTypeID(entity.ShipmentTypeID).
		SetNillableOriginLocationID(entity.OriginLocationID).
		SetNillableDestinationLocationID(entity.DestinationLocationID).
		SetRatingMethod(entity.RatingMethod).
		SetRateAmount(entity.RateAmount).
		SetComment(entity.Comment).
		SetNillableApprovedByID(entity.ApprovedByID).
		SetApprovedDate(entity.ApprovedDate).
		SetMaximumCharge(entity.MaximumCharge).
		SetMinimumCharge(entity.MinimumCharge).
		Save(ctx)
	if err != nil {
		rq.Logger.Err(err).Msg("RateQueryService: Error creating rate entity")
		return nil, err
	}

	return createdEntity, nil
}

// UpdateRateEntity updates a Rate entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *ent.Rate: The Rate update request containing the details of the Rate to be updated.
//
// Returns:
//   - *ent.Rate: A pointer to the updated Rate entity.
//   - error: An error object that indicates why the update failed, nil if no error occurred.
func (rq *RateQueryService) UpdateRateEntity(ctx context.Context, tx *ent.Tx, entity *ent.Rate) (*ent.Rate, error) {
	current, err := tx.Rate.Get(ctx, entity.ID)
	if err != nil {
		rq.Logger.Err(err).Msg("RateQueryService: Error getting rate entity")
		return nil, err
	}
	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"rateNumber")
	}

	updateOp := tx.Rate.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetStatus(entity.Status).
		SetCustomerID(entity.CustomerID).
		SetEffectiveDate(entity.EffectiveDate).
		SetExpirationDate(entity.ExpirationDate).
		SetNillableCommodityID(entity.CommodityID).
		SetNillableShipmentTypeID(entity.ShipmentTypeID).
		SetNillableOriginLocationID(entity.OriginLocationID).
		SetNillableDestinationLocationID(entity.DestinationLocationID).
		SetRatingMethod(entity.RatingMethod).
		SetRateAmount(entity.RateAmount).
		SetComment(entity.Comment).
		SetNillableApprovedByID(entity.ApprovedByID).
		SetApprovedDate(entity.ApprovedDate).
		SetMaximumCharge(entity.MaximumCharge).
		SetMinimumCharge(entity.MinimumCharge).
		SetVersion(entity.Version + 1) // Increment the version

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		rq.Logger.Err(err).Msg("RateQueryService: Error updating rate entity")
		return nil, err
	}

	return updatedEntity, nil
}

// GetsRatesNearExpiration retrieves a list of rates that are near expiration for a given organization and business unit.
// It returns a slice of Rate entities, the total number of rate records, and an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - limit int: The maximum number of records to return.
//   - offset int: The number of records to skip before starting to return records.
//   - orgID uuid.UUID: The identifier of the organization.
//   - buID uuid.UUID: The identifier of the business unit.
//
// Returns:
//   - []*ent.Rate: A slice of Rate entities.
//   - int: The total number of rate records.
//   - error: An error object that indicates why the retrieval failed, nil if no error occurred.
func (rq *RateQueryService) GetsRatesNearExpiration(ctx context.Context, orgID, buID uuid.UUID) ([]*ent.Rate, int, error) {
	now := &pgtype.Date{Time: time.Now()}

	count, err := rq.Client.Rate.Query().Where(
		rate.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
		rate.ExpirationDateLTE(now),
	).Count(ctx)
	if err != nil {
		rq.Logger.Err(err).Msg("RateQueryService: Error getting rate count")
		return nil, 0, err
	}

	entities, err := rq.Client.Rate.Query().
		WithCustomer().
		WithCommodity().
		WithApprovedBy().
		Where(
			rate.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
			rate.ExpirationDateLTE(now),
		).All(ctx)
	if err != nil {
		rq.Logger.Err(err).Msg("RateQueryService: Error getting rates")
		return nil, 0, err
	}

	return entities, count, nil
}
