package cover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sample = `package demo

import "fmt"

const Limit = 10

type Config struct {
	Name string
	Size int
}

// Run does a thing.
func Run(c Config) int {
	total := c.Size
	if total > Limit {
		total = Limit
	}
	fmt.Println(total)

	return total
}

var handler = func(n int) int {
	doubled := n * 2

	return doubled
}

//go:generate stringer -type=Kind
type Kind int

func OneLiner() int { return 1 }
`

func lineOf(t *testing.T, needle string) int {
	t.Helper()

	for i, line := range strings.Split(sample, "\n") {
		if strings.Contains(line, needle) {
			return i + 1
		}
	}
	t.Fatalf("needle %q not found in sample", needle)

	return 0
}

func classify(t *testing.T, lines ...int) FileClassification {
	t.Helper()

	got, err := classifySource("demo.go", []byte(sample), lines)
	require.NoError(t, err)

	return got
}

func TestFunctionBodyLinesAreExecutable(t *testing.T) {
	got := classify(t, lineOf(t, "total := c.Size"), lineOf(t, "total = Limit"), lineOf(t, "return total"))

	assert.True(t, got.Narrowable())
	assert.Len(t, got.Executable, 3)
	assert.Empty(t, got.Declaration)
}

func TestFuncLitBodyLinesAreExecutable(t *testing.T) {
	got := classify(t, lineOf(t, "doubled := n * 2"))

	assert.True(t, got.Narrowable(), "a package-level func literal body is executable code")
	assert.Equal(t, []int{lineOf(t, "doubled := n * 2")}, got.Executable)
}

func TestStructFieldChangeIsNotNarrowable(t *testing.T) {
	got := classify(t, lineOf(t, "Size int"))

	assert.False(t, got.Narrowable(),
		"a struct field never appears in a coverage block, so it must force a full run")
	assert.Equal(t, []int{lineOf(t, "Size int")}, got.Declaration)
}

func TestConstChangeIsNotNarrowable(t *testing.T) {
	got := classify(t, lineOf(t, "const Limit"))

	assert.False(t, got.Narrowable())
}

func TestImportChangeIsNotNarrowable(t *testing.T) {
	got := classify(t, lineOf(t, `import "fmt"`))

	assert.False(t, got.Narrowable())
}

func TestFunctionSignatureChangeIsNotNarrowable(t *testing.T) {
	got := classify(t, lineOf(t, "func Run(c Config) int {"))

	assert.False(t, got.Narrowable(),
		"the signature line sits outside the body, so a signature change forces a full run")
}

func TestOneLineFunctionIsNotNarrowable(t *testing.T) {
	got := classify(t, lineOf(t, "func OneLiner()"))

	assert.False(t, got.Narrowable(),
		"a body that shares its line with the braces cannot be attributed safely")
}

func TestPlainCommentIsIgnorable(t *testing.T) {
	got := classify(t, lineOf(t, "// Run does a thing."))

	assert.True(t, got.Narrowable())
	assert.Empty(t, got.Executable)
	assert.Empty(t, got.Declaration, "a plain comment cannot change behaviour")
}

func TestGoDirectiveIsNotIgnorable(t *testing.T) {
	got := classify(t, lineOf(t, "//go:generate"))

	assert.False(t, got.Narrowable(),
		"//go: directives change build or generation behaviour and must not be skipped")
}

func TestBlankLineIsIgnorable(t *testing.T) {
	got := classify(t, 2)

	assert.True(t, got.Narrowable())
	assert.Empty(t, got.Executable)
	assert.Empty(t, got.Declaration)
}

func TestMixedChangeFallsBackWhenAnyLineIsDeclaration(t *testing.T) {
	got := classify(t, lineOf(t, "total := c.Size"), lineOf(t, "Size int"))

	assert.False(t, got.Narrowable(),
		"one unattributable line poisons the whole file")
}

func TestLineBeyondEndOfFileIsDeclaration(t *testing.T) {
	got := classify(t, 10_000)

	assert.False(t, got.Narrowable())
}

func TestZeroAndNegativeLinesAreDeclaration(t *testing.T) {
	got := classify(t, 0)

	assert.False(t, got.Narrowable())
}

func TestClassifyFileReadsFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.go")
	require.NoError(t, os.WriteFile(path, []byte(sample), 0o644))

	got, err := ClassifyFile(path, []int{lineOf(t, "total := c.Size")})
	require.NoError(t, err)

	assert.True(t, got.Narrowable())
	assert.Equal(t, path, got.Path)
}

func TestClassifyFileFailsOnMissingFile(t *testing.T) {
	_, err := ClassifyFile(filepath.Join(t.TempDir(), "absent.go"), []int{1})

	require.Error(t, err)
}

func TestClassifyFailsOnUnparseableSource(t *testing.T) {
	_, err := classifySource("broken.go", []byte("package demo\n\nfunc ("), []int{1})

	require.Error(t, err, "a file we cannot parse must be an error, never a narrowing decision")
}

func TestLineKindString(t *testing.T) {
	assert.Equal(t, "ignorable", LineIgnorable.String())
	assert.Equal(t, "in-function", LineInFunction.String())
	assert.Equal(t, "declaration", LineDeclaration.String())
}

// The tests below exist because `assay mutate` found them missing: each one kills
// a mutant that survived a run over this package.

const boundarySample = `package demo

func First() int {
	total := 1

	return total
}

const AfterFirst = 5

func External() int

func Last() int {
	return 2
}
`

func classifyBoundary(t *testing.T, lines ...int) FileClassification {
	t.Helper()

	got, err := classifySource("boundary.go", []byte(boundarySample), lines)
	require.NoError(t, err)

	return got
}

func boundaryLineOf(t *testing.T, needle string) int {
	t.Helper()

	for i, line := range strings.Split(boundarySample, "\n") {
		if strings.Contains(line, needle) {
			return i + 1
		}
	}
	t.Fatalf("needle %q not found", needle)

	return 0
}

func TestLineAfterClosingBraceIsNotExecutable(t *testing.T) {
	closing := boundaryLineOf(t, "return total") + 1

	assert.False(t, classifyBoundary(t, closing).Narrowable(),
		"the closing brace line is outside the body; treating it as executable would let "+
			"narrowing trust a declaration")
	assert.False(t, classifyBoundary(t, boundaryLineOf(t, "const AfterFirst")).Narrowable(),
		"a declaration following a body must stay a declaration")
}

func TestFirstAndLastBodyLinesAreExecutable(t *testing.T) {
	assert.True(t, classifyBoundary(t, boundaryLineOf(t, "total := 1")).Narrowable(),
		"the first statement of a body is executable")
	assert.True(t, classifyBoundary(t, boundaryLineOf(t, "return total")).Narrowable(),
		"the last statement of a body is executable")
}

func TestBodylessFunctionDeclarationIsHandled(t *testing.T) {
	got := classifyBoundary(t, boundaryLineOf(t, "func External() int"))

	assert.False(t, got.Narrowable(),
		"a declaration with no body has nothing executable and must not panic")
}

func TestTrailingCommentOnTheLastLineIsIgnorable(t *testing.T) {
	source := "package demo\n\nfunc F() int {\n\treturn 1\n}\n\n// trailing"
	got, err := classifySource("trailing.go", []byte(source), []int{7})
	require.NoError(t, err)

	assert.True(t, got.Narrowable())
	assert.Empty(t, got.Declaration,
		"the ignorable-text check must cover the final line of the file")
}

func TestExecutableRangesReportsBodySpans(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "boundary.go")
	require.NoError(t, os.WriteFile(path, []byte(boundarySample), 0o644))

	ranges, err := ExecutableRanges(path)
	require.NoError(t, err)
	require.NotEmpty(t, ranges)

	first := boundaryLineOf(t, "total := 1")
	closing := boundaryLineOf(t, "return total") + 1

	var covered, leaked bool
	for _, span := range ranges {
		if span.Contains(first) {
			covered = true
		}
		if span.Contains(closing) || span.Contains(boundaryLineOf(t, "const AfterFirst")) {
			leaked = true
		}
	}
	assert.True(t, covered, "the body's statements must be reported")
	assert.False(t, leaked, "the span must stop before the closing brace")
}

// TestMinimalBodyIsNarrowable pins the boundary in functionBodyRanges, which
// mutation testing caught the moment ExecutableRanges stopped calling it: a
// function holding exactly one statement has two lines between its braces, and
// treating that as too small to be a body would send narrowing back to the whole
// package for the commonest shape in Go. Nothing here asserted it before, because
// the existing tests check that narrowing is *sound*, never that it is tight.
func TestMinimalBodyIsNarrowable(t *testing.T) {
	source := "package demo\n\nfunc F() int {\n\treturn 1\n}\n"

	got, err := classifySource("minimal.go", []byte(source), []int{4})
	require.NoError(t, err)

	assert.Equal(t, []int{4}, got.Executable)
	assert.True(t, got.Narrowable())
}

func TestBodyWithNoRoomForAStatementIsNotExecutable(t *testing.T) {
	source := "package demo\n\nfunc F() {\n}\n"

	got, err := classifySource("empty.go", []byte(source), []int{3, 4})
	require.NoError(t, err)

	assert.Empty(t, got.Executable,
		"there is no line between the braces, so there is nothing coverage could attribute")
}

// TestExecutableRangesCoversSingleLineBodies pins the difference between what
// mutation may touch and what narrowing may narrow. Coverage attributes a one-line
// body to the function's own line, so refusing to mutate it would leave every
// accessor and one-line helper in a Go codebase untested by the mutation engine.
func TestExecutableRangesCoversSingleLineBodies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oneline.go")
	source := "package demo\n\nconst Limit = 3\n\nfunc Add(a, b int) int { return a + b }\n\nfunc Empty() {}\n"
	require.NoError(t, os.WriteFile(path, []byte(source), 0o644))

	ranges, err := ExecutableRanges(path)
	require.NoError(t, err)

	assert.True(t, containsLine(ranges, 5), "the body of a one-line function is executable")
	assert.False(t, containsLine(ranges, 3), "a const declaration is not")
	assert.False(t, containsLine(ranges, 7), "an empty body has no statement to mutate")
}

func TestExecutableRangesStopsAtTheClosingBrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multiline.go")
	source := "package demo\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n\nvar X = 1\n"
	require.NoError(t, os.WriteFile(path, []byte(source), 0o644))

	ranges, err := ExecutableRanges(path)
	require.NoError(t, err)

	assert.True(t, containsLine(ranges, 4))
	assert.False(t, containsLine(ranges, 3), "the signature line carries no statement")
	assert.False(t, containsLine(ranges, 5), "nor does the closing brace")
	assert.False(t, containsLine(ranges, 7))
}

func containsLine(ranges []Block, line int) bool {
	for _, span := range ranges {
		if span.Contains(line) {
			return true
		}
	}

	return false
}

func TestExecutableRangesFailsOnMissingFile(t *testing.T) {
	_, err := ExecutableRanges(filepath.Join(t.TempDir(), "absent.go"))

	require.Error(t, err)
}

func TestExecutableRangesFailsOnUnparseableSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.go")
	require.NoError(t, os.WriteFile(path, []byte("package demo\n\nfunc ("), 0o644))

	_, err := ExecutableRanges(path)

	require.Error(t, err, "an unparseable file must not silently report no executable lines")
}

func TestExecutableRangesOnFileWithNoFunctions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "decls.go")
	require.NoError(t, os.WriteFile(path,
		[]byte("package demo\n\nconst A = 1\n\ntype B struct{ C int }\n"), 0o644))

	ranges, err := ExecutableRanges(path)

	require.NoError(t, err)
	assert.Empty(t, ranges,
		"a file of pure declarations has no executable span, and the empty case must not "+
			"reach the merge, which indexes blocks[:1]")
}
