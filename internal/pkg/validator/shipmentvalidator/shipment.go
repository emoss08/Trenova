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
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB                  db.Connection
	MoveValidator       *MoveValidator
	ShipmentControlRepo repositories.ShipmentControlRepository
}

type Validator struct {
	db  db.Connection
	mv  *MoveValidator
	scp repositories.ShipmentControlRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:  p.DB,
		mv:  p.MoveValidator,
		scp: p.ShipmentControlRepo,
	}
}

func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, shp *shipment.Shipment) *errors.MultiError {
	multiErr := errors.NewMultiError()

	shp.Validate(ctx, multiErr)

	sc, err := v.scp.GetByOrgID(ctx, shp.OrganizationID)
	if err != nil {
		multiErr.Add("shipmentControl", errors.ErrSystemError, err.Error())
		return multiErr
	}

	// If the organization has duplicate BOLs checking enabled, check for duplicates
	if sc.CheckForDuplicateBOLs {
		if err := v.checkForDuplicateBOLs(ctx, shp, multiErr); err != nil {
			multiErr.Add("duplicateBOLs", errors.ErrSystemError, err.Error())
		}
	}

	// Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, shp, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(shp, valCtx, multiErr)

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

func (v *Validator) validateTemperature(shp *shipment.Shipment, multiErr *errors.MultiError) {
	if shp.TemperatureMin.Valid && shp.TemperatureMax.Valid && shp.TemperatureMin.Decimal.GreaterThan(shp.TemperatureMax.Decimal) {
		multiErr.Add("temperatureMin", errors.ErrInvalid, "Temperature Min must be less than Temperature Max")
	}
}

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

func (v *Validator) checkForDuplicateBOLs(ctx context.Context, shp *shipment.Shipment, multiErr *errors.MultiError) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	// Query to find duplicates, selecting only necessary fields for efficiency
	var duplicates []struct {
		ID        pulid.ID `bun:"id"`
		ProNumber string   `bun:"pro_number"`
	}

	query := dba.NewSelect().
		Column("sp.id").
		Column("sp.pro_number").
		Model((*shipment.Shipment)(nil)).
		Where("sp.organization_id = ?", shp.OrganizationID).
		Where("sp.business_unit_id = ?", shp.BusinessUnitID).
		Where("sp.bol = ?", shp.BOL)

	// If this is an update operation, exclude the current shipment from the check
	if shp.ID.IsNotNil() {
		query = query.Where("sp.id != ?", shp.ID)
	}

	if err := query.Scan(ctx, &duplicates); err != nil {
		return eris.Wrapf(err, "query duplicate BOLs for BOL '%s'", shp.BOL)
	}

	// If duplicates found, construct a meaningful error message
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
