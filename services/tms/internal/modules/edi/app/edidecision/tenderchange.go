package edidecision

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func validateTenderChangeReviewer(
	change *editenderchange.TenderChange,
	tenantInfo pagination.TenantInfo,
) error {
	if change == nil || change.Recipient == nil {
		return errortypes.NewValidationError(
			"recipientId",
			errortypes.ErrRequired,
			"EDI tender change recipient is required for review",
		)
	}
	if change.Recipient.RecipientKind != editenderchange.TenderRecipientKindInternal {
		return errortypes.NewValidationError(
			"recipientKind",
			errortypes.ErrInvalidOperation,
			"Only internal tender changes can be manually reviewed",
		)
	}
	if change.Recipient.RecipientOrganizationID != tenantInfo.OrgID ||
		change.Recipient.BusinessUnitID != tenantInfo.BuID {
		return errortypes.NewValidationError(
			"tenantInfo",
			errortypes.ErrInvalidOperation,
			"Only the original internal tender recipient can review this tender change",
		)
	}
	return nil
}
