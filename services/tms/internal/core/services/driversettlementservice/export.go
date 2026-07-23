package driversettlementservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

type PayrollExportRow struct {
	SettlementNumber    string `json:"settlementNumber"`
	WorkerName          string `json:"workerName"`
	Classification      string `json:"classification"`
	PeriodStart         int64  `json:"periodStart"`
	PeriodEnd           int64  `json:"periodEnd"`
	PayDate             int64  `json:"payDate"`
	GrossEarningsMinor  int64  `json:"grossEarningsMinor"`
	ReimbursementsMinor int64  `json:"reimbursementsMinor"`
	DeductionsMinor     int64  `json:"deductionsMinor"`
	NetPayMinor         int64  `json:"netPayMinor"`
	CurrencyCode        string `json:"currencyCode"`
}

func (s *Service) ExportBatchPayrollCSV(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
) (string, error) {
	batch, err := s.batchRepo.GetByID(ctx, repositories.GetSettlementBatchByIDRequest{
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
		"settlement_number,worker_name,classification,period_start,period_end,pay_date," +
			"gross_earnings,reimbursements,deductions,net_pay,currency\n",
	)
	for _, settlement := range batch.Settlements {
		if settlement == nil || settlement.Status == driversettlement.StatusVoided {
			continue
		}
		workerName := ""
		if settlement.Worker != nil {
			workerName = strings.TrimSpace(
				settlement.Worker.FirstName + " " + settlement.Worker.LastName,
			)
		}
		writeCSVField(&sb, settlement.SettlementNumber)
		sb.WriteByte(',')
		writeCSVField(&sb, workerName)
		sb.WriteByte(',')
		writeCSVField(&sb, settlement.Classification.String())
		sb.WriteByte(',')
		sb.WriteString(formatCSVDate(settlement.PeriodStart))
		sb.WriteByte(',')
		sb.WriteString(formatCSVDate(settlement.PeriodEnd))
		sb.WriteByte(',')
		sb.WriteString(formatCSVDate(settlement.PayDate))
		sb.WriteByte(',')
		sb.WriteString(money.DecimalFromMinor(settlement.GrossEarningsMinor).StringFixed(2))
		sb.WriteByte(',')
		sb.WriteString(money.DecimalFromMinor(settlement.ReimbursementsMinor).StringFixed(2))
		sb.WriteByte(',')
		sb.WriteString(money.DecimalFromMinor(settlement.DeductionsMinor).StringFixed(2))
		sb.WriteByte(',')
		sb.WriteString(money.DecimalFromMinor(settlement.NetPayMinor).StringFixed(2))
		sb.WriteByte(',')
		sb.WriteString(settlement.CurrencyCode)
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

type WorkerYTDPaySummary struct {
	WorkerID            pulid.ID                      `json:"workerId"`
	WorkerName          string                        `json:"workerName"`
	Classification      driverpay.PayeeClassification `json:"classification"`
	Year                int                           `json:"year"`
	SettlementCount     int                           `json:"settlementCount"`
	GrossEarningsMinor  int64                         `json:"grossEarningsMinor"`
	ReimbursementsMinor int64                         `json:"reimbursementsMinor"`
	DeductionsMinor     int64                         `json:"deductionsMinor"`
	NetPayMinor         int64                         `json:"netPayMinor"`
}

func (s *Service) GetYTDPaySummaries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	year int,
	classification driverpay.PayeeClassification,
) ([]*WorkerYTDPaySummary, error) {
	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	yearEnd := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()

	filter := &pagination.QueryOptions{TenantInfo: tenantInfo}
	filter.Pagination.Limit = 10000
	result, err := s.settlementRepo.List(ctx, &repositories.ListDriverSettlementsRequest{
		Filter: filter,
	})
	if err != nil {
		return nil, err
	}

	summaries := make(map[pulid.ID]*WorkerYTDPaySummary)
	order := make([]pulid.ID, 0)
	for _, settlement := range result.Items {
		if settlement == nil ||
			settlement.Status == driversettlement.StatusVoided ||
			settlement.Status == driversettlement.StatusDraft ||
			settlement.Status == driversettlement.StatusPendingApproval {
			continue
		}
		if settlement.PayDate < yearStart || settlement.PayDate >= yearEnd {
			continue
		}
		if classification != "" && settlement.Classification != classification {
			continue
		}
		summary, ok := summaries[settlement.WorkerID]
		if !ok {
			workerName := ""
			if settlement.Worker != nil {
				workerName = strings.TrimSpace(
					settlement.Worker.FirstName + " " + settlement.Worker.LastName,
				)
			}
			summary = &WorkerYTDPaySummary{
				WorkerID:       settlement.WorkerID,
				WorkerName:     workerName,
				Classification: settlement.Classification,
				Year:           year,
			}
			summaries[settlement.WorkerID] = summary
			order = append(order, settlement.WorkerID)
		}
		summary.SettlementCount++
		summary.GrossEarningsMinor += settlement.GrossEarningsMinor
		summary.ReimbursementsMinor += settlement.ReimbursementsMinor
		summary.DeductionsMinor += settlement.DeductionsMinor
		summary.NetPayMinor += settlement.NetPayMinor
	}

	results := make([]*WorkerYTDPaySummary, 0, len(order))
	for _, workerID := range order {
		results = append(results, summaries[workerID])
	}
	return results, nil
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
