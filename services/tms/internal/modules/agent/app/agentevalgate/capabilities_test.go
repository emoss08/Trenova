package agentevalgate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryRegisteredToolIsInTheCapabilityRegistry(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	registry, err := agenttoolcatalog.BuildCapabilities(kit.Queries, kit.Actions, kit.Runtime)
	require.NoError(t, err)

	for _, facts := range agenttoolcatalog.CapabilityFacts(kit.Queries, kit.Actions, kit.Runtime) {
		id := agentcapability.CapabilityIDForTool(facts.Name)
		if _, _, include := agentcapability.EntityBacking(facts.Name); include && id == facts.Name {
			continue
		}
		capability, ok := registry.Get(id)
		if !assert.True(t, ok, "%s has no registry entry", facts.Name) {
			continue
		}
		assert.NotEmpty(t, capability.Area, facts.Name)
		assert.NotEmpty(t, capability.Verb, facts.Name)
		assert.NotEmpty(t, capability.Access, facts.Name)
		if capability.Access != agentcapability.AccessTurn {
			assert.NotEqual(t, ":", capability.Permission, facts.Name)
			assert.NotEmpty(t, capability.Summary, facts.Name)
		}
	}
}

func TestEveryEntityReadsThroughARegisteredTool(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	for _, entity := range agentcapability.Entities() {
		for _, reader := range []string{entity.List, entity.Get} {
			if reader == "" {
				continue
			}
			_, ok := kit.Queries.Get(reader)
			assert.True(t, ok, "%s reads through %s, which is not a read tool", entity.Name, reader)
		}
		for _, include := range entity.Includes {
			if include.Reader == "" {
				continue
			}
			_, ok := kit.Queries.Get(include.Reader)
			assert.True(t, ok, "%s includes %s through %s", entity.Name, include.Name, include.Reader)
		}
	}
}

func TestEveryGuidanceEntryNamesACapability(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	registry, err := agenttoolcatalog.BuildCapabilities(kit.Queries, kit.Actions, kit.Runtime)
	require.NoError(t, err)
	for _, id := range agentcapability.GuidanceIDs() {
		_, ok := registry.Get(id)
		assert.True(t, ok, "guidance is written for %s, which is no capability", id)
	}
	for _, capability := range registry.All() {
		for _, instead := range capability.InsteadOf {
			_, ok := registry.Get(instead.Capability)
			assert.True(t, ok, "%s says use %s instead, which is no capability", capability.ID, instead.Capability)
		}
		for _, step := range capability.Chain {
			_, ok := registry.Get(agentcapability.CapabilityIDForTool(step))
			assert.True(t, ok, "%s chains to %s, which is no capability", capability.ID, step)
		}
	}
}
