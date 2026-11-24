package ediquerytools

import (
	"testing"

	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/assert"
)

func TestAnswerNotInRow_SaysWhereAnAnsweredTendersNineNinetyIsRead(t *testing.T) {
	t.Parallel()

	approved := map[string]any{
		"id": "edilt_01M85JM3GBC6GG1DN85XW6W620", "direction": "Inbound", "status": "Approved",
	}
	answerNotInRow(approved, nil, filtercatalog.Clock{})
	assert.Contains(t, approved[fieldOurAnswer],
		`edi_list_messages reads, by transferId "edilt_01M85JM3GBC6GG1DN85XW6W620"`,
		"final-a EDI twin #1 and #3 called an Approved tender through without reading its 990")

	pending := map[string]any{"id": "edilt_01M85JM3FY0CV3268MED8VB0GD", "direction": "Inbound",
		"status": "PendingApproval"}
	answerNotInRow(pending, nil, filtercatalog.Clock{})
	assert.NotContains(t, pending, fieldOurAnswer, "no 990 is due on a tender not yet decided")

	outbound := map[string]any{"id": "edilt_01M85JM3FKYZ54JC3KNQSP1YD8", "direction": "Outbound",
		"status": "Approved"}
	answerNotInRow(outbound, nil, filtercatalog.Clock{})
	assert.NotContains(t, outbound, fieldOurAnswer, "an outbound tender is answered by the partner")
}
