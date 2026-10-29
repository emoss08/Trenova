package benchcell

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCellPin_CarriesTheCellsProviderEffortAndAdvisor(t *testing.T) {
	t.Parallel()

	cell := Cell{
		Provider: &aiprovider.Provider{
			ID: pulid.MustNew("aiprv_"), Kind: aiprovider.KindAnthropicMessages,
			Model: "claude-haiku-5-5",
		},
		Effort:  aireasoning.ReasoningLow,
		Advisor: llmports.DefaultAdvisor(""),
	}

	assert.Equal(t, benchrecord.ProviderPin{
		ProviderID: cell.Provider.ID, Effort: cell.Effort, Advisor: cell.Advisor,
	}, cell.Pin())
}
