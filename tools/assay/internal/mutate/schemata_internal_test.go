package mutate

import (
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const schemataModule = "example.com/schemata"

// schemataFixture exercises every mutator, and every gate: named numeric and
// boolean types, a map-index increment that has no address, constant declarations
// and a generic function whose constraint does not imply the helpers'.
const schemataFixture = `package target

type Meter int

type Flag bool

const Limit = 10

const Derived = Limit * 2

type Bucket struct {
	Total int
	Hits  map[string]int
}

func Sum(a, b int) int { return a + b }

func Difference(a, b int) int { return a - b }

func Product(a, b int) int { return a * b }

func Quotient(a, b int) int { return a / b }

func Remainder(a, b int) int { return a % b }

func Span(d Meter, n int) Meter { return d * Meter(n) }

func Below(a, b int) bool { return a < b }

func AtMost(a, b int) bool { return a <= b }

func Above(a, b int) bool { return a > b }

func AtLeast(a, b int) bool { return a >= b }

func Sorted(a, b string) bool { return a < b }

func NamedResult(a, b int) Flag { return a < b }

func Same(a, b any) bool { return a == b }

func Differs(err error) bool { return err != nil }

func Both(a, b bool) bool { return a && b }

func Either(a, b bool) bool { return a || b }

func NamedConnector(a, b Flag) Flag { return a && b }

func Enabled() bool { return true }

func Disabled() Flag { return false }

func (b *Bucket) Bump() { b.Total++ }

func (b *Bucket) Hit(key string) { b.Hits[key]++ }

func (b *Bucket) Drop() { b.Total-- }

func Guard(n int) int {
	if n > Limit {
		return Limit
	}

	return n
}

func Chained(a, b, c int) bool {
	return a+b < c && a > 0
}

func Generic[T int | string](a, b T) bool { return a < b }

func Scaled(d Meter) Meter { return d * 2 }
`

func writeSchemataFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module "+schemataModule+"\n\ngo 1.26\n"), 0o644))

	dir := filepath.Join(root, "target")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "target.go"), []byte(schemataFixture), 0o644))

	return root
}

func generateSchemataFixture(t *testing.T, root string) []Mutant {
	t.Helper()
	t.Setenv("GOWORK", "off")

	mutants, err := Generate(t.Context(), GenerateOptions{
		Root:    root,
		Package: schemataModule + "/target",
		Env:     append(os.Environ(), "GOWORK=off"),
		// planBatches skips mutants with no covering test, so the fixture stands in
		// for an index here; this test is about compiling, not judging.
		Tests: func(string, int) TestPlan {
			return TestPlan{schemataModule + "/target": {"TestPlaceholder"}}
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, mutants)

	return mutants
}

// TestSchemataPackageCompiles is the property the whole fast path rests on: one
// rewritten file carrying every mutation site at once still type-checks. A single
// bad rewrite would not cost one mutant, it would invalidate the shared binary and
// every mutant in it.
func TestSchemataPackageCompiles(t *testing.T) {
	root := writeSchemataFixture(t)
	mutants := generateSchemataFixture(t, root)

	batches, _ := planBatches(mutants)
	require.Len(t, batches, 1)

	batch := batches[0]
	t.Cleanup(batch.cleanup)
	require.NoError(t, batch.prepare())

	cmd := exec.Command("go", "build", "-overlay="+batch.overlay,
		"-o", filepath.Join(t.TempDir(), "out.a"), schemataModule+"/target")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")

	out, err := cmd.CombinedOutput()
	if err != nil {
		rendered, readErr := os.ReadFile(filepath.Join(batch.workdir, "0", "target.go"))
		require.NoError(t, readErr)
		t.Fatalf("the schemata package does not compile:\n%s\n--- rendered ---\n%s", out, rendered)
	}
}

// TestSchemataBuildsOnlyTheBinariesItUses pins the laziness. Building a binary for
// every test package a batch might reach costs more than it saves, because a mutant
// stops at the first package that kills it — that eagerness made a diff-scoped run
// slower than compiling each mutant separately, which is the whole thing this path
// exists to beat. Nothing else would notice if it came back.
func TestSchemataBuildsOnlyTheBinariesItUses(t *testing.T) {
	root := writeSchemataFixture(t)
	mutants := generateSchemataFixture(t, root)

	batches, _ := planBatches(mutants)
	require.Len(t, batches, 1)

	batch := batches[0]
	t.Cleanup(batch.cleanup)
	require.NoError(t, batch.prepare())

	bin := filepath.Join(batch.workdir, "bin")
	_, err := os.Stat(bin)
	assert.True(t, os.IsNotExist(err),
		"preparing a batch must not compile anything; the first mutant that needs a binary pays for it")

	for _, slot := range batch.binaries {
		assert.Empty(t, slot.path)
		assert.NoError(t, slot.err)
	}
}

func TestSchemataCoversEveryMutatorKind(t *testing.T) {
	mutants := generateSchemataFixture(t, writeSchemataFixture(t))

	schemata := make(map[Kind]int)
	declined := make(map[Kind]int)
	for _, m := range mutants {
		if m.Schemata() {
			schemata[m.Kind]++

			continue
		}
		declined[m.Kind]++
	}

	for _, kind := range Kinds() {
		assert.Positive(t, schemata[kind],
			"no %s mutant took the shared-binary path, so the fast path silently covers nothing", kind)
	}
}

// TestSchemataDeclinesWhatItCannotType pins the gates. Each of these would compile
// into something the type checker rejects, and being wrong here breaks every
// mutant in the package rather than one.
func TestSchemataDeclinesWhatItCannotType(t *testing.T) {
	mutants := generateSchemataFixture(t, writeSchemataFixture(t))

	declined := make(map[string][]Kind)
	for _, m := range mutants {
		if !m.Schemata() {
			declined[m.Function] = append(declined[m.Function], m.Kind)
		}
	}

	assert.Contains(t, declined, "NamedResult",
		"a comparison converted to a named boolean cannot take a helper returning bool")
	assert.Contains(t, declined, "NamedConnector",
		"the connector helper's thunk is spelled func() bool, which a named boolean is not")
	assert.Contains(t, declined, "*Bucket.Hit",
		"m[k]++ is legal Go but &m[k] is not, so the pointer helper cannot be used")
	assert.Contains(t, declined, "Generic",
		"a type parameter constrained by int|string does not satisfy the ordered helper")

	// Negating a boolean value yields an *untyped* boolean, which stays assignable to
	// a named boolean type. That is the whole reason equality and literal flips need
	// no type analysis where the comparison helpers do.
	for _, m := range mutants {
		switch m.Function {
		case "Same", "Differs", "Disabled", "Enabled":
			assert.True(t, m.Schemata(),
				"%s in %s should be rewritten by negation, not declined", m.Kind, m.Function)
		}
	}
}

func TestSchemataSkipsConstantDeclarations(t *testing.T) {
	mutants := generateSchemataFixture(t, writeSchemataFixture(t))

	for _, m := range mutants {
		if m.Function != "" {
			continue
		}
		assert.False(t, m.Schemata(),
			"a mutant in a const declaration cannot become a function call: %s at %s", m.Kind, m.Location())
	}
}

func TestSchemataRuntimeIsFormatted(t *testing.T) {
	helpers := make([]string, 0, len(helperSource))
	for name := range helperSource {
		helpers = append(helpers, name)
	}

	source := SchemataSource("target", helpers)
	formatted, err := format.Source(source)
	require.NoError(t, err, "the injected runtime must at least parse")
	assert.Equal(t, string(formatted), string(source),
		"the injected runtime is read by humans when a build fails; keep it gofmt-clean")
}

func TestSchemataRuntimeOmitsUnusedHelpers(t *testing.T) {
	source := string(SchemataSource("target", []string{helperOn}))

	assert.Contains(t, source, "func "+helperOn+"(")
	assert.NotContains(t, source, helperAddSub)
	assert.NotContains(t, source, constraintNum,
		"a constraint no helper mentions is dead weight in every compile")
}

func form(from, to int, spans ...schemaSpan) *schemaForm {
	if len(spans) == 0 {
		spans = []schemaSpan{{from: from, to: to}}
	}

	return &schemaForm{
		from:  from,
		to:    to,
		spans: spans,
		render: func(id int, parts []string) string {
			return "<" + strconv.Itoa(id) + ":" + strings.Join(parts, "|") + ">"
		},
	}
}

func TestRenderSchemataNestsInnerSites(t *testing.T) {
	source := []byte("return a + b < c")

	outer := &schemaSite{form: form(7, 16, schemaSpan{from: 7, to: 12}, schemaSpan{from: 15, to: 16}), id: 1}
	inner := &schemaSite{form: form(7, 12, schemaSpan{from: 7, to: 8}, schemaSpan{from: 11, to: 12}), id: 2}

	out, err := renderSchemata(source, []*schemaSite{outer, inner})
	require.NoError(t, err)

	assert.Equal(t, "return <1:<2:a|b>|c>", string(out),
		"the comparison contains the addition, so expanding it must expand the addition too")
}

func TestRenderSchemataNestsSitesWithIdenticalRanges(t *testing.T) {
	source := []byte("if ok {")

	first := &schemaSite{form: form(3, 5), id: 1}
	second := &schemaSite{form: form(3, 5), id: 2}

	out, err := renderSchemata(source, []*schemaSite{first, second})
	require.NoError(t, err)

	assert.Equal(t, "if <1:<2:ok>> {", string(out),
		"forcing a branch both ways rewrites one range twice, which has to compose")
}

func TestRenderSchemataKeepsSiblingsSideBySide(t *testing.T) {
	source := []byte("f(a, b)")

	left := &schemaSite{form: form(2, 3), id: 1}
	right := &schemaSite{form: form(5, 6), id: 2}

	out, err := renderSchemata(source, []*schemaSite{left, right})
	require.NoError(t, err)

	assert.Equal(t, "f(<1:a>, <2:b>)", string(out))
}

func TestRenderSchemataRefusesPartialOverlap(t *testing.T) {
	source := []byte("aaaaaaaaaa")

	_, err := renderSchemata(source, []*schemaSite{
		{form: form(0, 5), id: 1},
		{form: form(3, 8), id: 2},
	})

	require.Error(t, err,
		"a rewrite that straddles another cannot be rendered, and guessing would emit code that does not compile")
}

// TestRenderSchemataRefusesToDropANestedSite guards the case that would be worst
// if it passed silently: a mutant that never reaches the binary but is still
// scored against it.
func TestRenderSchemataRefusesToDropANestedSite(t *testing.T) {
	source := []byte("a + b")

	// The parent re-renders only its first operand, so a site in the second would
	// vanish.
	parent := form(0, 5, schemaSpan{from: 0, to: 1})

	_, err := renderSchemata(source, []*schemaSite{
		{form: parent, id: 1},
		{form: form(4, 5), id: 2},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "falls outside every span")
}
