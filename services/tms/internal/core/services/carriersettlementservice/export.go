package carriersettlementservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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

		writeCSVField(&sb, settlement.SettlementNumber)
		sb.WriteByte(',')
		writeCSVField(&sb, carrierCode)
		sb.WriteByte(',')
		writeCSVField(&sb, carrierName)
		sb.WriteByte(',')
		writeCSVField(&sb, remitToName)
		sb.WriteByte(',')
		writeCSVField(&sb, remitAddress)
		sb.WriteByte(',')
		sb.WriteString(formatCSVDate(settlement.PeriodStart))
		sb.WriteByte(',')
		sb.WriteString(formatCSVDate(settlement.PeriodEnd))
		sb.WriteByte(',')
		sb.WriteString(formatCSVDate(settlement.PayDate))
		sb.WriteByte(',')
		sb.WriteString(money.DecimalFromMinor(settlement.NetPayableMinor).StringFixed(2))
		sb.WriteByte(',')
		sb.WriteString(settlement.CurrencyCode)
		sb.WriteByte(',')
		writeCSVField(&sb, paymentMethod)
		sb.WriteByte(',')
		writeCSVField(&sb, settlement.PaymentReference)
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

func writeCSVField(sb *strings.Builder, value string) {
	if strings.ContainsAny(value, ",\"\n") {
		sb.WriteByte('"')
		sb.WriteString(strings.ReplaceAll(value, "\"", "\"\""))
		sb.WriteByte('"')
		return
	}
	sb.WriteString(value)
}

func formatCSVDate(unix int64) string {
	return time.Unix(unix, 0).UTC().Format("2006-01-02")
}
