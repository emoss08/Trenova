package agentevalgate_test

import (
	"testing"

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
