package managedusage

import (
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const StorageWarnRatio = 0.8

var allowanceRefusals = map[platformcatalog.MeterKey]errortypes.BaseError{
	platformcatalog.MeterDocumentStorageBytes: {
		Message: "Your document storage is full. Turn on overage from your Trenova account to keep uploading, or delete documents you no longer need.",
	},
	platformcatalog.MeterDocumentIntelligencePages: {
		Message: "The Document Intelligence pages for this period are used up. Turn on overage from your Trenova account to keep extracting, or wait for the next period.",
	},
	platformcatalog.MeterOrganizationsTotal: {
		Message: "Your plan includes {0} organization(s), and this instance has them all. Add organizations to your plan from your Trenova account to create another.",
	},
}

var ceilingRefusal = errortypes.BaseError{
	Message: "Your plan allows at most {0} of this in a billing period, and this period has reached it. Move to a larger plan from your Trenova account to keep going.",
}

func Refusal(
	meter platformcatalog.MeterKey,
	limit, used, requested int64,
	plan string,
) *errortypes.QuotaExceededError {
	err := errortypes.NewQuotaExceededError(string(meter), limit, used, plan)
	wording, ok := allowanceRefusals[meter]
	if !ok || requested <= 0 || used+requested <= limit {
		return err
	}
	err.Message = wording.Message
	if meter == platformcatalog.MeterOrganizationsTotal {
		err.Args = []any{limit}
	}
	return err
}

func CeilingRefusal(
	meter platformcatalog.MeterKey,
	ceiling, used int64,
	plan string,
) *errortypes.QuotaExceededError {
	err := errortypes.NewQuotaExceededError(string(meter), ceiling, used, plan)
	err.Message = ceilingRefusal.Message
	err.Args = []any{ceiling}
	return err
}

func NearStorageAllowance(m Meter) bool {
	return m.Allowance > 0 && float64(m.Quantity) >= StorageWarnRatio*float64(m.Allowance)
}
