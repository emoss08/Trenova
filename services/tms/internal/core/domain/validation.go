package domain

import "github.com/emoss08/trenova/pkg/errortypes"

type Validatable interface {
	Validate(multiErr *errortypes.MultiError)
	GetTableName() string
}
