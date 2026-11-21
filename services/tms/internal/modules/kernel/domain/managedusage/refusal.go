package managedusage

import (
	"strconv"
	"strings"

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

var storageCeilingRefusal = errortypes.BaseError{
	Message: "Your organization's document storage is full at {0}, the most your plan can hold. Delete documents you no longer need, or move to a larger plan from your Trenova account, to keep uploading.",
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
	if meter == platformcatalog.MeterDocumentStorageBytes {
		err.Message = storageCeilingRefusal.Message
		err.Args = []any{DecimalBytes(ceiling)}
		return err
	}
	err.Message = ceilingRefusal.Message
	err.Args = []any{ceiling}
	return err
}

func DecimalBytes(n int64) string {
	units := []string{"bytes", "KB", "MB", "GB", "TB", "PB"}
	value, unit := float64(n), 0
	for value >= 1000 && unit < len(units)-1 {
		value /= 1000
		unit++
	}
	if unit == 0 {
		return strconv.FormatInt(n, 10) + " bytes"
	}
	text := strconv.FormatFloat(value, 'f', 1, 64)
	text = strings.TrimSuffix(text, ".0")
	return text + " " + units[unit]
}

func NearStorageAllowance(m Meter) bool {
	return m.Allowance > 0 && float64(m.Quantity) >= StorageWarnRatio*float64(m.Allowance)
}
