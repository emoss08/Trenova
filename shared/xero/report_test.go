package xero_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrialBalanceReadsAccountRows(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Reports/TrialBalance")
		assert.Equal(t, "2026-09-30", r.URL.Query().Get("date"))
		_, _ = w.Write(fixture(t, "trial_balance.json"))
	})

	rows, err := client.TrialBalance(t.Context(), time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, rows, 3, "headers, sections and the summary row are skipped")

	revenue := rows[0]
	assert.Equal(t, "e0a3a8f4-9a61-4b5e-bd0a-7a3c9c4f5a11", revenue.AccountID)
	assert.Equal(t, "Freight Revenue (200)", revenue.Label)
	assert.True(t, revenue.Debit.IsZero())
	assert.True(t, dec("1500").Equal(revenue.Credit), "thousands separators are dropped")
	assert.True(t, revenue.YTDDebit.IsZero())
	assert.True(t, dec("12345.67").Equal(revenue.YTDCredit))

	receivable := rows[1]
	assert.Equal(t, "Accounts Receivable (610)", receivable.Label)
	assert.True(t, dec("1500").Equal(receivable.Debit))
	assert.True(t, dec("12345.67").Equal(receivable.YTDDebit))

	assert.Equal(t, testAccount, rows[2].AccountID)
	assert.True(t, dec("2500").Equal(rows[2].YTDDebit))
}

func TestTrialBalanceNeedsADateAndReadableAmounts(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Reports":[{"Rows":[{"RowType":"Section","Rows":[{"RowType":"Row",` +
			`"Cells":[{"Value":"Sales","Attributes":[{"Id":"account","Value":"a"}]},{"Value":"n/a"}]}]}]}]}`))
	})

	_, err := client.TrialBalance(t.Context(), time.Time{})
	require.ErrorIs(t, err, xero.ErrReportDateRequired)
	_, err = client.TrialBalance(t.Context(), time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, xero.ErrUnexpectedPayload)
}
