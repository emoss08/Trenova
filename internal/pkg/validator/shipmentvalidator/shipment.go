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
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
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
	ValidationEngineFactory    framework.ValidationEngineFactory
}

// Validator is a validator for shipments.
// It validates shipments, moves, and other related entities.
type Validator struct {
	db  db.Connection
	mv  *MoveValidator
	scp repositories.ShipmentControlRepository
	hsr *hazmatsegreationrulevalidator.Validator
	vef framework.ValidationEngineFactory
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
		vef: p.ValidationEngineFactory,
	}
}

// Validate validates a shipment.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - shp: The shipment to validate.
func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, shp *shipment.Shipment) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// Basic validation rules (field presence, format, etc.)
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh, func(ctx context.Context, multiErr *errors.MultiError) error {
		shp.Validate(ctx, multiErr)
		return nil
	}))

	// Data integrity validation (uniqueness, references, etc.)
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageDataIntegrity, framework.ValidationPriorityHigh, func(ctx context.Context, multiErr *errors.MultiError) error {
		return v.ValidateUniqueness(ctx, valCtx, shp, multiErr)
	}))

	// Business rules validation (domain-specific rules)
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBusinessRules, framework.ValidationPriorityHigh, func(ctx context.Context, multiErr *errors.MultiError) error {
		v.ValidateMoves(ctx, shp, multiErr)
		return nil
	}))

	// Load shipment control for further validations
	var sc *shipment.ShipmentControl
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh, func(ctx context.Context, multiErr *errors.MultiError) error {
		var err error
		sc, err = v.scp.GetByOrgID(ctx, shp.OrganizationID)
		if err != nil {
			multiErr.Add("shipmentControl", errors.ErrSystemError, err.Error())
			return err
		}
		return nil
	}))

	// Check for duplicate BOLs if enabled
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageDataIntegrity, framework.ValidationPriorityMedium, func(ctx context.Context, multiErr *errors.MultiError) error {
		if sc != nil && sc.CheckForDuplicateBOLs {
			return v.CheckForDuplicateBOLs(ctx, shp, multiErr)
		}
		return nil
	}))

	// Validate hazmat segregation if enabled
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageCompliance, framework.ValidationPriorityHigh, func(ctx context.Context, multiErr *errors.MultiError) error {
		if sc != nil && sc.CheckHazmatSegregation && shp.HasCommodities() {
			log.Info().Interface("shipment", shp).Msg("Validating hazmat segregation")
			segregationErr, _ := v.hsr.ValidateShipment(ctx, shp)
			if segregationErr != nil && segregationErr.HasErrors() {
				for _, err := range segregationErr.Errors {
					multiErr.Add(err.Field, err.Code, err.Message)
				}
			}
		}
		return nil
	}))

	// Add FMCSA compliance validation rules
	AddWeightComplianceRules(engine, shp)
	AddHazmatComplianceRules(engine, shp)

	// Only add HOS compliance rules if the shipment has moves assigned
	if shp.HasMoves() {
		AddHOSComplianceRules(engine, shp)
	}

	return engine.Validate(ctx)
}

// ValidateMoves validates the moves of a shipment.
//
// Parameters:
//   - ctx: The context of the request.
//   - shp: The shipment to validate.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) ValidateMoves(ctx context.Context, shp *shipment.Shipment, multiErr *errors.MultiError) {
	if len(shp.Moves) == 0 {
		multiErr.Add("moves", errors.ErrInvalid, "Shipment must have at least one move")
		return
	}

	for idx, move := range shp.Moves {
		v.mv.Validate(ctx, move, multiErr, idx)
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

// ValidateCancellation validates the cancellation of a shipment.
//
// Parameters:
//   - shp: The shipment to validate.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) ValidateCancellation(shp *shipment.Shipment) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// Validate the shipment can be cancelled based on status
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBusinessRules, framework.ValidationPriorityHigh, func(_ context.Context, multiErr *errors.MultiError) error {
		if !cancelableShipmentStatuses[shp.Status] {
			multiErr.Add(
				"__all__",
				errors.ErrInvalid,
				fmt.Sprintf("Cannot cancel shipment in status `%s`", shp.Status),
			)
		}
		return nil
	}))

	// Use background context since this validation doesn't need any specific context
	return engine.Validate(context.Background())
}

// CheckForDuplicateBOLs checks for duplicate BOLs in the database.
//
// Parameters:
//   - ctx: The context of the request.
//   - shp: The shipment to validate.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - error: An error if database operations fail.
func (v *Validator) CheckForDuplicateBOLs(ctx context.Context, shp *shipment.Shipment, multiErr *errors.MultiError) error {
	// Skip validation if BOL is empty
	if shp.BOL == "" {
		return nil
	}

	dba, err := v.db.DB(ctx)
	if err != nil {
		multiErr.Add("database", errors.ErrSystemError, "Failed to connect to database")
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
		multiErr.Add("database", errors.ErrSystemError, "Failed to query database")
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

		multiErr.Add("bol", errors.ErrDuplicate, errorMsg)
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
	engine := v.vef.CreateEngine()

	// Load shipment control for further validations
	var sc *shipment.ShipmentControl
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh, func(ctx context.Context, multiErr *errors.MultiError) error {
		var err error
		sc, err = v.scp.GetByOrgID(ctx, shp.OrganizationID)
		if err != nil {
			multiErr.Add("shipmentControl", errors.ErrSystemError, err.Error())
			return err
		}
		return nil
	}))

	// Validate hazmat segregation if enabled
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageCompliance, framework.ValidationPriorityHigh, func(ctx context.Context, multiErr *errors.MultiError) error {
		// Skip hazmat validation if not enabled
		if sc == nil || !sc.CheckHazmatSegregation {
			return nil
		}

		// Validate hazmat segregation for the new commodity
		segregationErr, _ := v.hsr.ValidateShipmentCommodityAddition(ctx, shp, commodityID)
		if segregationErr != nil && segregationErr.HasErrors() {
			for _, err := range segregationErr.Errors {
				multiErr.Add(err.Field, err.Code, err.Message)
			}
		}
		return nil
	}))

	return engine.Validate(ctx)
}
