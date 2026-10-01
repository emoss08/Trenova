package resolver

import (
	"os"
	"regexp"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
AgentSubjectType has the same shape of hole WatchtowerSourceKind had.

It is bound to the Go type in gqlgen.yml, so gqlgen generates a pass-through
marshaller with no check against the schema's list. A subject type added to the
domain and not to the schema still marshals: the server emits it, and the
generated client — whose union does not contain it — fails to parse the run,
proposal or thread that carries it.

Every subject type is also a Postgres enum value, so the same drift has a second
edge: a value the database has not been taught is a failed insert at the moment
an event fires, not at build time.
*/

const agentSchemaFile = "../schema/agent.graphqls"

var subjectTypeEnumBlock = regexp.MustCompile(
	`(?s)enum\s+AgentSubjectType\s*\{(.*?)\n\}`,
)

func schemaSubjectTypes(t *testing.T) map[string]struct{} {
	t.Helper()

	source, err := os.ReadFile(agentSchemaFile)
	require.NoError(t, err, "could not read the agent schema")

	block := subjectTypeEnumBlock.FindSubmatch(source)
	require.NotNil(t, block, "could not find enum AgentSubjectType in the schema")

	declared := make(map[string]struct{})
	for _, match := range enumMember.FindAllSubmatch(block[1], -1) {
		declared[string(match[1])] = struct{}{}
	}
	require.NotEmpty(t, declared, "the schema enum parsed as empty")

	return declared
}

func TestAgentSubjectTypeParity_DomainMatchesSchema(t *testing.T) {
	t.Parallel()

	declared := schemaSubjectTypes(t)

	for _, subject := range agent.AllSubjectTypes() {
		assert.Containsf(t, declared, string(subject),
			"agent.SubjectType(%q) is not in enum AgentSubjectType; the server can emit "+
				"it and no generated client will be able to read it", subject)
	}
}

func TestAgentSubjectTypeParity_SchemaNamesNoSubjectTheDomainLacks(t *testing.T) {
	t.Parallel()

	known := make(map[string]struct{}, len(agent.AllSubjectTypes()))
	for _, subject := range agent.AllSubjectTypes() {
		known[string(subject)] = struct{}{}
	}

	for name := range schemaSubjectTypes(t) {
		assert.Containsf(t, known, name,
			"enum AgentSubjectType names %q, which is not an agent.SubjectType", name)
	}
}

// A subject type the domain does not call valid is one an event cannot carry,
// so declaring it anywhere else is declaring something that never arrives.
func TestAgentSubjectTypeParity_EverySubjectIsValid(t *testing.T) {
	t.Parallel()

	for _, subject := range agent.AllSubjectTypes() {
		assert.Truef(t, subject.IsValid(), "agent.SubjectType(%q) reported invalid", subject)
	}
	assert.False(t, agent.SubjectType("NotARealSubject").IsValid())
}

// Every event kind names a subject type that exists. EventKind.SubjectType
// walks knownEvents, so a kind whose descriptor names a retired subject
// resolves to something no describer has an arm for, and the run starts blind.
func TestAgentSubjectTypeParity_EveryEventNamesAKnownSubject(t *testing.T) {
	t.Parallel()

	for _, descriptor := range agent.KnownEvents() {
		assert.Truef(t, descriptor.SubjectType.IsValid(),
			"event %q names subject type %q, which is not valid",
			descriptor.Kind, descriptor.SubjectType)
	}
}
