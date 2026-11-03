package operationstools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
	"github.com/stretchr/testify/assert"
)

// A shipment comment records the tool that wrote it, and the customer-update
// queries find the updates emailed to a customer by that value, so the tool's
// name and the stored source must be the same word.
func TestEmailCustomerTool_IsNamedAsTheCommentsItWritesRecordIt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, customerupdateservice.SourceAgentEmail, (&EmailCustomerTool{}).Name())
}
