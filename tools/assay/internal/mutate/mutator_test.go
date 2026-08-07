package mutate_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/mutate"
)

const fixtureModule = "example.com/mut"

const fixtureSource = `package target

// Threshold is documented, and this comment must survive mutation.
const Threshold = 10

type Counter struct {
	Total int
}

//go:noinline
func Sum(a, b int) int {
	return a + b
}

func Concat(a, b string) string {
	return a + b
}

func Scale(a, b int) int {
	return a * b
}

func Below(n int) bool {
	return n < Threshold
}

func Same(a, b int) bool {
	return a == b
}

func Both(a, b bool) bool {
	return a && b
}

func Always() bool {
	return true
}

func Shadowed(flag bool) bool {
	true := flag

	return true
}

func (c *Counter) Bump() {
	c.Total++
}

func Guard(n int) int {
	if n > Threshold {
		return Threshold
	}

	return n
}
`

func writeFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte("module "+fixtureModule+"\n\ngo 1.26\n"), 0o644))

	dir := filepath.Join(root, "target")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "target.go"), []byte(fixtureSource), 0o644))

	return root
}

func generate(t *testing.T, root string) []mutate.Mutant {
	t.Helper()
	t.Setenv("GOWORK", "off")

	mutants, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:    root,
		Package: fixtureModule + "/target",
		Env:     append(os.Environ(), "GOWORK=off"),
	})
	require.NoError(t, err)

	return mutants
}

func requireCompiles(t *testing.T, root string, m mutate.Mutant) {
	t.Helper()

	dir := t.TempDir()
	replacement := filepath.Join(dir, "mutant.go")
	require.NoError(t, os.WriteFile(replacement, m.Source(), 0o644))

	overlay := filepath.Join(dir, "overlay.json")
	payload, err := json.Marshal(map[string]map[string]string{
		"Replace": {m.File: replacement},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(overlay, payload, 0o644))

	cmd := exec.Command("go", "build", "-overlay="+overlay, "-o", filepath.Join(dir, "out.a"),
		fixtureModule+"/target")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")

	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "mutant %s (%s at %s) does not compile:\n%s\n--- source ---\n%s",
		m.ID, m.Kind, m.Location(), out, m.Source())
}

func TestEveryMutantCompiles(t *testing.T) {
	root := writeFixture(t)
	mutants := generate(t, root)

	require.NotEmpty(t, mutants)
	for _, m := range mutants {
		t.Run(string(m.Kind)+"/"+m.Location()+"/"+m.ID, func(t *testing.T) {
			requireCompiles(t, root, m)
		})
	}
}

func kindsPresent(mutants []mutate.Mutant) map[mutate.Kind]int {
	out := make(map[mutate.Kind]int)
	for _, m := range mutants {
		out[m.Kind]++
	}

	return out
}

func TestAllMutatorKindsFire(t *testing.T) {
	mutants := generate(t, writeFixture(t))
	present := kindsPresent(mutants)

	for _, kind := range mutate.Kinds() {
		assert.Positive(t, present[kind], "no mutants of kind %s were generated", kind)
	}
}

func functionsMutated(mutants []mutate.Mutant, kind mutate.Kind) []string {
	var out []string
	for _, m := range mutants {
		if m.Kind == kind {
			out = append(out, m.Function)
		}
	}

	return out
}

func TestStringConcatenationIsNotTreatedAsArithmetic(t *testing.T) {
	mutants := generate(t, writeFixture(t))

	assert.NotContains(t, functionsMutated(mutants, mutate.KindArithmetic), "Concat",
		"swapping + for - on strings would not compile")
	assert.Contains(t, functionsMutated(mutants, mutate.KindArithmetic), "Sum")
}

func TestShadowedBooleanIsNotFlipped(t *testing.T) {
	mutants := generate(t, writeFixture(t))

	flipped := functionsMutated(mutants, mutate.KindBoolean)
	assert.Contains(t, flipped, "Always")
	assert.NotContains(t, flipped, "Shadowed",
		"Shadowed contains no universe literal; its `true` is a local variable")
}

func TestBranchMutantsForceBothDirections(t *testing.T) {
	mutants := generate(t, writeFixture(t))

	var replacements []string
	for _, m := range mutants {
		if m.Kind == mutate.KindBranch {
			replacements = append(replacements, m.Replacement)
		}
	}

	require.Len(t, replacements, 2)
	assert.Contains(t, strings.Join(replacements, " "), "&& false")
	assert.Contains(t, strings.Join(replacements, " "), "|| true")
}

func TestMethodReceiverAppearsInFunctionName(t *testing.T) {
	mutants := generate(t, writeFixture(t))

	assert.Contains(t, functionsMutated(mutants, mutate.KindIncDec), "*Counter.Bump")
}

func TestMutantSourcePreservesCommentsAndDirectives(t *testing.T) {
	mutants := generate(t, writeFixture(t))
	require.NotEmpty(t, mutants)

	source := string(mutants[0].Source())
	assert.Contains(t, source, "// Threshold is documented",
		"splicing bytes must not drop comments the way reprinting the AST would")
	assert.Contains(t, source, "//go:noinline",
		"losing a //go: directive could change how the mutant compiles")
}

func TestMutantSourceAppliesExactlyOneChange(t *testing.T) {
	root := writeFixture(t)
	mutants := generate(t, root)

	for _, m := range mutants {
		if m.Kind != mutate.KindArithmetic || m.Function != "Sum" {
			continue
		}
		source := string(m.Source())
		assert.Contains(t, source, "func Sum(a, b int) int {\n\treturn a - b\n}")
		assert.Contains(t, source, "func Concat(a, b string) string {\n\treturn a + b\n}",
			"a mutant must change exactly one site and leave every other alone")

		return
	}
	t.Fatal("no arithmetic mutant found in Sum")
}

func TestMutantIDsAreUniqueAndStable(t *testing.T) {
	root := writeFixture(t)

	first := generate(t, root)
	second := generate(t, root)

	require.Len(t, second, len(first))

	seen := make(map[string]struct{}, len(first))
	for i, m := range first {
		_, dup := seen[m.ID]
		assert.False(t, dup, "duplicate mutant ID %s", m.ID)
		seen[m.ID] = struct{}{}
		assert.Equal(t, m.ID, second[i].ID, "IDs must be stable across runs")
	}
}

func TestMutantIDSurvivesLineShift(t *testing.T) {
	root := writeFixture(t)
	before := generate(t, root)

	path := filepath.Join(root, "target", "target.go")
	require.NoError(t, os.WriteFile(path,
		[]byte("// a new leading comment\n// and another\n"+fixtureSource), 0o644))

	after := generate(t, root)

	require.Len(t, after, len(before))
	for i := range before {
		assert.Equal(t, before[i].ID, after[i].ID,
			"IDs exclude line numbers so unrelated edits above must not churn the baseline")
		assert.NotEqual(t, before[i].Line, after[i].Line, "the line really did shift")
	}
}

func TestEligibleFilterSuppressesSites(t *testing.T) {
	root := writeFixture(t)
	t.Setenv("GOWORK", "off")

	all, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:    root,
		Package: fixtureModule + "/target",
		Env:     append(os.Environ(), "GOWORK=off"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, all)

	target := all[0].Line
	filtered, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:     root,
		Package:  fixtureModule + "/target",
		Env:      append(os.Environ(), "GOWORK=off"),
		Eligible: func(_ string, line int) bool { return line == target },
	})
	require.NoError(t, err)

	assert.NotEmpty(t, filtered)
	assert.Less(t, len(filtered), len(all))
	for _, m := range filtered {
		assert.Equal(t, target, m.Line)
	}
}

func TestGenerateRefusesBrokenPackage(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module "+fixtureModule+"\n\ngo 1.26\n"), 0o644))
	dir := filepath.Join(root, "target")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "target.go"),
		[]byte("package target\n\nfunc Broken() int { return undefinedThing }\n"), 0o644))

	t.Setenv("GOWORK", "off")
	_, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:    root,
		Package: fixtureModule + "/target",
		Env:     append(os.Environ(), "GOWORK=off"),
	})

	require.Error(t, err, "mutating a package that does not compile would produce noise, not signal")
}

func TestTestFilesAreNotMutated(t *testing.T) {
	root := writeFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "target", "target_test.go"),
		[]byte("package target\n\nimport \"testing\"\n\nfunc TestSum(t *testing.T) {\n\tif Sum(1, 2) != 3 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n"),
		0o644))

	for _, m := range generate(t, root) {
		assert.NotContains(t, m.File, "_test.go",
			"mutating the tests themselves proves nothing")
	}
}
