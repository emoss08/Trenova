package imagingbench

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

const (
	CorpusEnv  = "TRENOVA_IMAGING_CORPUS"
	ResultsEnv = "TRENOVA_IMAGING_RESULTS"
	LabelEnv   = "TRENOVA_IMAGING_LABEL"
)

type Entry struct {
	File        string `json:"file"`
	Kind        string `json:"kind"`
	Pages       int    `json:"pages,omitempty"`
	Format      string `json:"format,omitempty"`
	Expect      string `json:"expect,omitempty"`
	Codes       int    `json:"codes,omitempty"`
	Blank       bool   `json:"blank,omitempty"`
	Orientation int    `json:"orientation,omitempty"`
}

type Corpus struct {
	Dir     string
	Entries []Entry
}

func Load(t testing.TB) *Corpus {
	t.Helper()

	dir := os.Getenv(CorpusEnv)
	if dir == "" {
		t.Skipf(
			"%s is not set; generate the corpus with tools/docling-fixtures/corpus.py",
			CorpusEnv,
		)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("read corpus manifest: %v", err)
	}

	var entries []Entry
	if err = sonic.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("parse corpus manifest: %v", err)
	}

	return &Corpus{Dir: dir, Entries: entries}
}

func (c *Corpus) Kind(kinds ...string) []Entry {
	out := make([]Entry, 0, len(c.Entries))
	for _, entry := range c.Entries {
		if slices.Contains(kinds, entry.Kind) {
			out = append(out, entry)
		}
	}

	return out
}

func (c *Corpus) Find(t testing.TB, name string) Entry {
	t.Helper()

	for _, entry := range c.Entries {
		if entry.File == name {
			return entry
		}
	}
	t.Fatalf("corpus has no %s", name)

	return Entry{}
}

func (c *Corpus) Path(name string) string {
	return filepath.Join(c.Dir, name)
}

func (c *Corpus) Read(t testing.TB, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(c.Path(name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}

	return data
}

type Latency struct {
	Op         string  `json:"op"`
	Input      string  `json:"input"`
	Iterations int     `json:"iterations"`
	Units      string  `json:"units,omitempty"`
	UnitsPerOp float64 `json:"unitsPerOp,omitempty"`
	P50Ms      float64 `json:"p50Ms"`
	P95Ms      float64 `json:"p95Ms"`
	MeanMs     float64 `json:"meanMs"`
	Errors     int     `json:"errors,omitempty"`
	LastError  string  `json:"lastError,omitempty"`
}

type Throughput struct {
	Op           string  `json:"op"`
	Input        string  `json:"input"`
	Concurrency  int     `json:"concurrency"`
	Completed    int     `json:"completed"`
	Seconds      float64 `json:"seconds"`
	PerSecond    float64 `json:"perSecond"`
	PerSecondCPU float64 `json:"perSecondPerCore"`
	Errors       int     `json:"errors,omitempty"`
}

type PeakRSS struct {
	Input      string  `json:"input"`
	InputBytes int64   `json:"inputBytes"`
	Pipeline   string  `json:"pipeline"`
	PeakRSSMB  float64 `json:"peakRssMb"`
	BaselineMB float64 `json:"baselineRssMb"`
	Seconds    float64 `json:"seconds"`
}

type Machine struct {
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPUs      int    `json:"cpus"`
	CPUModel  string `json:"cpuModel,omitempty"`
}

type Report struct {
	Label      string       `json:"label"`
	RecordedAt string       `json:"recordedAt"`
	Machine    Machine      `json:"machine"`
	Latency    []Latency    `json:"latency"`
	Throughput []Throughput `json:"throughput"`
	PeakRSS    []PeakRSS    `json:"peakRss"`
}

type Recorder struct {
	mu     sync.Mutex
	report Report
}

func NewRecorder() *Recorder {
	return &Recorder{report: Report{Label: os.Getenv(LabelEnv), Machine: machine()}}
}

func (r *Recorder) AddLatency(l Latency) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.report.Latency = append(r.report.Latency, l)
}

func (r *Recorder) AddThroughput(tp Throughput) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.report.Throughput = append(r.report.Throughput, tp)
}

func (r *Recorder) AddPeakRSS(p PeakRSS) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.report.PeakRSS = append(r.report.PeakRSS, p)
}

func (r *Recorder) Flush(t testing.TB) {
	t.Helper()

	path := os.Getenv(ResultsEnv)
	if path == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	merged := Report{Label: r.report.Label, Machine: r.report.Machine}
	if raw, err := os.ReadFile(path); err == nil {
		if err = sonic.Unmarshal(raw, &merged); err != nil {
			t.Fatalf("parse existing results %s: %v", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read existing results %s: %v", path, err)
	}

	merged.Label = r.report.Label
	merged.Machine = r.report.Machine
	merged.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	merged.Latency = mergeBy(merged.Latency, r.report.Latency, func(l Latency) string {
		return l.Op + "\x00" + l.Input
	})
	merged.Throughput = mergeBy(merged.Throughput, r.report.Throughput, func(tp Throughput) string {
		return tp.Op + "\x00" + tp.Input + "\x00" + strconv.Itoa(tp.Concurrency)
	})
	merged.PeakRSS = mergeBy(merged.PeakRSS, r.report.PeakRSS, func(p PeakRSS) string {
		return p.Input + "\x00" + p.Pipeline
	})

	out, err := sonic.ConfigStd.MarshalIndent(merged, "", "  ")
	if err != nil {
		t.Fatalf("encode results: %v", err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create results directory: %v", err)
	}
	if err = os.WriteFile(path, append(out, '\n'), 0o600); err != nil {
		t.Fatalf("write results %s: %v", path, err)
	}
}

func mergeBy[T any](existing, fresh []T, key func(T) string) []T {
	index := make(map[string]int, len(existing))
	for i, item := range existing {
		index[key(item)] = i
	}
	for _, item := range fresh {
		if i, ok := index[key(item)]; ok {
			existing[i] = item
			continue
		}
		index[key(item)] = len(existing)
		existing = append(existing, item)
	}

	return existing
}

type Op func(ctx context.Context) (units int, err error)

func MeasureLatency(
	ctx context.Context,
	op, input string,
	iterations int,
	units string,
	fn Op,
) Latency {
	if _, err := fn(ctx); err != nil {
		return Latency{Op: op, Input: input, Errors: 1, LastError: err.Error()}
	}

	durations := make([]time.Duration, 0, iterations)
	result := Latency{Op: op, Input: input, Iterations: iterations, Units: units}
	totalUnits := 0
	for range iterations {
		start := time.Now()
		n, err := fn(ctx)
		elapsed := time.Since(start)
		if err != nil {
			result.Errors++
			result.LastError = err.Error()
			continue
		}
		totalUnits += n
		durations = append(durations, elapsed)
	}
	if len(durations) == 0 {
		return result
	}

	result.P50Ms = millis(percentile(durations, 0.50))
	result.P95Ms = millis(percentile(durations, 0.95))
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	result.MeanMs = millis(sum / time.Duration(len(durations)))
	if units != "" {
		result.UnitsPerOp = float64(totalUnits) / float64(len(durations))
	}

	return result
}

func MeasureThroughput(
	ctx context.Context,
	op, input string,
	concurrency int,
	window time.Duration,
	cores int,
	fn Op,
) Throughput {
	ctx, cancel := context.WithTimeout(ctx, window)
	defer cancel()

	var (
		mu        sync.Mutex
		completed int
		failed    int
		wg        sync.WaitGroup
	)
	start := time.Now()
	for range concurrency {
		wg.Go(func() {
			for ctx.Err() == nil {
				_, err := fn(ctx)
				if ctx.Err() != nil {
					return
				}
				mu.Lock()
				if err != nil {
					failed++
				} else {
					completed++
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	seconds := time.Since(start).Seconds()

	perSecond := float64(completed) / seconds
	return Throughput{
		Op:           op,
		Input:        input,
		Concurrency:  concurrency,
		Completed:    completed,
		Seconds:      seconds,
		PerSecond:    perSecond,
		PerSecondCPU: perSecond / float64(max(cores, 1)),
		Errors:       failed,
	}
}

func percentile(durations []time.Duration, p float64) time.Duration {
	sorted := slices.Clone(durations)
	slices.Sort(sorted)
	rank := int(float64(len(sorted)-1)*p + 0.5)

	return sorted[min(rank, len(sorted)-1)]
}

func millis(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}

func PeakRSSMB() (float64, error) {
	return procStatusMB("VmHWM:")
}

func CurrentRSSMB() (float64, error) {
	return procStatusMB("VmRSS:")
}

func procStatusMB(field string) (float64, error) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, field) {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, field))
		if len(fields) == 0 {
			break
		}
		kb, parseErr := strconv.ParseFloat(fields[0], 64)
		if parseErr != nil {
			return 0, parseErr
		}
		return kb / 1024, nil
	}

	return 0, fmt.Errorf("%s not in /proc/self/status", field)
}

func machine() Machine {
	m := Machine{
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.GOMAXPROCS(0),
	}
	if raw, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for line := range strings.SplitSeq(string(raw), "\n") {
			if name, ok := strings.CutPrefix(line, "model name"); ok {
				m.CPUModel = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), ":"))
				break
			}
		}
	}

	return m
}

func SortedInputs(entries []Entry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.File)
	}
	sort.Strings(names)

	return names
}
