package agent_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validMemory(content string) *agent.Memory {
	return &agent.Memory{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Kind:           agent.MemoryKindInstruction,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		Content:        content,
	}
}

func TestMemoryValidate_ContentLimitComesFromTheConstant(t *testing.T) {
	t.Parallel()

	atLimit := validMemory(strings.Repeat("a", agent.MaxMemoryContentChars))
	me := errortypes.NewMultiError()
	atLimit.Validate(me)
	assert.False(t, me.HasErrors(), "a memory exactly at the limit is accepted")

	over := validMemory(strings.Repeat("a", agent.MaxMemoryContentChars+1))
	me = errortypes.NewMultiError()
	over.Validate(me)
	require.True(t, me.HasErrors())

	var message string
	for _, err := range me.Errors {
		if err.Field == "content" {
			message = err.Message
		}
	}
	assert.Equal(t,
		"Content must be at most "+strconv.Itoa(agent.MaxMemoryContentChars)+" characters",
		message,
	)
	assert.NotContains(t, message, "2000")
}

func TestMemoryLimits_AreTheAgreedDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 4000, agent.MaxMemoryContentChars)
	assert.Equal(t, 500, agent.MaxMemoryCandidates)
	assert.Equal(t, 5000, agent.MemoryActiveSoftCap)
	assert.Equal(t, 4000, agent.MemoryActiveWarnAt, "the warning shows at 80% of the cap")
	assert.Equal(t, 10, agent.DefaultMemoryRecallLimit)
	assert.Equal(t, 50, agent.MaxMemoryRecallLimit)
	assert.Less(t, agent.MemoryPromptExcerptChars, agent.MaxMemoryContentChars)
}
