package agentevalgate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/stretchr/testify/assert"
)

func TestEveryStarterToolIsInTheCatalog(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	for _, template := range agenttemplate.AllTemplates() {
		for _, name := range agentpreset.StarterTools(template) {
			_, query := kit.Queries.Get(name)
			_, action := kit.Actions.Get(name)
			assert.Truef(t, query || action,
				"the %s starter lists %s, which no registered tool answers to, so an agent "+
					"made from it silently goes without",
				template.Label(), name,
			)
		}
	}
}
