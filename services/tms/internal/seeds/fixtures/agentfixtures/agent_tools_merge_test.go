package agentfixtures

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/stretchr/testify/assert"
)

func TestRaisedDataAccess_RaisesToTheTemplateAndNeverLowers(t *testing.T) {
	t.Parallel()

	access, raised := raisedDataAccess(
		agentdefinition.DataAccessInternal,
		agentpreset.StarterDataAccess(agenttemplate.TemplateReportAnalyst),
	)
	assert.True(t, raised, "the report analyst reads amounts, which are Restricted")
	assert.Equal(t, agentdefinition.DataAccessRestricted, access)

	access, raised = raisedDataAccess(
		agentdefinition.DataAccessRestricted,
		agentdefinition.DataAccessInternal,
	)
	assert.False(t, raised, "an administrator's wider grant is not the seeder's to narrow")
	assert.Equal(t, agentdefinition.DataAccessRestricted, access)

	_, raised = raisedDataAccess(
		agentdefinition.DataAccessInternal,
		agentdefinition.DataAccessInternal,
	)
	assert.False(t, raised)
}
