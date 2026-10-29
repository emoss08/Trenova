package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
)

func TestAPortalTaskMustBeNamedInTheReply(t *testing.T) {
	t.Parallel()

	shown := agentruntimeports.ShownArtifact{Kind: string(assistantartifact.KindPortalTask)}
	assert.True(t, shown.MustBeLinked())
}
