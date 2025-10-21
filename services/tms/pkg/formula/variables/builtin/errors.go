package builtin

import "errors"

var (
	ErrTempDiffMustBeNumber     = errors.New("temperature differential must be a number")
	ErrTempDiffCannotBeNegative = errors.New("temperature differential cannot be negative")
	ErrEntityNotShipment        = errors.New("entity is not a shipment")
)
