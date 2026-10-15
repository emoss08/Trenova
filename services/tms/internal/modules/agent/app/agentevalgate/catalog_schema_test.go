package agentevalgate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
)

func TestEveryToolSchemaCompiles(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	for _, descriptor := range kit.Queries.Descriptors() {
		assert.NoError(t, toolschema.Compile(descriptor.Parameters), descriptor.Name)
	}
	for _, descriptor := range kit.Actions.Descriptors() {
		assert.NoError(t, toolschema.Compile(descriptor.Parameters), descriptor.Name)
	}
}

func TestEveryToolThatCapsLaterWritesIsOneTheRuntimeCapsOn(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	for _, tool := range kit.Queries.All() {
		policy := tool.Policy()
		assert.Equal(t, policy.CapsLaterWrites, agentruntime.ReadsExternalContent(policy.Name),
			"%s: its policy and the runtime must agree on whether a write after it waits "+
				"for a person", policy.Name)
	}
	for _, tool := range kit.Actions.All() {
		policy := tool.Policy()
		assert.False(t, policy.CapsLaterWrites, policy.Name)
		assert.False(t, agentruntime.ReadsExternalContent(policy.Name), policy.Name)
	}
}
