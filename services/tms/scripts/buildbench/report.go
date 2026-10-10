package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bytedance/sonic"
)

type benchMeta struct {
	GitSHA    string   `json:"gitSha"`
	Dirty     bool     `json:"dirty"`
	Target    string   `json:"target"`
	Package   string   `json:"package"`
	Mode      string   `json:"mode"`
	Strip     bool     `json:"strip"`
	BuildArgs []string `json:"buildArgs"`
	Jobs      int      `json:"jobs"`
	CPUs      int      `json:"cpus"`
	GoVersion string   `json:"goVersion"`
	GoCache   string   `json:"goCache"`
	Started   string   `json:"started"`
}

type benchReport struct {
	Meta      benchMeta        `json:"meta"`
	Scenarios []scenarioResult `json:"scenarios"`
}

func runReport(args []string) error {
	var (
		rep       benchReport
		out       string
		buildArgs string
	)
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.StringVar(&out, "out", "", "JSON report path")
	fs.StringVar(&rep.Meta.GitSHA, "sha", "", "git commit")
	fs.BoolVar(&rep.Meta.Dirty, "dirty", false, "working tree had uncommitted changes")
	fs.StringVar(&rep.Meta.Target, "target", "", "target name")
	fs.StringVar(&rep.Meta.Package, "package", "", "target package")
	fs.StringVar(&rep.Meta.Mode, "mode", "", "flag set")
	fs.BoolVar(&rep.Meta.Strip, "strip", false, "linker strips symbols")
	fs.StringVar(&buildArgs, "build-args", "", "go build arguments, tab separated")
	fs.IntVar(&rep.Meta.Jobs, "jobs", 0, "go build -p")
	fs.IntVar(&rep.Meta.CPUs, "cpus", 0, "online CPUs")
	fs.StringVar(&rep.Meta.GoVersion, "go-version", "", "go version")
	fs.StringVar(&rep.Meta.GoCache, "gocache", "", "benchmark build cache")
	fs.StringVar(&rep.Meta.Started, "started", "", "start time")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rep.Meta.BuildArgs = strings.Split(buildArgs, "\t")

	rep.Scenarios = make([]scenarioResult, 0, fs.NArg())
	for _, path := range fs.Args() {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read scenario result: %w", err)
		}
		var res scenarioResult
		if err = sonic.Unmarshal(data, &res); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
		rep.Scenarios = append(rep.Scenarios, res)
	}

	if err := writeReport(out, &rep); err != nil {
		return err
	}

	return printReport(os.Stdout, &rep)
}

func writeReport(path string, rep *benchReport) error {
	data, err := sonic.ConfigStd.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create results directory: %w", err)
	}
	if err = os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

func printReport(w io.Writer, rep *benchReport) error {
	m := &rep.Meta
	dirty := ""
	if m.Dirty {
		dirty = " (dirty)"
	}
	fmt.Fprintf(w, "\n%s %s, mode %s, strip %t, -p %d of %d CPUs, %s, commit %s%s\n\n",
		m.Target, m.Package, m.Mode, m.Strip, m.Jobs, m.CPUs, m.GoVersion, m.GitSHA, dirty)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Scenario\tWall\tLink\t1st-party recompiled\tAll recompiled\tBinary\tProbe")
	for i := range rep.Scenarios {
		s := &rep.Scenarios[i]
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%s\t%s\n",
			s.ID,
			seconds(s.WallSeconds),
			seconds(s.LinkSeconds),
			s.FirstPartyRecompile,
			s.TotalRecompiled,
			megabytes(s.BinaryBytes),
			probeLabel(s),
		)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("write table: %w", err)
	}

	for i := range rep.Scenarios {
		s := &rep.Scenarios[i]
		fmt.Fprintf(w, "\n%s: %s, slowest actions\n", s.ID, s.Description)
		for _, a := range s.Slowest {
			fmt.Fprintf(w, "  %8s  %-6s %s\n", seconds(a.Seconds), a.Mode, a.Package)
		}
	}

	return nil
}

func probeLabel(s *scenarioResult) string {
	if s.Package == "" {
		return s.Description
	}

	return fmt.Sprintf("%s in %s (%d dependents)", s.Kind, s.Package, s.PackageDependents)
}

func seconds(v float64) string {
	return (time.Duration(v * float64(time.Second))).Round(100 * time.Millisecond).String()
}

func megabytes(v int64) string {
	return fmt.Sprintf("%.0f MB", float64(v)/1e6)
}
