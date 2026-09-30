package xero

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	rowTypeHeader      = "Header"
	accountAttributeID = "account"
)

type TrialBalanceRow struct {
	AccountID string
	Label     string
	Debit     decimal.Decimal
	Credit    decimal.Decimal
	YTDDebit  decimal.Decimal
	YTDCredit decimal.Decimal
}

type reportAttribute struct {
	ID    string `json:"Id"`
	Value string `json:"Value"`
}

type reportCell struct {
	Value      string            `json:"Value"`
	Attributes []reportAttribute `json:"Attributes"`
}

type reportRow struct {
	RowType string       `json:"RowType"`
	Title   string       `json:"Title"`
	Cells   []reportCell `json:"Cells"`
	Rows    []reportRow  `json:"Rows"`
}

type reportsEnvelope struct {
	Reports []struct {
		Rows []reportRow `json:"Rows"`
	} `json:"Reports"`
}

type trialBalanceColumns struct {
	debit     int
	credit    int
	ytdDebit  int
	ytdCredit int
}

func (c *Client) TrialBalance(ctx context.Context, date time.Time) ([]TrialBalanceRow, error) {
	if date.IsZero() {
		return nil, ErrReportDateRequired
	}

	var out reportsEnvelope
	if err := c.do(ctx, &call{
		endpoint: "report-trial-balance",
		method:   http.MethodGet,
		path:     accountingPath("Reports", "TrialBalance"),
		query:    url.Values{"date": {formatDate(date)}},
		out:      &out,
	}); err != nil {
		return nil, err
	}
	if len(out.Reports) == 0 {
		return nil, ErrUnexpectedPayload
	}

	rows := out.Reports[0].Rows
	columns := trialBalanceColumnsOf(rows)
	result := make([]TrialBalanceRow, 0, len(rows))
	var walk func(items []reportRow) error
	walk = func(items []reportRow) error {
		for idx := range items {
			item := &items[idx]
			if len(item.Rows) > 0 {
				if err := walk(item.Rows); err != nil {
					return err
				}
			}
			row, ok, err := trialBalanceRowOf(item, columns)
			if err != nil {
				return err
			}
			if ok {
				result = append(result, row)
			}
		}
		return nil
	}
	if err := walk(rows); err != nil {
		return nil, err
	}
	return result, nil
}

func trialBalanceColumnsOf(rows []reportRow) trialBalanceColumns {
	columns := trialBalanceColumns{debit: 1, credit: 2, ytdDebit: 3, ytdCredit: 4}
	for idx := range rows {
		if rows[idx].RowType != rowTypeHeader {
			continue
		}
		for col := range rows[idx].Cells {
			switch strings.ToLower(strings.TrimSpace(rows[idx].Cells[col].Value)) {
			case "debit":
				columns.debit = col
			case "credit":
				columns.credit = col
			case "ytd debit":
				columns.ytdDebit = col
			case "ytd credit":
				columns.ytdCredit = col
			}
		}
		break
	}
	return columns
}

func trialBalanceRowOf(
	item *reportRow,
	columns trialBalanceColumns,
) (TrialBalanceRow, bool, error) {
	if len(item.Cells) == 0 {
		return TrialBalanceRow{}, false, nil
	}
	accountID := ""
	for _, attribute := range item.Cells[0].Attributes {
		if strings.EqualFold(attribute.ID, accountAttributeID) {
			accountID = strings.TrimSpace(attribute.Value)
			break
		}
	}
	if accountID == "" {
		return TrialBalanceRow{}, false, nil
	}

	row := TrialBalanceRow{AccountID: accountID, Label: strings.TrimSpace(item.Cells[0].Value)}
	targets := [...]struct {
		index int
		value *decimal.Decimal
	}{
		{columns.debit, &row.Debit},
		{columns.credit, &row.Credit},
		{columns.ytdDebit, &row.YTDDebit},
		{columns.ytdCredit, &row.YTDCredit},
	}
	for _, target := range targets {
		value, err := reportAmount(item.Cells, target.index)
		if err != nil {
			return TrialBalanceRow{}, false, err
		}
		*target.value = value
	}
	return row, true, nil
}

func reportAmount(cells []reportCell, idx int) (decimal.Decimal, error) {
	if idx < 0 || idx >= len(cells) {
		return decimal.Zero, nil
	}
	raw := strings.ReplaceAll(strings.TrimSpace(cells[idx].Value), ",", "")
	if raw == "" {
		return decimal.Zero, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, ErrUnexpectedPayload
	}
	return value, nil
}
