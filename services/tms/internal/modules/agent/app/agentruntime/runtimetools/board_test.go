package runtimetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoardStepOutcome_ReadsTheBoardItRevises(t *testing.T) {
	t.Parallel()

	revises := pulid.MustNew("art_")
	args := runtimetest.ShipmentBoard(pulid.MustNew("shp_").String())
	args["boardId"] = revises.String()

	outcome := BoardStepOutcome(args)

	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, revises, request.ArtifactID)
	assert.True(t, outcome.Publishes)
}
