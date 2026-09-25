package tablechangealertservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) CheckSubscription(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	allowed, err := s.allowlistRepo.IsTableAllowed(ctx, entity.TableName, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return err
	}
	if !allowed {
		multiErr.Add("tableName", errortypes.ErrInvalid, "Table is not eligible for change alerts")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
