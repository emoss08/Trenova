package consolidationvalidator

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection and validation engine factory.
type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	ShipmentRepo            repositories.ShipmentRepository
	ValidationEngineFactory framework.ValidationEngineFactory
}

// Validator is a struct that contains the database connection and the validator.
// It provides methods to validate consolidation groups and other related entities.
type Validator struct {
	db           db.Connection
	shipmentRepo repositories.ShipmentRepository
	vef          framework.ValidationEngineFactory
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
		db:           p.DB,
		shipmentRepo: p.ShipmentRepo,
		vef:          p.ValidationEngineFactory,
	}
}

// Validate validates a consolidation group.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - cg: The consolidation group to validate.
//
// Returns:
//   - *errors.MultiError: A MultiError containing validation errors.
func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	cg *consolidation.ConsolidationGroup,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic validation rules (field presence, format, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(c context.Context, multiErr *errors.MultiError) error {
				cg.Validate(c, multiErr)
				return nil
			},
		),
	)

	// * Data integrity validation (uniqueness, references, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(c context.Context, multiErr *errors.MultiError) error {
				return v.ValidateUniqueness(c, valCtx, cg, multiErr)
			},
		),
	)

	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(c context.Context, multiErr *errors.MultiError) error {
				if cg.Shipments != nil {
					return v.ValidateShipments(c, valCtx, cg, multiErr)
				}
				return nil
			},
		),
	)

	return engine.Validate(ctx)
}

// ValidateUniqueness validates the uniqueness of a consolidation group.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - cg: The consolidation group to validate.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	cg *consolidation.ConsolidationGroup,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.ReadDB(ctx)
	if err != nil {
		return err
	}

	vb := queryutils.NewUniquenessValidator(cg.GetTableName()).
		WithTenant(cg.OrganizationID, cg.BusinessUnitID).
		WithModelName("ConsolidationGroup").
		WithFieldAndTemplate("consolidation_number", cg.ConsolidationNumber,
			"Consolidation group with consolidation number ':value' already exists in the organization.",
			map[string]string{
				"value": cg.ConsolidationNumber,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).WithPrimaryKey("id", cg.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) ValidateShipments(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	cg *consolidation.ConsolidationGroup,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.ReadDB(ctx)
	if err != nil {
		return err
	}

	// * Extract shipment IDs from the incoming consolidation group
	shipmentIds := make([]string, 0, len(cg.Shipments))
	for _, shipment := range cg.Shipments {
		shipmentIds = append(shipmentIds, shipment.GetID())
	}

	// * If no shipments to validate, return early
	if len(shipmentIds) == 0 {
		return nil
	}

	// * Query for all shipments that are already in consolidation groups
	shipmentsInConsolidations := make([]*shipment.Shipment, 0)
	if err := dba.NewSelect().
		Model(&shipmentsInConsolidations).
		Where("id IN (?)", bun.In(shipmentIds)).
		Where("consolidation_group_id IS NOT NULL").
		Scan(ctx); err != nil {
		return err
	}

	// * Collect problematic shipments for batch error reporting
	var alreadyConsolidatedShipments []string
	var inOtherConsolidationShipments []string

	// * For each shipment that's already in a consolidation group, check if it's allowed
	for _, existingShipment := range shipmentsInConsolidations {
		// * For create operations, no shipment should be in any consolidation group
		if valCtx.IsCreate {
			alreadyConsolidatedShipments = append(
				alreadyConsolidatedShipments,
				existingShipment.ProNumber,
			)
			continue
		}

		// * For update operations, shipments can only be in the current consolidation group
		if !valCtx.IsCreate && existingShipment.ConsolidationGroupID != nil {
			// * If the shipment is in a different consolidation group, it's not allowed
			if existingShipment.ConsolidationGroupID.String() != cg.GetID() {
				inOtherConsolidationShipments = append(
					inOtherConsolidationShipments,
					existingShipment.ProNumber,
				)
			}
		}
	}

	// * Add consolidated error messages
	if len(alreadyConsolidatedShipments) > 0 {
		multiErr.Add(
			"shipments",
			errors.ErrInvalid,
			fmt.Sprintf("Shipments %s are already in consolidation groups",
				fmt.Sprintf("%s", strings.Join(alreadyConsolidatedShipments, ", ")),
			),
		)
	}

	if len(inOtherConsolidationShipments) > 0 {
		multiErr.Add(
			"shipments",
			errors.ErrInvalid,
			fmt.Sprintf("Shipments %s are already in other consolidation groups",
				fmt.Sprintf("%s", strings.Join(inOtherConsolidationShipments, ", ")),
			),
		)
	}

	return nil
}
