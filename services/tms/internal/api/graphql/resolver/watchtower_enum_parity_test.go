package resolver

import (
	"os"
	"regexp"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Every watchtower kind the server can emit has to be declared in the schema.

Nothing else catches this. WatchtowerSourceKind is bound to the Go type in
gqlgen.yml, so gqlgen generates a pass-through marshaller — graphql.MarshalString
over the raw value, with no check against the schema's list. The server will
happily send a kind the schema never declared, and the failure lands on whoever
generated a client from that schema: the value is absent from their union, so a
feed carrying one row of it fails to parse.

That is exactly what happened to WorkerCredential and MoveCoverage. They were
added to the domain with describers and publish sites, the schema was not
touched, and the build stayed green while the credential sweep and the dispatch
planner produced items no client could read.

The schema is parsed as text rather than through gqlmodel, because a bound enum
generates no All… slice to compare against — which is the same fact that makes
the drift possible in the first place.
*/

const watchtowerSchemaFile = "../schema/watchtower.graphqls"

var sourceKindEnumBlock = regexp.MustCompile(
	`(?s)enum\s+WatchtowerSourceKind\s*\{(.*?)\n\}`,
)

var enumMember = regexp.MustCompile(`(?m)^\s{2}([A-Za-z][A-Za-z0-9_]*)\s*$`)

func schemaSourceKinds(t *testing.T) map[string]struct{} {
	t.Helper()

	source, err := os.ReadFile(watchtowerSchemaFile)
	require.NoError(t, err, "could not read the watchtower schema")

	block := sourceKindEnumBlock.FindSubmatch(source)
	require.NotNil(t, block, "could not find enum WatchtowerSourceKind in the schema")

	declared := make(map[string]struct{})
	for _, match := range enumMember.FindAllSubmatch(block[1], -1) {
		declared[string(match[1])] = struct{}{}
	}
	require.NotEmpty(t, declared, "the schema enum parsed as empty")

	return declared
}

func TestWatchtowerSourceKindParity_DomainMatchesSchema(t *testing.T) {
	t.Parallel()

	declared := schemaSourceKinds(t)

	for _, kind := range watchtower.AllSourceKinds() {
		assert.Containsf(t, declared, string(kind),
			"watchtower.SourceKind(%q) is not in enum WatchtowerSourceKind; the server "+
				"can emit it and no generated client will be able to read it", kind)
	}
}

func TestWatchtowerSourceKindParity_SchemaNamesNoKindTheDomainLacks(t *testing.T) {
	t.Parallel()

	known := make(map[string]struct{}, len(watchtower.AllSourceKinds()))
	for _, kind := range watchtower.AllSourceKinds() {
		known[string(kind)] = struct{}{}
	}

	for name := range schemaSourceKinds(t) {
		assert.Containsf(t, known, name,
			"enum WatchtowerSourceKind names %q, which is not a watchtower.SourceKind; "+
				"a client could ask to filter by a kind that can never arrive", name)
	}
}

// A kind with no read permission is one visibleKinds cannot filter, so it would
// either leak to every reader or vanish for all of them.
func TestWatchtowerSourceKindParity_EveryKindIsValid(t *testing.T) {
	t.Parallel()

	for _, kind := range watchtower.AllSourceKinds() {
		assert.Truef(t, kind.IsValid(), "watchtower.SourceKind(%q) reported invalid", kind)
	}
	assert.False(t, watchtower.SourceKind("NotARealSourceKind").IsValid())
	assert.False(t, watchtower.SourceKind("").IsValid())
}
