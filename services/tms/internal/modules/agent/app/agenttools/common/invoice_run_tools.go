package common

import (
	"time"

	"github.com/emoss08/trenova/pkg/toolschema"
)

const ParamCustomerID = "customerId"

const ParamCustomerIDs = "customerIds"

const ParamPeriodStart = "periodStart"

const ParamPeriodEnd = "periodEnd"

const ParamInvoiceDate = "invoiceDate"

const ParamItemID = "itemId"

const DayLayout = "2006-01-02"

func RequireUTCDay(params map[string]any, key string) (time.Time, error) {
	raw, err := RequireString(params, key)
	if err != nil {
		return time.Time{}, err
	}
	day, err := time.Parse(DayLayout, raw)
	if err != nil {
		return time.Time{}, DateRefusal(key, raw, toolschema.FormatDate)
	}

	return day, nil
}
