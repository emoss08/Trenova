package shipmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const externalReferenceConstraint = "uq_shipments_external_reference"

func (s *service) checkExternalReference(ctx context.Context, entity *shipment.Shipment) error {
	reference := strings.TrimSpace(entity.ExternalReference)
	if reference == "" {
		return nil
	}

	existing, err := s.repo.FindByExternalReference(
		ctx,
		&repositories.ExternalReferenceCheckRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			CustomerID:        entity.CustomerID,
			ExternalReference: reference,
			ShipmentID:        entity.ID,
		},
	)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	return errDuplicateExternalReference(reference, existing.ProNumber)
}

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

func externalReferenceConflict(err error, entity *shipment.Shipment) error {
	if !dberror.IsUniqueConstraintViolation(err) ||
		dberror.ExtractConstraintName(err) != externalReferenceConstraint {
		return nil
	}
	return errDuplicateExternalReference(strings.TrimSpace(entity.ExternalReference), "")
}
