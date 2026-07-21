package resolver

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func requiredCustomerID(id string) (pulid.ID, error) {
	customerID, err := pulid.MustParse(id)
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(
			"customerId",
			errortypes.ErrInvalid,
			"Invalid customer",
		)
	}
	return customerID, nil
}

func optionalCustomerID(id *string) (pulid.ID, error) {
	if id == nil || *id == "" {
		return pulid.Nil, nil
	}
	return requiredCustomerID(*id)
}
