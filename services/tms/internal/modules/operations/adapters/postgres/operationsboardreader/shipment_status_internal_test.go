package operationsboardreader

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/stretchr/testify/assert"
)

func TestShipmentStatusIsTheShipmentPagesWordAndTone(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		word string
		tone boardcatalog.Tone
	}{
		"New":            {"New", ""},
		"InTransit":      {"In transit", boardcatalog.ToneInfo},
		"Delayed":        {"Delayed", boardcatalog.ToneWarn},
		"ReadyToInvoice": {"Ready to invoice", boardcatalog.ToneWarn},
		"Completed":      {"Completed", boardcatalog.ToneOK},
		"Canceled":       {"Canceled", boardcatalog.ToneBad},
	}
	for status, want := range cases {
		word, tone := shipmentStatus(context.Background(), status)
		assert.Equal(t, want.word, word, status)
		assert.Equal(t, want.tone, tone, status)
	}
}
