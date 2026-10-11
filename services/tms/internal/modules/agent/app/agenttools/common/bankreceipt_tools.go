package common

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxPaymentApplications = 50

const ParamApplications = "applications"

const ParamAmount = "amount"

const ParamShortPayAmount = "shortPayAmount"

func ReadPaymentApplications(
	params map[string]any,
	key string,
) ([]*customerpaymentports.CustomerPaymentApplicationInput, int64, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, 0, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, 0, fmt.Errorf("%s must be a list of {invoiceId, amount}", key)
	}
	if len(entries) > MaxPaymentApplications {
		return nil, 0, fmt.Errorf(
			"%s lists %d invoices; at most %d can be applied in one payment",
			key,
			len(entries),
			MaxPaymentApplications,
		)
	}

	applications := make([]*customerpaymentports.CustomerPaymentApplicationInput, 0, len(entries))
	seen := make(map[pulid.ID]struct{}, len(entries))
	total := int64(0)
	for idx, entry := range entries {
		fields, ok := entry.(map[string]any)
		if !ok {
			return nil, 0, fmt.Errorf(
				"%s[%d] must be an object with invoiceId and amount",
				key,
				idx,
			)
		}
		invoiceID, err := RequirePulid(fields, ParamInvoiceID)
		if err != nil {
			return nil, 0, fmt.Errorf("%s[%d]: %w", key, idx, err)
		}
		if _, duplicate := seen[invoiceID]; duplicate {
			return nil, 0, fmt.Errorf("%s[%d] repeats invoice %s", key, idx, invoiceID)
		}
		seen[invoiceID] = struct{}{}
		applied, err := RequireMoney(fields, ParamAmount)
		if err != nil {
			return nil, 0, fmt.Errorf("%s[%d]: %w", key, idx, err)
		}
		shortPay, _, err := OptionalMoney(fields, ParamShortPayAmount)
		if err != nil {
			return nil, 0, fmt.Errorf("%s[%d]: %w", key, idx, err)
		}
		total += applied
		applications = append(applications, &customerpaymentports.CustomerPaymentApplicationInput{
			InvoiceID:           invoiceID,
			AppliedAmountMinor:  applied,
			ShortPayAmountMinor: shortPay,
		})
	}

	return applications, total, nil
}
