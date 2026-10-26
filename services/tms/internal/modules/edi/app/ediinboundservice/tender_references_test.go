package ediinboundservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/stretchr/testify/require"
)

func TestTenderReferencesReadsAStoredTransactionWithItsSeparators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		envelope  *edi.X12EnvelopeSettings
		want      TenderReference
		wantFound bool
	}{
		{
			name:      "an X12 B2",
			raw:       "ST*204*0001~B2**ACME**LT-88240**PP~B2A*00~SE*4*0001~",
			envelope:  separators("*", "~"),
			want:      TenderReference{ShipmentID: "LT-88240", PaymentMethod: "PP"},
			wantFound: true,
		},
		{
			name:      "the layout Trenova once wrote",
			raw:       "ST*204*0001~B2*ABCD*shp_01JABCDEF**PP~SE*3*0001~",
			envelope:  separators("*", "~"),
			want:      TenderReference{ShipmentID: "shp_01JABCDEF", PaymentMethod: "PP"},
			wantFound: true,
		},
		{
			name:      "a partner's own separators, one segment to a line",
			raw:       "ST|204|0001!\nB2||ACME||LT-88240||CC!\nSE|3|0001!\n",
			envelope:  separators("|", "!"),
			want:      TenderReference{ShipmentID: "LT-88240", PaymentMethod: "CC"},
			wantFound: true,
		},
		{
			name:     "no B2",
			raw:      "ST*204*0001~B2A*00~SE*3*0001~",
			envelope: separators("*", "~"),
		},
		{
			name: "nothing kept",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, found, err := TenderReferences(t.Context(), tt.raw, tt.envelope)
			require.NoError(t, err)
			require.Equal(t, tt.wantFound, found)
			require.Equal(t, tt.want, got)
		})
	}
}

func separators(element, segment string) *edi.X12EnvelopeSettings {
	return &edi.X12EnvelopeSettings{
		ElementSeparator:    element,
		SegmentTerminator:   segment,
		ComponentSeparator:  ">",
		RepetitionSeparator: "^",
	}
}
