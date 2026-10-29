package shipmentwrite

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func errDuplicateExternalReference(reference, proNumber string) error {
	multiErr := errortypes.NewMultiError()
	if proNumber == "" {
		multiErr.Add(
			"externalReference",
			errortypes.ErrDuplicate,
			"Customer reference {0} is already used by another shipment for this customer",
			reference,
		)
		return multiErr
	}
	multiErr.Add(
		"externalReference",
		errortypes.ErrDuplicate,
		"Customer reference {0} is already used by shipment {1} for this customer",
		reference,
		proNumber,
	)
	return multiErr
}

func ExternalReferenceConflict(err error, entity *shipment.Shipment) error {
	if !dberror.IsUniqueConstraintViolation(err) ||
		dberror.ExtractConstraintName(err) != shipmenteffects.ExternalReferenceConstraint {
		return nil
	}
	return errDuplicateExternalReference(strings.TrimSpace(entity.ExternalReference), "")
}
