package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The domain has four priorities and the tool used to list three; a note
// marked Urgent was quietly filed as Normal. The list comes from the domain
// now, and a value outside it is refused rather than read as the default,
// since widening a note's audience on a typo is what reaches a customer.
func TestAddShipmentComment_TakesEveryDomainPriorityAndRefusesTheRest(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentService{}
	tool := newAddShipmentCommentTool(comments)

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"comment":    "Reefer alarm; call the driver now.",
		"priority":   "Urgent",
	})))
	assert.Equal(t, shipment.CommentPriorityUrgent, comments.created.Priority)
	assert.Equal(t, shipment.CommentVisibilityInternal, comments.created.Visibility,
		"an absent visibility is Internal")

	comments = &fakeCommentService{}
	tool = newAddShipmentCommentTool(comments)
	err := tool.Execute(t.Context(), executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"comment":    "Update.",
		"visibility": "Customers",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Customers")
	assert.Nil(t, comments.created)

	_, validates := tool.(serviceports.ToolValidator)
	assert.True(t, validates, "the note is checked before it is filed")
}
