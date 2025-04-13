package shipmentvalidator

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/hazmatsegreationrulevalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection, move validator, shipment control repository, and hazmat segregation validator.
type ValidatorParams struct {
	fx.In

	DB                         db.Connection
	MoveValidator              *MoveValidator
	ShipmentControlRepo        repositories.ShipmentControlRepository
	HazmatSegregationValidator *hazmatsegreationrulevalidator.Validator
}

// Validator is a validator for shipments.
// It validates shipments, moves, and other related entities.
type Validator struct {
	db  db.Connection
	mv  *MoveValidator
	scp repositories.ShipmentControlRepository
	hsr *hazmatsegreationrulevalidator.Validator
}

// NewValidator initializes a new Validator with the provided dependencies.
//
// Parameters:
//   - p: ValidatorParams containing dependencies.
//
// Returns:
//   - *Validator: A new Validator instance.
func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:  p.DB,
		mv:  p.MoveValidator,
		scp: p.ShipmentControlRepo,
		hsr: p.HazmatSegregationValidator,
	}
}

// Validate validates a shipment.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - shp: The shipment to validate.
func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, shp *shipment.Shipment) *errors.MultiError {
	multiErr := errors.NewMultiError()

	shp.Validate(ctx, multiErr)

	sc, err := v.scp.GetByOrgID(ctx, shp.OrganizationID)
	if err != nil {
		multiErr.Add("shipmentControl", errors.ErrSystemError, err.Error())
		return multiErr
	}

	// * If the organization has duplicate BOLs checking enabled, check for duplicates
	if sc.CheckForDuplicateBOLs {
		if err = v.CheckForDuplicateBOLs(ctx, shp, multiErr); err != nil {
			multiErr.Add("duplicateBOLs", errors.ErrSystemError, err.Error())
		}
	}

	// * Validate uniqueness
	if err = v.ValidateUniqueness(ctx, valCtx, shp, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// // * Validate ID
	// v.validateID(shp, valCtx, multiErr)

	// * Validate Moves
	v.ValidateMoves(ctx, valCtx, shp, multiErr)

	// * Validate Hazmat Segregation
	if sc.CheckHazmatSegregation && shp.HasCommodities() {
		log.Info().Interface("shipment", shp).Msg("Validating hazmat segregation")
		segregationErr, _ := v.hsr.ValidateShipment(ctx, shp)
		if segregationErr != nil && segregationErr.HasErrors() {
			for _, err := range segregationErr.Errors {
				multiErr.Add(err.Field, err.Code, err.Message)
			}
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// ValidateMoves validates the moves of a shipment.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - shp: The shipment to validate.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) ValidateMoves(ctx context.Context, valCtx *validator.ValidationContext, shp *shipment.Shipment, multiErr *errors.MultiError) {
	if len(shp.Moves) == 0 {
		multiErr.Add("moves", errors.ErrInvalid, "Shipment must have at least one move")
		return
	}

	for idx, move := range shp.Moves {
		v.mv.Validate(ctx, valCtx, move, multiErr, idx)
	}
}

// ValidateUniqueness validates the uniqueness of a shipment.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - shp: The shipment to validate.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) ValidateUniqueness(ctx context.Context, valCtx *validator.ValidationContext, shp *shipment.Shipment, multiErr *errors.MultiError) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(shp.GetTableName()).
		WithTenant(shp.OrganizationID, shp.BusinessUnitID).
		WithModelName("Shipment").
		WithFieldAndTemplate("pro_number", shp.ProNumber,
			"Shipment with Pro Number ':value' already exists in the organization.",
			map[string]string{
				"value": shp.ProNumber,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", shp.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

// validateID validates the ID of a shipment.
//
// Parameters:
//   - shp: The shipment to validate.
//   - valCtx: The validation context.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) validateID(shp *shipment.Shipment, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && shp.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}

// validateTemperature validates the temperature of a shipment.
//
// Parameters:
//   - shp: The shipment to validate.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
// func (v *Validator) validateTemperature(shp *shipment.Shipment, multiErr *errors.MultiError) {
// 	if shp.TemperatureMin.Valid && shp.TemperatureMax.Valid && shp.TemperatureMin.Decimal.GreaterThan(shp.TemperatureMax.Decimal) {
// 		multiErr.Add("temperatureMin", errors.ErrInvalid, "Temperature Min must be less than Temperature Max")
// 	}
// }

// ValidateCancellation validates the cancellation of a shipment.
//
// Parameters:
//   - shp: The shipment to validate.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) ValidateCancellation(shp *shipment.Shipment) *errors.MultiError {
	multiErr := errors.NewMultiError()

	if !cancelableShipmentStatuses[shp.Status] {
		multiErr.Add(
			"__all__",
			errors.ErrInvalid,
			fmt.Sprintf("Cannot cancel shipment in status `%s`", shp.Status),
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// CheckForDuplicateBOLs checks for duplicate BOLs in the database.
//
// Parameters:
//   - ctx: The context of the request.
//   - shp: The shipment to validate.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) CheckForDuplicateBOLs(ctx context.Context, shp *shipment.Shipment, multiErr *errors.MultiError) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	// * Small struct to store the results of the query
	var duplicates []struct {
		ID        pulid.ID `bun:"id"`
		ProNumber string   `bun:"pro_number"`
	}

	// * Query to find duplicates, selecting only necessary fields for efficiency
	query := dba.NewSelect().
		Column("sp.id").
		Column("sp.pro_number").
		Model((*shipment.Shipment)(nil)).
		Where("sp.organization_id = ?", shp.OrganizationID).
		Where("sp.business_unit_id = ?", shp.BusinessUnitID).
		Where("sp.bol = ?", shp.BOL)

	// * If this is an update operation, exclude the current shipment from the check
	if shp.ID.IsNotNil() {
		query = query.Where("sp.id != ?", shp.ID)
	}

	// * Scan the results into the duplicates slice
	if err = query.Scan(ctx, &duplicates); err != nil {
		return eris.Wrapf(err, "query duplicate BOLs for BOL '%s'", shp.BOL)
	}

	// * If duplicates found, construct a meaningful error message
	if len(duplicates) > 0 {
		proNumbers := make([]string, 0, len(duplicates))
		for _, dup := range duplicates {
			proNumbers = append(proNumbers, dup.ProNumber)
		}

		errorMsg := fmt.Sprintf(
			"BOL '%s' already exists in shipment(s) with Pro Number(s): %s",
			shp.BOL,
			strings.Join(proNumbers, ", "),
		)

		multiErr.Add("bol", errors.ErrInvalid, errorMsg)
	}

	return nil
}

// ValidateCommodityAddition checks if adding a new commodity to the shipment
// would violate hazmat segregation rules
//
// Parameters:
//   - ctx: The context of the request.
//   - shp: The shipment to validate.
//   - commodityID: The ID of the commodity to add.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) ValidateCommodityAddition(ctx context.Context, shp *shipment.Shipment, commodityID pulid.ID) *errors.MultiError {
	multiErr := errors.NewMultiError()

	sc, err := v.scp.GetByOrgID(ctx, shp.OrganizationID)
	if err != nil {
		multiErr.Add("shipmentControl", errors.ErrSystemError, err.Error())
		return multiErr
	}

	// * Skip hazmat validation if not enabled
	if !sc.CheckHazmatSegregation {
		return nil
	}

	// * Validate hazmat segregation for the new commodity
	segregationErr, _ := v.hsr.ValidateShipmentCommodityAddition(ctx, shp, commodityID)
	if segregationErr != nil && segregationErr.HasErrors() {
		for _, err := range segregationErr.Errors {
			multiErr.Add(err.Field, err.Code, err.Message)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
