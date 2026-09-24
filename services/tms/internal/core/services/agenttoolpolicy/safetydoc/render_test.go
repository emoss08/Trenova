package safetydoc_test

import (
	"os"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/registered"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/safetydoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const committed = "../../../../../../../docs/engineering/ai-tool-safety.md"

func registeredPolicies(t *testing.T) []serviceports.ToolPolicy {
	t.Helper()

	catalog, err := registered.Catalog()
	require.NoError(t, err)

	return catalog.All()
}

func TestRenderIsStable(t *testing.T) {
	t.Parallel()

	policies := registeredPolicies(t)
	first := safetydoc.Render(policies)

	reversed := make([]serviceports.ToolPolicy, 0, len(policies))
	for idx := len(policies) - 1; idx >= 0; idx-- {
		reversed = append(reversed, policies[idx])
	}

	assert.Equal(t, string(first), string(safetydoc.Render(policies)))
	assert.Equal(t, string(first), string(safetydoc.Render(reversed)),
		"the order tools are registered in must not move the document")
}

func TestCommittedDocumentIsCurrent(t *testing.T) {
	t.Parallel()

	want, err := os.ReadFile(committed)
	require.NoError(t, err)

	assert.Equal(t, string(want), string(safetydoc.Render(registeredPolicies(t))),
		"docs/engineering/ai-tool-safety.md is stale; run "+
			"go generate ./internal/core/services/agenttoolpolicy/safetydoc/...")
}

func TestEveryToolIsListedOnceUnderItsHighestClass(t *testing.T) {
	t.Parallel()

	policies := registeredPolicies(t)
	doc := string(safetydoc.Render(policies))

	for idx := range policies {
		assert.Equal(t, 1, strings.Count(doc, "(`"+policies[idx].Name+"`)"), policies[idx].Name)
	}

	sendSection := doc[strings.Index(doc, "## Sent outside the organization"):]
	assert.Contains(t, sendSection, "(`email_customer`)")
	assert.NotContains(t, doc[:strings.Index(doc, "## Sent outside the organization")],
		"(`email_customer`)")
}

func TestRenderEscapesTableCells(t *testing.T) {
	t.Parallel()

	doc := string(safetydoc.Render([]serviceports.ToolPolicy{{
		Name:          "pipe_tool",
		Kind:          agent.ToolKindAction,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		MaxTier:       agent.TierAutoExecute,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Splits a | b\nacross lines",
	}}))

	assert.Contains(t, doc, `Splits a \| b across lines`)
	assert.Contains(t, doc, "Tools listed: 1.")
}
