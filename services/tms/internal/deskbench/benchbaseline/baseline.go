package benchbaseline

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/emoss08/trenova/internal/deskbench/benchcell"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchreport"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchstats"
	"gopkg.in/yaml.v3"
)

const (
	baselineMarkdown = "baseline.md"
	baselineJSON     = "baseline.json"
	sweepMarkdown    = "sweep.md"
	sweepJSON        = "sweep.json"
)

type BaselineSpec struct {
	Repeat int       `yaml:"repeat"`
	PassK  int       `yaml:"passK"`
	Cells  []string  `yaml:"cells"`
	Sweep  SweepSpec `yaml:"sweep"`
}

type SweepSpec struct {
	Held   []int                     `yaml:"held"`
	Modes  []benchscenario.SweepMode `yaml:"modes"`
	Cells  []string                  `yaml:"cells"`
	Repeat int                       `yaml:"repeat"`
	Tags   []string                  `yaml:"tags"`
}

var ErrBaselineSpec = errors.New("the baseline spec is incomplete")

func LoadBaselineSpec(path string) (*BaselineSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	spec := new(BaselineSpec)
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err = decoder.Decode(spec); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err = spec.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return spec, nil
}

func (s *BaselineSpec) validate() error {
	if len(s.Cells) == 0 {
		return fmt.Errorf("%w: name at least one cell", ErrBaselineSpec)
	}
	for _, cell := range s.Cells {
		if _, _, err := benchcell.SplitEffort(cell); err != nil {
			return err
		}
	}
	if s.Repeat <= 0 {
		s.Repeat = benchstats.DefaultPassK
	}
	if s.PassK <= 0 {
		s.PassK = benchstats.DefaultPassK
	}
	if len(s.Sweep.Cells) == 0 {
		s.Sweep.Cells = slices.Clone(s.Cells)
	}
	if s.Sweep.Repeat <= 0 {
		s.Sweep.Repeat = s.Repeat
	}
	if len(s.Sweep.Modes) == 0 {
		s.Sweep.Modes = []benchscenario.SweepMode{
			benchscenario.SweepAll,
			benchscenario.SweepDisclosed,
		}
	}
	for _, mode := range s.Sweep.Modes {
		if mode != benchscenario.SweepAll && mode != benchscenario.SweepDisclosed {
			return fmt.Errorf("%w: a sweep mode is all or disclosed, not %q", ErrBaselineSpec, mode)
		}
	}
	for _, held := range s.Sweep.Held {
		if held <= 0 {
			return fmt.Errorf("%w: a held tool count is positive, not %d", ErrBaselineSpec, held)
		}
	}

	return nil
}

type Baseline struct {
	Meta    benchreport.ReportMeta  `json:"meta"`
	Cases   int                     `json:"cases"`
	Passed  int                     `json:"passed"`
	Skipped int                     `json:"skipped,omitempty"`
	PassK   int                     `json:"passK"`
	Usage   benchrecord.Usage       `json:"usage"`
	Cells   []benchreport.CellRow   `json:"cells"`
	Matrix  []benchreport.MatrixRow `json:"matrix"`
}

func BaselineOf(report *benchreport.Report) *Baseline {
	return &Baseline{
		Meta:    report.Meta,
		Cases:   report.Summary.Cases,
		Passed:  report.Summary.Passed,
		Skipped: report.Summary.Skipped,
		PassK:   report.Summary.PassK,
		Usage:   report.Summary.Usage,
		Cells:   report.Summary.Cells,
		Matrix:  report.Summary.Matrix,
	}
}

func (b *Baseline) Write(dir string) error {
	encoded, err := benchrecord.JSON.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("encode the baseline: %w", err)
	}
	if err = os.WriteFile(
		filepath.Join(dir, baselineJSON),
		encoded,
		benchrecord.FileMode,
	); err != nil {
		return fmt.Errorf("write the baseline: %w", err)
	}

	return os.WriteFile(
		filepath.Join(dir, baselineMarkdown),
		[]byte(b.Render()),
		benchrecord.FileMode,
	)
}

func (b *Baseline) Render() string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Desk baseline\n\n%d of %d cases passed", b.Passed, b.Cases)
	if b.Skipped > 0 {
		fmt.Fprintf(&out, ", %d skipped", b.Skipped)
	}
	fmt.Fprintf(&out, " · %s · commit %s · %s\n\n",
		b.Meta.FinishedAt.Sub(b.Meta.StartedAt).Round(time.Second),
		stringOr(b.Meta.Commit, "unknown"), benchrecord.UsageLine(&b.Usage))
	out.WriteString("Every figure is a mean per trial with a 95% interval in brackets. A model's " +
		"pass is the mean of its scenarios' pass rates with a t-interval over scenarios (a " +
		"scenario's trials are not independent); a scenario's is a Wilson interval over its " +
		"trials. pass^" + strconv.Itoa(b.PassK) + " is the chance all " +
		strconv.Itoa(b.PassK) + " trials of a scenario pass, averaged over scenarios. A cost " +
		"marked ~ was priced from the model's list price or the price reference because the " +
		"provider carries none. In is input the cache did not serve.\n\n")

	out.WriteString("```\n")
	benchreport.RenderCells(&out, b.Cells, b.PassK)
	benchreport.RenderMatrix(&out, b.Matrix, b.PassK)
	out.WriteString("```\n")

	return out.String()
}

type SweepPoint struct {
	Held      int                     `json:"held"`
	Mode      benchscenario.SweepMode `json:"mode"`
	Threshold int                     `json:"toolDisclosureThreshold"`
	Run       string                  `json:"run"`
	Cases     int                     `json:"cases"`
	Skipped   int                     `json:"skipped"`
	Cells     []benchreport.CellRow   `json:"cells"`
}

type SweepReport struct {
	StartedAt  time.Time    `json:"startedAt"`
	FinishedAt time.Time    `json:"finishedAt"`
	Commit     string       `json:"commit,omitempty"`
	PassK      int          `json:"passK"`
	Points     []SweepPoint `json:"points"`
}

func (r *SweepReport) Add(point *SweepPoint) {
	r.Points = append(r.Points, *point)
}

func (r *SweepReport) Write(dir string) error {
	encoded, err := benchrecord.JSON.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode the sweep: %w", err)
	}
	if err = os.WriteFile(
		filepath.Join(dir, sweepJSON),
		encoded,
		benchrecord.FileMode,
	); err != nil {
		return fmt.Errorf("write the sweep: %w", err)
	}

	return os.WriteFile(filepath.Join(dir, sweepMarkdown), []byte(r.Render()), benchrecord.FileMode)
}

func (r *SweepReport) Render() string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Tool-count sweep\n\nCommit %s · %s\n\n", stringOr(r.Commit, "unknown"),
		r.FinishedAt.Sub(r.StartedAt).Round(time.Second))
	out.WriteString("Each point holds the agent to N tools: the ones the scenario needs and the " +
		"agent's others up to N. \"all\" sends every held tool on every call; \"disclosed\" sends " +
		"the 8 ranked for the request and find_tools. Set disclosureThreshold " +
		"(ai.toolDisclosureThreshold) to the largest N at which \"all\" is at least as good.\n\n")

	rows := make([]sweepRow, 0, len(r.Points)*2)
	for _, point := range r.Points {
		for idx := range point.Cells {
			if point.Cells[idx].Split != "" {
				continue
			}
			rows = append(rows, sweepRow{point: point, cell: &point.Cells[idx]})
		}
	}
	slices.SortStableFunc(rows, func(a, b sweepRow) int {
		if a.cell.Provider != b.cell.Provider {
			return strings.Compare(a.cell.Provider, b.cell.Provider)
		}
		if a.point.Held != b.point.Held {
			return a.point.Held - b.point.Held
		}

		return strings.Compare(string(a.point.Mode), string(b.point.Mode))
	})

	out.WriteString("```\n")
	table := tabwriter.NewWriter(&out, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "MODEL\tHELD\tMODE\tTRIALS\tPASS\tPASS^%d\tMODEL CALLS\tTOOL CALLS\t"+
		"REFUSED\tRECALL\tPRESELECTION R@8\tIN+CACHE\t$/TASK\n", r.PassK)
	for _, row := range rows {
		m := &row.cell.Measures
		pass := benchreport.PercentStatCell(row.cell.ScenarioPass)
		fmt.Fprintf(
			table,
			"%s\t%d\t%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%.0f\t%s\n",
			row.cell.Provider,
			row.point.Held,
			row.point.Mode,
			row.cell.Trials,
			pass,
			benchreport.StatCell(row.cell.PassK, 2),
			benchreport.StatCell(m.ModelCalls, 1),
			benchreport.StatCell(
				m.ToolCalls,
				1,
			),
			benchreport.StatCell(m.Refused, 2),
			benchreport.StatCell(m.Recall, 2),
			benchreport.StatCell(m.PreselectionRecall, 2),
			m.UncachedInput.Mean+m.CacheRead.Mean+m.CacheWrite.Mean,
			benchreport.CostCell(m),
		)
	}
	_ = table.Flush()
	out.WriteString("```\n")

	return out.String()
}

type sweepRow struct {
	point SweepPoint
	cell  *benchreport.CellRow
}

func stringOr(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
