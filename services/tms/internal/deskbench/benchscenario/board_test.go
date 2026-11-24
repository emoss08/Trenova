package benchscenario

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpectedBoardHoldsItsNamesToTheCatalog(t *testing.T) {
	t.Parallel()

	assert.NoError(t, (&ExpectedBoard{
		Reads:   []string{"expiry", "lateness"},
		Kinds:   []string{"worker_credential", "shipment"},
		Metrics: []string{"receivables"},
		Figures: []string{"arOverdue:receivables"},
	}).validate())
	for name, board := range map[string]*ExpectedBoard{
		"an unknown measure":       {Reads: []string{"latenes"}},
		"a kind no board shows":    {Kinds: []string{"edi_partner"}},
		"a metric by its alias":    {Metrics: []string{"ar"}},
		"a figure with no metric":  {Figures: []string{"arOverdue"}},
		"a figure it does not say": {Figures: []string{"lowest:receivables"}},
	} {
		assert.Error(t, board.validate(), name)
	}
}
