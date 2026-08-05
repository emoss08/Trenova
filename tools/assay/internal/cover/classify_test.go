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
