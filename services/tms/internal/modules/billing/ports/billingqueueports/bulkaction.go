package billingqueueports

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrBulkActionNotApplicable = errors.New("the action does not apply to this item")

type ApplyBulkActionRequest struct {
	ItemID     pulid.ID
	TenantInfo pagination.TenantInfo
	Action     billingapprovalrun.RunAction
	Params     billingapprovalrun.RunParams
}
