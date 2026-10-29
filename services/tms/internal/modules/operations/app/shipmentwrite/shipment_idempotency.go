package shipmentwrite

import (
	"github.com/emoss08/trenova/pkg/dberror"
)

const idempotencyKeyConstraint = "uq_shipments_idempotency_key"

func IsIdempotencyKeyConflict(err error) bool {
	return dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == idempotencyKeyConstraint
}
