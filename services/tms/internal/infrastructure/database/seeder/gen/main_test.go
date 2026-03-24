package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToConstName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"USStates", "SeedUSStates"},
		{"TestOrganizations", "SeedTestOrganizations"},
		{"test-org", "SeedTestOrg"},
		{"user_data", "SeedUserData"},
		{"", "Seed"},
		{"123", "Seed123"},
		{"test123data", "SeedTest123data"},
		{"ALLCAPS", "SeedALLCAPS"},
		{"mixedCase", "SeedMixedCase"},
		{"with spaces", "SeedWithSpaces"},
		{"with.dots", "SeedWithDots"},
		{"multiple--dashes", "SeedMultipleDashes"},
		{"under__scores", "SeedUnderScores"},
		{"CamelCase", "SeedCamelCase"},
		{"a", "SeedA"},
		{"A", "SeedA"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := toConstName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseFile_ValidSeed(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "test_seed.go")

	content := `package seeds

import "github.com/emoss08/trenova/pkg/seedhelpers"

type TestSeed struct {
    seedhelpers.BaseSeed
}

func NewTestSeed() *TestSeed {
    seed := &TestSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(
        "TestData",
        "1.0.0",
        "Test seed for data",
        nil,
    )
    return seed
}
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	seeds, err := parseFile(seedFile, "base")
	require.NoError(t, err)
	require.Len(t, seeds, 1)

	assert.Equal(t, "TestData", seeds[0].Name)
	assert.Equal(t, "SeedTestData", seeds[0].ConstName)
	assert.Equal(t, "test_seed.go", seeds[0].SourceFile)
	assert.Equal(t, "base", seeds[0].Directory)
}

func TestParseFile_MultipleSeeds(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "multi_seed.go")

	content := `package seeds

import "github.com/emoss08/trenova/pkg/seedhelpers"

func NewSeed1() *Seed1 {
    seed := &Seed1{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("First", "1.0.0", "First seed", nil)
    return seed
}

func NewSeed2() *Seed2 {
    seed := &Seed2{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("Second", "1.0.0", "Second seed", nil)
    return seed
}
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	seeds, err := parseFile(seedFile, "development")
	require.NoError(t, err)
	require.Len(t, seeds, 2)

	names := []string{seeds[0].Name, seeds[1].Name}
	assert.Contains(t, names, "First")
	assert.Contains(t, names, "Second")
}

func TestParseFile_NoSeeds(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "no_seeds.go")

	content := `package seeds

type NotASeed struct {
    Name string
}

func NewNotASeed() *NotASeed {
    return &NotASeed{Name: "test"}
}
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	seeds, err := parseFile(seedFile, "base")
	require.NoError(t, err)
	assert.Empty(t, seeds)
}

func TestParseFile_NonStringArg(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "non_string.go")

	content := `package seeds

import "github.com/emoss08/trenova/pkg/seedhelpers"

const seedName = "DynamicName"

func NewDynamicSeed() *DynamicSeed {
    seed := &DynamicSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(seedName, "1.0.0", "Dynamic", nil)
    return seed
}
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	seeds, err := parseFile(seedFile, "base")
	require.NoError(t, err)
	assert.Empty(t, seeds)
}

func TestParseFile_SyntaxError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "syntax_error.go")

	content := `package seeds

func broken( {
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	_, err = parseFile(seedFile, "base")
	require.Error(t, err)
}

func TestFindSeeds_Directory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	baseDir := filepath.Join(tmpDir, "base")
	devDir := filepath.Join(tmpDir, "development")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))
	require.NoError(t, os.MkdirAll(devDir, 0o755))

	baseSeed := `package base
import "github.com/emoss08/trenova/pkg/seedhelpers"
func NewBaseSeed1() *BaseSeed1 {
    seed := &BaseSeed1{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("BaseOne", "1.0.0", "Base seed", nil)
    return seed
}
`
	devSeed := `package development
import "github.com/emoss08/trenova/pkg/seedhelpers"
func NewDevSeed1() *DevSeed1 {
    seed := &DevSeed1{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("DevOne", "1.0.0", "Dev seed", nil)
    return seed
}
`

	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "01_base.go"), []byte(baseSeed), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(devDir, "01_dev.go"), []byte(devSeed), 0o644))

	seeds, err := findSeeds(tmpDir)
	require.NoError(t, err)
	require.Len(t, seeds, 2)

	var baseFound, devFound bool
	for _, s := range seeds {
		if s.Name == "BaseOne" {
			baseFound = true
			assert.Equal(t, "base", s.Directory)
		}
		if s.Name == "DevOne" {
			devFound = true
			assert.Equal(t, "development", s.Directory)
		}
	}
	assert.True(t, baseFound, "BaseOne seed not found")
	assert.True(t, devFound, "DevOne seed not found")
}

func TestFindSeeds_SkipTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	realSeed := `package base
import "github.com/emoss08/trenova/pkg/seedhelpers"
func NewRealSeed() *RealSeed {
    seed := &RealSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("RealSeed", "1.0.0", "Real", nil)
    return seed
}
`
	testSeed := `package base
import "github.com/emoss08/trenova/pkg/seedhelpers"
func TestHelper() {
    seedhelpers.NewBaseSeed("TestSeed", "1.0.0", "Test", nil)
}
`

	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "real.go"), []byte(realSeed), 0o644))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(baseDir, "real_test.go"), []byte(testSeed), 0o644),
	)

	seeds, err := findSeeds(tmpDir)
	require.NoError(t, err)
	require.Len(t, seeds, 1)
	assert.Equal(t, "RealSeed", seeds[0].Name)
}

func TestFindSeeds_SkipGenFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	realSeed := `package base
import "github.com/emoss08/trenova/pkg/seedhelpers"
func NewRealSeed() *RealSeed {
    seed := &RealSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("RealSeed", "1.0.0", "Real", nil)
    return seed
}
`
	genFile := `package base
import "github.com/emoss08/trenova/pkg/seedhelpers"
func Generated() {
    seedhelpers.NewBaseSeed("GenSeed", "1.0.0", "Gen", nil)
}
`

	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "real.go"), []byte(realSeed), 0o644))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(baseDir, "seed_ids_gen.go"), []byte(genFile), 0o644),
	)

	seeds, err := findSeeds(tmpDir)
	require.NoError(t, err)
	require.Len(t, seeds, 1)
	assert.Equal(t, "RealSeed", seeds[0].Name)
}

func TestFindSeeds_Sorting(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	seeds := []struct {
		file string
		name string
	}{
		{"c_seed.go", "Zebra"},
		{"a_seed.go", "Alpha"},
		{"b_seed.go", "Beta"},
	}

	for _, s := range seeds {
		content := `package base
import "github.com/emoss08/trenova/pkg/seedhelpers"
func New` + s.name + `Seed() *` + s.name + `Seed {
    seed := &` + s.name + `Seed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("` + s.name + `", "1.0.0", "` + s.name + `", nil)
    return seed
}
`
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, s.file), []byte(content), 0o644))
	}

	result, err := findSeeds(tmpDir)
	require.NoError(t, err)
	require.Len(t, result, 3)

	assert.Equal(t, "Alpha", result[0].Name)
	assert.Equal(t, "Beta", result[1].Name)
	assert.Equal(t, "Zebra", result[2].Name)
}

func TestFindSeeds_EmptyDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	seeds, err := findSeeds(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, seeds)
}

func TestGenerateFile_ValidOutput(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "seed_ids_gen.go")

	seeds := []SeedInfo{
		{
			Name:       "USStates",
			ConstName:  "SeedUSStates",
			SourceFile: "01_states.go",
			Directory:  "base",
		},
		{
			Name:       "TestOrg",
			ConstName:  "SeedTestOrg",
			SourceFile: "01_org.go",
			Directory:  "development",
		},
	}

	err := generateFile(seeds, outputFile, "seedhelpers")
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "package seedhelpers")
	assert.Contains(t, contentStr, "SeedUSStates SeedID = \"USStates\"")
	assert.Contains(t, contentStr, "SeedTestOrg SeedID = \"TestOrg\"")
	assert.Contains(t, contentStr, "Code generated by seedgen")
	assert.Contains(t, contentStr, "func ValidateSeedID")
}

func TestGenerateFile_Categorization(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.go")

	seeds := []SeedInfo{
		{Name: "Base1", ConstName: "SeedBase1", SourceFile: "base1.go", Directory: "base"},
		{Name: "Base2", ConstName: "SeedBase2", SourceFile: "base2.go", Directory: "base"},
		{Name: "Dev1", ConstName: "SeedDev1", SourceFile: "dev1.go", Directory: "development"},
		{Name: "Test1", ConstName: "SeedTest1", SourceFile: "test1.go", Directory: "test"},
		{
			Name:       "Testing1",
			ConstName:  "SeedTesting1",
			SourceFile: "testing1.go",
			Directory:  "testing",
		},
	}

	err := generateFile(seeds, outputFile, "test")
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)

	baseSeedsSection := extractSection(contentStr, "var BaseSeedIDs", "}")
	assert.Contains(t, baseSeedsSection, "SeedBase1")
	assert.Contains(t, baseSeedsSection, "SeedBase2")
	assert.NotContains(t, baseSeedsSection, "SeedDev1")
	assert.NotContains(t, baseSeedsSection, "SeedTest1")

	devSeedsSection := extractSection(contentStr, "var DevelopmentSeedIDs", "}")
	assert.Contains(t, devSeedsSection, "SeedDev1")
	assert.NotContains(t, devSeedsSection, "SeedBase1")

	testSeedsSection := extractSection(contentStr, "var TestSeedIDs", "}")
	assert.Contains(t, testSeedsSection, "SeedTest1")
	assert.Contains(t, testSeedsSection, "SeedTesting1")
}

func TestGenerateFile_EmptySeeds(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.go")

	var seeds []SeedInfo

	err := generateFile(seeds, outputFile, "empty")
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "package empty")
	assert.Contains(t, contentStr, "var AllSeedIDs = []SeedID{")
	assert.Contains(t, contentStr, "var BaseSeedIDs = []SeedID{")
}

func TestGenerateFile_Determinism(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	seeds := []SeedInfo{
		{Name: "Bravo", ConstName: "SeedBravo", SourceFile: "b.go", Directory: "base"},
		{Name: "Alpha", ConstName: "SeedAlpha", SourceFile: "a.go", Directory: "development"},
		{Name: "Charlie", ConstName: "SeedCharlie", SourceFile: "c.go", Directory: "test"},
	}

	var outputs []string
	for range 5 {
		outputFile := filepath.Join(tmpDir, "output.go")
		err := generateFile(seeds, outputFile, "test")
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		outputs = append(outputs, string(content))
	}

	for i := 1; i < len(outputs); i++ {
		assert.Equal(t, outputs[0], outputs[i], "run %d differs from run 0", i)
	}
}

func TestSeedInfo_Structure(t *testing.T) {
	t.Parallel()

	info := SeedInfo{
		Name:       "TestSeed",
		ConstName:  "SeedTestSeed",
		SourceFile: "test_seed.go",
		Directory:  "base",
	}

	assert.Equal(t, "TestSeed", info.Name)
	assert.Equal(t, "SeedTestSeed", info.ConstName)
	assert.Equal(t, "test_seed.go", info.SourceFile)
	assert.Equal(t, "base", info.Directory)
}

func extractSection(content, start, end string) string {
	startIdx := strings.Index(content, start)
	if startIdx == -1 {
		return ""
	}
	remaining := content[startIdx:]
	endIdx := strings.Index(remaining, end)
	if endIdx == -1 {
		return remaining
	}
	return remaining[:endIdx+len(end)]
}

func TestFindSeeds_NonExistentDirectory(t *testing.T) {
	t.Parallel()

	_, err := findSeeds("/nonexistent/path/that/does/not/exist")
	require.Error(t, err)
}

func TestFindSeeds_IgnoresNonGoFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "readme.md"), []byte("# Hello"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "data.json"), []byte("{}"), 0o644))

	goSeed := `package base
import "github.com/emoss08/trenova/pkg/seedhelpers"
func NewSeed() *Seed {
    seed := &Seed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("GoSeed", "1.0.0", "Go seed", nil)
    return seed
}
`
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "seed.go"), []byte(goSeed), 0o644))

	seeds, err := findSeeds(tmpDir)
	require.NoError(t, err)
	require.Len(t, seeds, 1)
	assert.Equal(t, "GoSeed", seeds[0].Name)
}

func TestParseFile_NoArgs(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "no_args.go")

	content := `package seeds

import "github.com/emoss08/trenova/pkg/seedhelpers"

func NewBadSeed() *BadSeed {
    seed := &BadSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed()
    return seed
}
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	seeds, err := parseFile(seedFile, "base")
	require.NoError(t, err)
	assert.Empty(t, seeds)
}

func TestParseFile_NonSelectorFunction(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "non_selector.go")

	content := `package seeds

func NewBaseSeed(name string) string {
    return name
}

func Init() {
    NewBaseSeed("test")
}
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	seeds, err := parseFile(seedFile, "base")
	require.NoError(t, err)
	assert.Empty(t, seeds)
}

func TestParseFile_DifferentSelectorName(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "different_selector.go")

	content := `package seeds

import "github.com/emoss08/trenova/pkg/seedhelpers"

func NewSeed() *Seed {
    seed := &Seed{}
    seed.BaseSeed = *seedhelpers.NewDifferentFunction("test", "1.0.0", "desc", nil)
    return seed
}
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	seeds, err := parseFile(seedFile, "base")
	require.NoError(t, err)
	assert.Empty(t, seeds)
}

func TestGenerateFile_InvalidPath(t *testing.T) {
	t.Parallel()

	seeds := []SeedInfo{
		{Name: "Test", ConstName: "SeedTest", SourceFile: "test.go", Directory: "base"},
	}

	err := generateFile(seeds, "/nonexistent/dir/output.go", "test")
	require.Error(t, err)
}

func TestGenerateFile_TestingDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.go")

	seeds := []SeedInfo{
		{Name: "TestSeed", ConstName: "SeedTestSeed", SourceFile: "seed.go", Directory: "testing"},
	}

	err := generateFile(seeds, outputFile, "pkg")
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	testSection := extractSection(contentStr, "var TestSeedIDs", "}")
	assert.Contains(t, testSection, "SeedTestSeed")
}

func TestFindSeeds_WithNestedDirectories(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "base", "nested")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))

	seed := `package nested
import "github.com/emoss08/trenova/pkg/seedhelpers"
func NewNestedSeed() *NestedSeed {
    seed := &NestedSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed("NestedSeed", "1.0.0", "Nested", nil)
    return seed
}
`
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "nested.go"), []byte(seed), 0o644))

	seeds, err := findSeeds(tmpDir)
	require.NoError(t, err)
	require.Len(t, seeds, 1)
	assert.Equal(t, "NestedSeed", seeds[0].Name)
	assert.Equal(t, "nested", seeds[0].Directory)
}

func TestToConstName_SpecialCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"!@#$%", "Seed"},
		{"a-b-c-d", "SeedABCD"},
		{"123abc", "Seed123abc"},
		{"hello world test", "SeedHelloWorldTest"},
		{"foo_bar_baz", "SeedFooBarBaz"},
		{"x", "SeedX"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := toConstName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateFile_AllDirectoryTypes(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.go")

	seeds := []SeedInfo{
		{Name: "Base1", ConstName: "SeedBase1", SourceFile: "base1.go", Directory: "base"},
		{Name: "Dev1", ConstName: "SeedDev1", SourceFile: "dev1.go", Directory: "development"},
		{Name: "Test1", ConstName: "SeedTest1", SourceFile: "test1.go", Directory: "test"},
		{
			Name:       "Testing1",
			ConstName:  "SeedTesting1",
			SourceFile: "testing1.go",
			Directory:  "testing",
		},
		{Name: "Other1", ConstName: "SeedOther1", SourceFile: "other1.go", Directory: "other"},
	}

	err := generateFile(seeds, outputFile, "pkg")
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)

	allSection := extractSection(contentStr, "var AllSeedIDs", "}")
	assert.Contains(t, allSection, "SeedBase1")
	assert.Contains(t, allSection, "SeedDev1")
	assert.Contains(t, allSection, "SeedTest1")
	assert.Contains(t, allSection, "SeedTesting1")
	assert.Contains(t, allSection, "SeedOther1")

	baseSec := extractSection(contentStr, "var BaseSeedIDs", "}")
	assert.Contains(t, baseSec, "SeedBase1")
	assert.NotContains(t, baseSec, "SeedOther1")

	devSec := extractSection(contentStr, "var DevelopmentSeedIDs", "}")
	assert.Contains(t, devSec, "SeedDev1")

	testSec := extractSection(contentStr, "var TestSeedIDs", "}")
	assert.Contains(t, testSec, "SeedTest1")
	assert.Contains(t, testSec, "SeedTesting1")
	assert.NotContains(t, testSec, "SeedOther1")
}

func TestGenerateFile_ValidGoSyntax(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.go")

	seeds := []SeedInfo{
		{Name: "Alpha", ConstName: "SeedAlpha", SourceFile: "alpha.go", Directory: "base"},
	}

	err := generateFile(seeds, outputFile, "testpkg")
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "package testpkg")
	assert.Contains(t, contentStr, "type SeedID string")
	assert.Contains(t, contentStr, "func (s SeedID) String() string")
	assert.Contains(t, contentStr, "func ValidateSeedID(id SeedID) bool")
}

func TestFindSeeds_FileWalkError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	badFile := filepath.Join(baseDir, "bad.go")
	content := `package base

func broken( {
`
	require.NoError(t, os.WriteFile(badFile, []byte(content), 0o644))

	_, err := findSeeds(tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing")
}

func TestExtractSection_NotFound(t *testing.T) {
	t.Parallel()

	result := extractSection("some content", "missing start", "}")
	assert.Empty(t, result)
}

func TestExtractSection_NoEnd(t *testing.T) {
	t.Parallel()

	content := "var AllSeedIDs = []SeedID{ no closing brace"
	result := extractSection(content, "var AllSeedIDs", "NOTFOUND")
	assert.Equal(t, content, result)
}

func TestParseFile_IntegerFirstArg(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	seedFile := filepath.Join(tmpDir, "int_arg.go")

	content := `package seeds

import "github.com/emoss08/trenova/pkg/seedhelpers"

func NewSeed() *Seed {
    seed := &Seed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(123, "1.0.0", "desc", nil)
    return seed
}
`
	err := os.WriteFile(seedFile, []byte(content), 0o644)
	require.NoError(t, err)

	seeds, err := parseFile(seedFile, "base")
	require.NoError(t, err)
	assert.Empty(t, seeds)
}
