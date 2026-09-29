package quickbooks

import (
	"context"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

type TrialBalanceRow struct {
	AccountID   string
	AccountName string
	Debit       decimal.Decimal
	Credit      decimal.Decimal
}

type TrialBalanceRequest struct {
	StartDate string
	EndDate   string
}

type reportColData struct {
	Value string `json:"value"`
	ID    string `json:"id"`
}

type reportRow struct {
	Type    string          `json:"type"`
	ColData []reportColData `json:"ColData"`
	Rows    *reportRows     `json:"Rows"`
}

type reportRows struct {
	Row []reportRow `json:"Row"`
}

type reportColumn struct {
	ColTitle string `json:"ColTitle"`
	ColType  string `json:"ColType"`
}

type reportEnvelope struct {
	Columns struct {
		Column []reportColumn `json:"Column"`
	} `json:"Columns"`
	Rows reportRows `json:"Rows"`
}

func (c *Client) TrialBalance(
	ctx context.Context,
	req *TrialBalanceRequest,
) ([]TrialBalanceRow, error) {
	if req == nil || strings.TrimSpace(req.EndDate) == "" {
		return nil, ErrReportDateRequired
	}
	query := c.query()
	query.Set("accounting_method", "Accrual")
	query.Set("end_date", strings.TrimSpace(req.EndDate))
	if start := strings.TrimSpace(req.StartDate); start != "" {
		query.Set("start_date", start)
	}

	var out reportEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: "report-trialbalance",
		Method:   http.MethodGet,
		Path:     c.companyPath("reports", "TrialBalance"),
		Query:    query,
		Out:      &out,
	}); err != nil {
		return nil, err
	}

	debitCol, creditCol := trialBalanceColumns(out.Columns.Column)
	rows := make([]TrialBalanceRow, 0, len(out.Rows.Row))
	var walk func(items []reportRow) error
	walk = func(items []reportRow) error {
		for idx := range items {
			item := &items[idx]
			if item.Rows != nil {
				if err := walk(item.Rows.Row); err != nil {
					return err
				}
			}
			if len(item.ColData) == 0 || strings.TrimSpace(item.ColData[0].ID) == "" {
				continue
			}
			debit, err := reportAmount(item.ColData, debitCol)
			if err != nil {
				return err
			}
			credit, err := reportAmount(item.ColData, creditCol)
			if err != nil {
				return err
			}
			rows = append(rows, TrialBalanceRow{
				AccountID:   strings.TrimSpace(item.ColData[0].ID),
				AccountName: strings.TrimSpace(item.ColData[0].Value),
				Debit:       debit,
				Credit:      credit,
			})
		}
		return nil
	}
	if err := walk(out.Rows.Row); err != nil {
		return nil, err
	}
	return rows, nil
}

func trialBalanceColumns(columns []reportColumn) (debit, credit int) {
	debit, credit = 1, 2
	for idx := range columns {
		switch strings.ToLower(strings.TrimSpace(columns[idx].ColTitle)) {
		case "debit":
			debit = idx
		case "credit":
			credit = idx
		}
	}
	return debit, credit
}

func reportAmount(cols []reportColData, idx int) (decimal.Decimal, error) {
	if idx < 0 || idx >= len(cols) {
		return decimal.Zero, nil
	}
	raw := strings.ReplaceAll(strings.TrimSpace(cols[idx].Value), ",", "")
	if raw == "" {
		return decimal.Zero, nil
	}
	amount, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, ErrUnexpectedPayload
	}
	return amount, nil
}
