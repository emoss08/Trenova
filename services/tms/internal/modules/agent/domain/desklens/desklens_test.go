package desklens_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emoss08/trenova/internal/modules/agent/domain/desklens"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
)

func TestLens_FilesDecisionsMessagesAndTheRest(t *testing.T) {
	assert.Equal(t, desklens.Decisions, desklens.Of(watchtower.SourceKindAgentProposal))
	assert.Equal(t, desklens.Decisions, desklens.Of(watchtower.SourceKindAgentPlan))
	assert.Equal(t, desklens.Triage, desklens.Of(watchtower.SourceKindInboundMessage))
	assert.Equal(t, desklens.Watchtower, desklens.Of(watchtower.SourceKindEtaAtRisk))
}

func TestLens_KindsCoverEveryKindExactlyOnce(t *testing.T) {
	assert.Nil(t, desklens.Everything.Kinds())
	seen := map[watchtower.SourceKind]int{}
	for _, lens := range []desklens.Lens{desklens.Watchtower, desklens.Triage, desklens.Decisions} {
		for _, kind := range lens.Kinds() {
			seen[kind]++
		}
	}
	for _, kind := range watchtower.AllSourceKinds() {
		assert.Equal(t, 1, seen[kind], "kind %s", kind)
	}
}
