package inboxquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/documentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchToolPoliciesReadOutsideTextAlways(t *testing.T) {
	t.Parallel()

	documents := documentquerytools.NewSearchDocumentsTool(nil, nil, nil).Policy()
	assert.Equal(t, permission.ResourceDocument, documents.Resource)
	assert.Equal(t, agenttooltrust.ExternalReadAlways, documents.ReadsExternal)
	assert.Equal(t, agent.TaintSourceDocument, documents.Source)

	inbox, err := ProvideListInboundMessagesTool(&querytooltest.KeyedReader{}, nil, nil)
	require.NoError(t, err)
	messages := inbox.Policy()
	assert.Equal(t, permission.ResourceInboundMessage, messages.Resource)
	assert.Equal(t, agenttooltrust.ExternalReadAlways, messages.ReadsExternal)
	assert.Equal(t, agent.TaintSourceInboundMessage, messages.Source)
}
