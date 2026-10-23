package tarifftypes_test

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bill = `{
  "lines": [
    {"kind":"charge","label":"Linehaul","code":null,"freight":true,
     "amount":{"$q":"1908.2","u":"$"},"quantity":{"$q":"812","u":"mi"},
     "rate":{"$q":"2.35","u":"$/mi"},"range":{"from":0,"to":10}},
    {"kind":"charge","label":"Fuel surcharge","code":"FSC","freight":false,
     "amount":{"$q":"190.82","u":"$"},"quantity":null,"rate":null,"range":null},
    {"kind":"rounding","label":null,"code":null,"freight":true,
     "amount":{"$q":"0.01","u":"$"},"range":null}
  ],
  "notes":[{"text":"Peg $1.20","range":null}],
  "reviews":[{"text":"Liftgate billed on a dock delivery","range":{"from":12,"to":40}},
             {"text":"Holiday pickup","range":null}],
  "freightTotal":{"$q":"1908.21","u":"$"},
  "accessorialTotal":{"$q":"190.82","u":"$"},
  "total":{"$q":"2099.03","u":"$"}
}`

func TestBillFromReadsLinesQuantitiesAndRates(t *testing.T) {
	t.Parallel()

	var wire rules.TariffBill
	require.NoError(t, sonic.Unmarshal([]byte(bill), &wire))
	out, err := tarifftypes.BillFrom(&wire, "USD")
	require.NoError(t, err)

	require.Len(t, out.Lines, 3)
	assert.Equal(t, tarifftypes.LineCharge, out.Lines[0].Kind)
	assert.Equal(t, "812", out.Lines[0].Quantity.String())
	assert.Equal(t, "$/mi", out.Lines[0].RateUnit)
	assert.Equal(t, tarifftypes.LineRounding, out.Lines[2].Kind)
	assert.True(t, out.BillsCode("FSC"))
	assert.Len(t, out.FreightLines(), 2)
	assert.Equal(t, "2099.03", out.Total.String())
	assert.Equal(t, "USD", out.Currency)
}

func TestBillFromKeepsALinesLimitAndAFreightAdjustmentsScope(t *testing.T) {
	t.Parallel()

	const raw = `{
  "lines": [
    {"kind":"charge","label":"Linehaul","code":null,"freight":true,
     "amount":{"$q":"450","u":"$"},"quantity":{"$q":"100","u":"mi"},
     "rate":{"$q":"2.35","u":"$/mi"},
     "limit":{"kind":"minimum","computed":{"$q":"235","u":"$"}},"on":null,"range":null},
    {"kind":"maximum","label":null,"code":null,"freight":true,
     "amount":{"$q":"-50","u":"$"},"limit":null,"on":"freight","range":null}
  ],
  "notes":[],
  "freightTotal":{"$q":"400","u":"$"},
  "accessorialTotal":{"$q":"0","u":"$"},
  "total":{"$q":"400","u":"$"}
}`
	var wire rules.TariffBill
	require.NoError(t, sonic.Unmarshal([]byte(raw), &wire))
	out, err := tarifftypes.BillFrom(&wire, "USD")
	require.NoError(t, err)

	require.Len(t, out.Lines, 2)
	require.NotNil(t, out.Lines[0].Limit)
	assert.Equal(t, tarifftypes.LineMinimum, out.Lines[0].Limit.Kind)
	assert.Equal(t, "235", out.Lines[0].Limit.Computed.String())
	assert.Empty(t, out.Lines[0].On)
	assert.Nil(t, out.Lines[1].Limit)
	assert.Equal(t, "freight", out.Lines[1].On)

	encoded, err := sonic.Marshal(out.Lines[1])
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "limit")
}

func TestBillFromRefusesAnUnknownLineLimit(t *testing.T) {
	t.Parallel()

	const raw = `{
  "lines": [
    {"kind":"charge","label":"Linehaul","code":null,"freight":true,
     "amount":{"$q":"450","u":"$"},
     "limit":{"kind":"floor","computed":{"$q":"235","u":"$"}},"range":null}
  ],
  "notes":[],
  "freightTotal":{"$q":"450","u":"$"},
  "accessorialTotal":{"$q":"0","u":"$"},
  "total":{"$q":"450","u":"$"}
}`
	var wire rules.TariffBill
	require.NoError(t, sonic.Unmarshal([]byte(raw), &wire))
	_, err := tarifftypes.BillFrom(&wire, "USD")
	require.ErrorIs(t, err, tarifftypes.ErrBillShape)
}

func TestBillFromReadsTheReviewsBesideTheNotes(t *testing.T) {
	t.Parallel()

	var wire rules.TariffBill
	require.NoError(t, sonic.Unmarshal([]byte(bill), &wire))
	out, err := tarifftypes.BillFrom(&wire, "USD")
	require.NoError(t, err)

	assert.Equal(t, []tarifftypes.Note{{Text: "Peg $1.20"}}, out.Notes)
	assert.Equal(t, []tarifftypes.Note{
		{Text: "Liftgate billed on a dock delivery"},
		{Text: "Holiday pickup"},
	}, out.Reviews)
}

func TestBillFromLeavesReviewsEmptyWhenTheBillHasNone(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		`{"lines":[],"notes":[],"reviews":[],"freightTotal":{"$q":"0","u":"$"},
		  "accessorialTotal":{"$q":"0","u":"$"},"total":{"$q":"0","u":"$"}}`,
		`{"lines":[],"notes":[],"freightTotal":{"$q":"0","u":"$"},
		  "accessorialTotal":{"$q":"0","u":"$"},"total":{"$q":"0","u":"$"}}`,
	} {
		var wire rules.TariffBill
		require.NoError(t, sonic.Unmarshal([]byte(raw), &wire))
		out, err := tarifftypes.BillFrom(&wire, "USD")
		require.NoError(t, err)
		assert.Empty(t, out.Reviews)

		encoded, err := sonic.Marshal(out)
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "reviews")
	}
}

func TestBillFromUsesTheConvertedAmounts(t *testing.T) {
	t.Parallel()

	var wire rules.TariffBill
	require.NoError(t, sonic.Unmarshal([]byte(bill), &wire))
	require.NoError(t, sonic.Unmarshal([]byte(`{"currency":"CAD","rate":"1.35",
	  "lines":[{"$q":"2576.07","u":"$"},{"$q":"257.61","u":"$"},{"$q":"0","u":"$"}],
	  "total":{"$q":"2833.68","u":"$"}}`), &wire.Converted))
	out, err := tarifftypes.BillFrom(&wire, "USD")
	require.NoError(t, err)

	assert.Equal(t, "CAD", out.Currency)
	assert.Equal(t, "2833.68", out.Total.String())
	assert.Equal(t, "2576.07", out.FreightTotal.String())
	assert.Equal(t, "USD", out.Exchange.Currency)
}

func TestInputsCompareByValueNotSpelling(t *testing.T) {
	t.Parallel()

	a := tarifftypes.Inputs{"rate": rules.RawJSON(`{"$q":"2.35", "u":"$/mi"}`)}
	b := tarifftypes.Inputs{"rate": rules.RawJSON(`{"u":"$/mi","$q":"2.35"}`)}
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(tarifftypes.Inputs{}))
}
