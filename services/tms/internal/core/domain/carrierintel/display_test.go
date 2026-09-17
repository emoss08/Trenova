package carrierintel_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFormatFieldValue(t *testing.T) {
	t.Parallel()

	ratingDate := float64(time.Date(1996, time.September, 25, 0, 0, 0, 0, time.UTC).Unix())

	cases := []struct {
		name  string
		path  string
		value any
		want  string
	}{
		{name: "missing", path: "safety.ratingDate", value: nil, want: "none"},
		{name: "empty text", path: "safety.rating", value: "", want: "none"},
		{name: "date", path: "safety.ratingDate", value: ratingDate, want: "Sep 25, 1996"},
		{name: "at suffix", path: "operations.mcs150At", value: ratingDate, want: "Sep 25, 1996"},
		{
			name:  "money from decimal json",
			path:  "insurance.bipdRequired",
			value: "5000000",
			want:  "$5,000,000",
		},
		{
			name:  "money from decimal",
			path:  "insurance.bondOnFile",
			value: decimal.NewFromInt(75000),
			want:  "$75,000",
		},
		{name: "money from number", path: "insurance.cargoOnFile", value: 5000.0, want: "$5,000"},
		{
			name:  "percent",
			path:  "inspections.nationalVehicleOosRate",
			value: 22.26,
			want:  "22.26%",
		},
		{name: "percentile", path: "basics.UnsafeDriving.percentile", value: 65.0, want: "65%"},
		{name: "fraction", path: "safety.riskProbability", value: 0.4433, want: "44.33%"},
		{name: "yes", path: "safety.outOfServiceOrder", value: true, want: "Yes"},
		{name: "no", path: "operations.hazmatCarrier", value: false, want: "No"},
		{name: "count", path: "fleet.powerUnits", value: 101844.0, want: "101,844"},
		{name: "measure", path: "basics.UnsafeDriving.measure", value: 3.13, want: "3.13"},
		{name: "text", path: "safety.rating", value: "Satisfactory", want: "Satisfactory"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, carrierintel.FormatFieldValue(tc.path, tc.value), tc.name)
	}
}
