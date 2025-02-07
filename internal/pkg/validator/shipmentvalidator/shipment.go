package shipmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/rotisserie/eris"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB            db.Connection
	MoveValidator *MoveValidator
}

type Validator struct {
	db db.Connection
	mv *MoveValidator
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db: p.DB,
		mv: p.MoveValidator,
	}
}

func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, shp *shipment.Shipment) *errors.MultiError {
	multiErr := errors.NewMultiError()

	shp.Validate(ctx, multiErr)

	// Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, shp, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(shp, valCtx, multiErr)

	// Validate Ready To Bill
	v.validateReadyToBill(shp, multiErr)

	// Validate Billing Flags
	v.validateBillingFlags(shp, multiErr)

	// Validate Temperature
	v.validateTemperature(shp, multiErr)

	// Validate Moves
	v.ValidateMoves(ctx, valCtx, shp, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateMoves(ctx context.Context, valCtx *validator.ValidationContext, shp *shipment.Shipment, multiErr *errors.MultiError) {
	if len(shp.Moves) == 0 {
		multiErr.Add("moves", errors.ErrInvalid, "Shipment must have at least one move")
		return
	}

	for idx, move := range shp.Moves {
		v.mv.Validate(ctx, valCtx, move, multiErr, idx)
	}
}

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

func (v *Validator) validateID(shp *shipment.Shipment, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && shp.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}

func (v *Validator) validateReadyToBill(shp *shipment.Shipment, multiErr *errors.MultiError) {
	// If the shipment is ready to bill, then the status must be "Completed"
	// ! This will change when we have shipment controls
	// ! That will determine if the organization allows ready to bill to be set
	// ! Whether the shipment is completed or not.
	if shp.ReadyToBill && shp.Status != shipment.StatusCompleted {
		multiErr.Add("readyToBill", errors.ErrInvalid, "Shipment must be completed to be marked as ready to bill")
	}
}

func (v *Validator) validateTemperature(shp *shipment.Shipment, multiErr *errors.MultiError) {
	if shp.TemperatureMin.Valid && shp.TemperatureMax.Valid && shp.TemperatureMin.Decimal.GreaterThan(shp.TemperatureMax.Decimal) {
		multiErr.Add("temperatureMin", errors.ErrInvalid, "Temperature Min must be less than Temperature Max")
	}
}

// validateBillingFlags performs comprehensive validation of billing-related fields and flags
// to ensure proper billing state transitions and data consistency.
func (v *Validator) validateBillingFlags(shp *shipment.Shipment, multiErr *errors.MultiError) { //nolint: gocognit,cyclop,funlen // validation
	// --------------------------------------
	// 1. Ready to Bill State Validation
	// Ensures that if a shipment is not marked as ready to bill,
	// no subsequent billing states or dates can be set.
	// This enforces the proper progression of the billing workflow.
	// --------------------------------------
	if !shp.ReadyToBill { //nolint: nestif // It is what it is
		if shp.ReadyToBillDate != nil {
			multiErr.Add(
				"readyToBillDate",
				errors.ErrInvalid,
				"Ready to bill date cannot be set when shipment is not ready to bill",
			)
		}
		if shp.SentToBilling {
			multiErr.Add(
				"sentToBilling",
				errors.ErrInvalid,
				"Cannot be sent to billing when shipment is not ready to bill",
			)
		}
		if shp.SentToBillingDate != nil {
			multiErr.Add(
				"sentToBillingDate",
				errors.ErrInvalid,
				"Sent to billing date cannot be set when shipment is not ready to bill",
			)
		}
		if shp.Billed {
			multiErr.Add(
				"billed",
				errors.ErrInvalid,
				"Cannot be marked as billed when shipment is not ready to bill",
			)
		}
		if shp.BillDate != nil {
			multiErr.Add(
				"billDate",
				errors.ErrInvalid,
				"Bill date cannot be set when shipment is not ready to bill",
			)
		}
	}

	// --------------------------------------
	// 2. Sent to Billing State Validation
	// Validates that if a shipment is not marked as sent to billing,
	// no billing completion states or dates can be set.
	// This prevents skipping steps in the billing process.
	// --------------------------------------
	if !shp.SentToBilling {
		if shp.SentToBillingDate != nil {
			multiErr.Add(
				"sentToBillingDate",
				errors.ErrInvalid,
				"Sent to billing date cannot be set when not sent to billing",
			)
		}
		if shp.Billed {
			multiErr.Add(
				"billed",
				errors.ErrInvalid,
				"Cannot be marked as billed when not sent to billing",
			)
		}
		if shp.BillDate != nil {
			multiErr.Add(
				"billDate",
				errors.ErrInvalid,
				"Bill date cannot be set when not sent to billing",
			)
		}
	}

	// --------------------------------------
	// 3. Billed State Validation
	// Ensures that the bill date can only be set when the shipment
	// is marked as billed, preventing inconsistent billing states.
	// --------------------------------------
	if !shp.Billed && shp.BillDate != nil {
		multiErr.Add(
			"billDate",
			errors.ErrInvalid,
			"Bill date cannot be set when not billed",
		)
	}

	// --------------------------------------
	// 4. Date Sequence Validation
	// Validates that all billing-related dates follow the correct
	// chronological order. This ensures a logical progression of
	// the billing process and prevents date inconsistencies.
	// --------------------------------------
	if shp.ReadyToBillDate != nil && shp.SentToBillingDate != nil {
		if *shp.SentToBillingDate < *shp.ReadyToBillDate {
			multiErr.Add(
				"sentToBillingDate",
				errors.ErrInvalid,
				"Sent to billing date cannot be before ready to bill date",
			)
		}
	}

	if shp.SentToBillingDate != nil && shp.BillDate != nil {
		if *shp.BillDate < *shp.SentToBillingDate {
			multiErr.Add(
				"billDate",
				errors.ErrInvalid,
				"Bill date cannot be before sent to billing date",
			)
		}
	}

	// --------------------------------------
	// 5. Charge Amount Validation
	// Validates billing amounts when a shipment is marked as billed.
	// Ensures all required charges are present and properly calculated,
	// maintaining financial accuracy in the system.
	// --------------------------------------
	if shp.Billed {
		if !shp.FreightChargeAmount.Valid || shp.FreightChargeAmount.Decimal.IsZero() {
			multiErr.Add(
				"freightChargeAmount",
				errors.ErrRequired,
				"Freight charge amount is required when shipment is billed",
			)
		}

		// Validate that total charge equals the sum of freight and other charges
		if shp.FreightChargeAmount.Valid && shp.OtherChargeAmount.Valid {
			expectedTotal := shp.FreightChargeAmount.Decimal.Add(shp.OtherChargeAmount.Decimal)
			if !shp.TotalChargeAmount.Decimal.Equal(expectedTotal) {
				multiErr.Add(
					"totalChargeAmount",
					errors.ErrInvalid,
					"Total charge amount must equal freight charge plus other charges",
				)
			}
		}
	}

	// --------------------------------------
	// 7. Delivery Verification
	// Ensures that a shipment cannot be marked as ready for billing
	// until it has been delivered. This prevents premature billing
	// and ensures service completion before billing processes begin.
	// --------------------------------------
	if shp.ReadyToBill && shp.ActualDeliveryDate == nil {
		multiErr.Add(
			"actualDeliveryDate",
			errors.ErrInvalid,
			"Actual delivery date is required to mark shipment as ready to bill",
		)
	}
}

func (v *Validator) ValidateCancellation(shp *shipment.Shipment) *errors.MultiError {
	multiErr := errors.NewMultiError()

	if !cancelableShipmentStatuses[shp.Status] {
		multiErr.Add(
			"status",

			errors.ErrInvalid,
			fmt.Sprintf("Cannot cancel shipment in status `%s`", shp.Status),
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
