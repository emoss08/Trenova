package reportservice

import (
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type GenerateReportRequest struct {
	OrganizationID pulid.ID                `json:"organizationId" form:"organizationId"`
	BusinessUnitID pulid.ID                `json:"businessUnitId" form:"businessUnitId"`
	UserID         pulid.ID                `json:"userId"         form:"userId"`
	ResourceType   string                  `json:"resourceType"   form:"resourceType"`
	Name           string                  `json:"name"           form:"name"`
	Format         report.Format           `json:"format"         form:"format"`
	DeliveryMethod report.DeliveryMethod   `json:"deliveryMethod" form:"deliveryMethod"`
	FilterState    pagination.QueryOptions `json:"filterState"    form:"filterState"`
}

func (r *GenerateReportRequest) Validate() *errortypes.MultiError {
	me := errortypes.NewMultiError()
	err := validation.ValidateStruct(
		r,
		validation.Field(
			&r.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
		validation.Field(
			&r.UserID,
			validation.Required.Error("User ID is required"),
		),
		validation.Field(
			&r.Format,
			validation.Required.Error("Format is required"),
		),
		validation.Field(
			&r.DeliveryMethod,
			validation.Required.Error("Delivery Method is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, me)
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}
