package shipmentvalidator

import (
	"context"

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

	DB db.Connection
}

type Validator struct {
	db db.Connection
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db: p.DB,
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

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
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

	// Validate Ready To Bill
	v.validateReadyToBill(shp, multiErr)

	// Validate Temperature
	v.validateTemperature(shp, multiErr)

	return nil
}

func (v *Validator) validateID(shp *shipment.Shipment, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && shp.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}

func (v *Validator) validateReadyToBill(shp *shipment.Shipment, multiErr *errors.MultiError) {
	// if the shipment is ready to bill, then the status must be "Completed"
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
