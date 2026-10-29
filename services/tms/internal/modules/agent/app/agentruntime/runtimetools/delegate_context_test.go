package runtimetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
)

// The records a task may name are the record-link registry's kinds, held to
// the registered source.
func TestDelegateTaskSpec_NamesTheRegistrysRecordKinds(t *testing.T) {
	t.Parallel()

	spec := DelegateTaskSpec([]agentturn.RuntimeDelegate{runtimetest.ShipmentDesk()})
	properties := spec.Parameters[toolschema.KeyProperties].(map[string]any)

	records := properties["records"].(map[string]any)
	assert.Equal(t, MaxDelegateRecords, records[toolschema.KeyMaxItems])
	item := records[toolschema.KeyItems].(map[string]any)
	assert.Equal(t, false, item[toolschema.KeyAdditionalProperties])
	entity := item[toolschema.KeyProperties].(map[string]any)["entityType"].(map[string]any)
	assert.Equal(t, productguide.Default.RecordEntities(), entity[toolschema.KeyEnum])
	assert.Equal(t, "guide.entity", entity[toolschema.KeyEnumOf])

	shared := properties["shareResults"].(map[string]any)
	assert.Equal(t, MaxSharedResults, shared[toolschema.KeyMaxItems])
}
