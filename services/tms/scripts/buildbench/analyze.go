package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const slowestActionCount = 10

type graphAction struct {
	Mode      string    `json:"Mode"`
	Package   string    `json:"Package"`
	Cmd       []string  `json:"Cmd"`
	TimeStart time.Time `json:"TimeStart"`
	TimeDone  time.Time `json:"TimeDone"`
}

type timedAction struct {
	Mode    string  `json:"mode"`
	Package string  `json:"package"`
	Seconds float64 `json:"seconds"`
}

type scenarioResult struct {
	ID                  string        `json:"id"`
	Description         string        `json:"description"`
	Kind                string        `json:"kind,omitempty"`
	Package             string        `json:"package,omitempty"`
	File                string        `json:"file,omitempty"`
	PackageDependents   int           `json:"packageDependents,omitempty"`
	WallSeconds         float64       `json:"wallSeconds"`
	LinkSeconds         float64       `json:"linkSeconds"`
	FirstPartyRecompile int           `json:"firstPartyRecompiled"`
	TotalRecompiled     int           `json:"totalRecompiled"`
	BinaryBytes         int64         `json:"binaryBytes"`
	Recompiled          []string      `json:"firstPartyRecompiledPackages"`
	Slowest             []timedAction `json:"slowestActions"`
}

type analyzeFlags struct {
	graph, module, binary string
	result                scenarioResult
	wallNanos             int64
}

func runAnalyze(args []string) error {
	var f analyzeFlags
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.StringVar(&f.graph, "graph", "", "path written by -debug-actiongraph")
	fs.StringVar(&f.module, "module", "", "first-party module path")
	fs.StringVar(&f.binary, "binary", "", "built binary")
	fs.Int64Var(&f.wallNanos, "wall-ns", 0, "measured wall time in nanoseconds")
	fs.StringVar(&f.result.ID, "id", "", "scenario id")
	fs.StringVar(&f.result.Kind, "kind", "", "probe kind")
	fs.StringVar(&f.result.Package, "package", "", "probed package")
	fs.StringVar(&f.result.File, "file", "", "probed file")
	fs.IntVar(&f.result.PackageDependents, "dependents", 0, "probed package's dependents")
	if err := fs.Parse(args); err != nil {
		return err
	}

	data, err := os.ReadFile(f.graph)
	if err != nil {
		return fmt.Errorf("read action graph: %w", err)
	}
	var actions []graphAction
	if err = sonic.Unmarshal(data, &actions); err != nil {
		return fmt.Errorf("decode action graph: %w", err)
	}
	stat, err := os.Stat(f.binary)
	if err != nil {
		return fmt.Errorf("stat binary: %w", err)
	}

	res := f.result
	res.Description = scenarioDescription(res.ID)
	res.WallSeconds = time.Duration(f.wallNanos).Seconds()
	res.BinaryBytes = stat.Size()
	summarize(&res, actions, f.module)

	out, err := sonic.ConfigStd.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("encode result: %w", err)
	}
	_, err = os.Stdout.Write(append(out, '\n'))

	return err
}

func summarize(res *scenarioResult, actions []graphAction, module string) {
	ran := make([]timedAction, 0, len(actions))
	recompiled := make(map[string]struct{})
	for i := range actions {
		a := &actions[i]
		if len(a.Cmd) == 0 {
			continue
		}
		seconds := a.TimeDone.Sub(a.TimeStart).Seconds()
		ran = append(ran, timedAction{Mode: a.Mode, Package: a.Package, Seconds: seconds})
		switch a.Mode {
		case "link":
			res.LinkSeconds += seconds
		case "build":
			res.TotalRecompiled++
			if a.Package == module || strings.HasPrefix(a.Package, module+"/") {
				recompiled[a.Package] = struct{}{}
			}
		}
	}

	res.Recompiled = make([]string, 0, len(recompiled))
	for p := range recompiled {
		res.Recompiled = append(res.Recompiled, p)
	}
	slices.Sort(res.Recompiled)
	res.FirstPartyRecompile = len(res.Recompiled)

	slices.SortFunc(ran, func(a, b timedAction) int {
		return cmp.Compare(b.Seconds, a.Seconds)
	})
	res.Slowest = ran[:min(len(ran), slowestActionCount)]
}

func scenarioDescription(id string) string {
	switch id {
	case "cold":
		return "cold build (empty buildbench cache)"
	case "warm":
		return "warm-up build (existing buildbench cache)"
	case "S1":
		return "no-op rebuild"
	case "S2":
		return "unexported func added to shipmentservice"
	case "S3":
		return "unexported func added to shipmentresolver"
	case "S4":
		return "exported func added to domain/shipment"
	case "S5":
		return "exported type added to the ShipmentRepository package"
	case "S6":
		return "exported func added to the least-depended-on planned domain package"
	default:
		return id
	}
}
