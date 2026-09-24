package agentevalgate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

func TestEveryStarterToolIsInTheCatalog(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	for _, template := range agentdefinition.AllTemplates() {
		for _, name := range template.StarterTools() {
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
