package carriersettlementservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ExportBatchRemittanceCSV(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
) (string, error) {
	batch, err := s.batchRepo.GetByID(ctx, repositories.GetCarrierSettlementBatchByIDRequest{
		ID:                 batchID,
		TenantInfo:         tenantInfo,
		IncludeSettlements: true,
	})
	if err != nil {
		return "", err
	}
	if len(batch.Settlements) == 0 {
		return "", errortypes.NewValidationError(
			"batchId",
			errortypes.ErrInvalid,
			"Batch contains no settlements to export",
		)
	}

	var sb strings.Builder
	sb.WriteString(
		"settlement_number,carrier_code,carrier_name,remit_to_name,remit_to_address," +
			"period_start,period_end,pay_date,net_payable,currency,payment_method," +
			"payment_reference\n",
	)
	for _, settlement := range batch.Settlements {
		if settlement == nil || settlement.Status == carriersettlement.StatusVoided {
			continue
		}
		var carrierCode, carrierName, remitToName, remitAddress, paymentMethod string
		if settlement.Carrier != nil {
			c := settlement.Carrier
			carrierCode = c.Code
			carrierName = c.Name
			remitToName = c.RemitToName
			if remitToName == "" {
				remitToName = c.Name
			}
			remitAddress = joinAddress(
				c.RemitAddressLine1,
				c.RemitAddressLine2,
				c.RemitCity,
				c.RemitPostalCode,
			)
			if remitAddress == "" {
				remitAddress = joinAddress(c.AddressLine1, c.AddressLine2, c.City, c.PostalCode)
			}
			paymentMethod = c.PaymentMethod.String()
		}
		if settlement.PaymentMethod != "" {
			paymentMethod = settlement.PaymentMethod
		}

		settlementshared.WriteCSVField(&sb, settlement.SettlementNumber)
		sb.WriteByte(',')
		settlementshared.WriteCSVField(&sb, carrierCode)
		sb.WriteByte(',')
		settlementshared.WriteCSVField(&sb, carrierName)
		sb.WriteByte(',')
		settlementshared.WriteCSVField(&sb, remitToName)
		sb.WriteByte(',')
		settlementshared.WriteCSVField(&sb, remitAddress)
		sb.WriteByte(',')
		sb.WriteString(settlementshared.FormatCSVDate(settlement.PeriodStart))
		sb.WriteByte(',')
		sb.WriteString(settlementshared.FormatCSVDate(settlement.PeriodEnd))
		sb.WriteByte(',')
		sb.WriteString(settlementshared.FormatCSVDate(settlement.PayDate))
		sb.WriteByte(',')
		sb.WriteString(money.DecimalFromMinor(settlement.NetPayableMinor).StringFixed(2))
		sb.WriteByte(',')
		sb.WriteString(settlement.CurrencyCode)
		sb.WriteByte(',')
		settlementshared.WriteCSVField(&sb, paymentMethod)
		sb.WriteByte(',')
		settlementshared.WriteCSVField(&sb, settlement.PaymentReference)
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

func joinAddress(parts ...string) string {
	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			nonEmpty = append(nonEmpty, trimmed)
		}
	}
	return strings.Join(nonEmpty, ", ")
}
