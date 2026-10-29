package benchbaseline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchbaseline"
	"github.com/emoss08/trenova/internal/deskbench/benchreport"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShippedBaselineSpec_NamesTheReviewsCells(t *testing.T) {
	t.Parallel()

	spec, err := benchbaseline.LoadBaselineSpec("../../../deskbench/baseline.yaml")
	require.NoError(t, err)

	assert.Equal(t, 4, spec.Repeat)
	assert.Equal(t, 4, spec.PassK)
	assert.Equal(t, []string{"claude-haiku-5-5@low", "claude-haiku-5-5@medium"}, spec.Cells)
	assert.Equal(t, []int{8, 12, 20, 30}, spec.Sweep.Held)
	assert.Equal(t, 2, spec.Sweep.Repeat)
	assert.Equal(t, spec.Cells, spec.Sweep.Cells, "the sweep crosses every model by default")
}

func TestLoadBaselineSpec_RefusesAnUnknownEffort(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "baseline.yaml")
	require.NoError(t, os.WriteFile(path, []byte("cells: [claude-haiku-5-5@turbo]\n"), 0o600))

	_, err := benchbaseline.LoadBaselineSpec(path)
	require.Error(t, err)
}

func TestBaseline_WritesMarkdownAndJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	report := benchreport.BuildReport(benchreport.ReportMeta{Commit: "abc123"}, nil)
	require.NoError(t, benchbaseline.BaselineOf(report).Write(dir))

	assert.FileExists(t, filepath.Join(dir, "baseline.md"))
	assert.FileExists(t, filepath.Join(dir, "baseline.json"))

	sweep := &benchbaseline.SweepReport{PassK: 4}
	sweep.Add(&benchbaseline.SweepPoint{Held: 12, Mode: benchscenario.SweepAll})
	require.NoError(t, sweep.Write(dir))
	rendered, err := os.ReadFile(filepath.Join(dir, "sweep.md"))
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "HELD")
}
