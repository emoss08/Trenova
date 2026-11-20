package soak

import (
	"cmp"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
)

type Report struct {
	Run           uint64           `json:"run"`
	Conversations int              `json:"conversations"`
	Rate          float64          `json:"rate"`
	Concurrency   int              `json:"concurrency"`
	StartedAt     time.Time        `json:"startedAt"`
	Took          play.Duration    `json:"took"`
	Throughput    float64          `json:"conversationsPerMinute"`
	Partners      []*PartnerReport `json:"partners"`
	Total         PartnerReport    `json:"total"`
}

type PartnerReport struct {
	Name          string           `json:"name"`
	Role          string           `json:"role"`
	Failure       string           `json:"failure,omitempty"`
	Conversations int              `json:"conversations"`
	Completed     int              `json:"completed"`
	Failed        int              `json:"failed"`
	Milestones    map[string]Stats `json:"milestones"`
	Lost          map[string]int   `json:"lost"`
	Duplicates    map[string]int   `json:"duplicates"`
	OpenFlows     int              `json:"openFlows"`
	Errors        []string         `json:"errors,omitempty"`
	samples       map[string][]time.Duration
}

type Stats struct {
	Count int           `json:"count"`
	P50   play.Duration `json:"p50"`
	P90   play.Duration `json:"p90"`
	P99   play.Duration `json:"p99"`
	Max   play.Duration `json:"max"`
}

func newPartner(name, role string) *PartnerReport {
	return &PartnerReport{
		Name:       name,
		Role:       role,
		Milestones: map[string]Stats{},
		Lost:       map[string]int{},
		Duplicates: map[string]int{},
		samples:    map[string][]time.Duration{},
	}
}

func (p *PartnerReport) add(o *outcome) {
	p.Conversations++
	for name, took := range o.took {
		p.samples[name] = append(p.samples[name], took)
	}
	for _, name := range o.lost {
		p.Lost[name]++
	}
	for set, count := range o.duplicates {
		p.Duplicates[set] += count
	}
	switch {
	case o.err != nil:
		p.Failed++
		if len(p.Errors) < maxErrorsKept {
			p.Errors = append(p.Errors, fmt.Sprintf("%s (%s): %v",
				o.conversation.Name, firstNonEmpty(o.conversation.Key, "no key"), o.err))
		}
	case len(o.lost) == 0:
		p.Completed++
	default:
		p.Failed++
	}
}

func (p *PartnerReport) summarize() {
	for name, samples := range p.samples {
		p.Milestones[name] = statsOf(samples)
	}
}

func statsOf(samples []time.Duration) Stats {
	sorted := slices.Clone(samples)
	slices.Sort(sorted)
	if len(sorted) == 0 {
		return Stats{}
	}
	return Stats{
		Count: len(sorted),
		P50:   play.Duration(percentile(sorted, 0.50)),
		P90:   play.Duration(percentile(sorted, 0.90)),
		P99:   play.Duration(percentile(sorted, 0.99)),
		Max:   play.Duration(sorted[len(sorted)-1]),
	}
}

func percentile(sorted []time.Duration, q float64) time.Duration {
	rank := int(math.Ceil(q*float64(len(sorted)))) - 1
	return sorted[min(max(rank, 0), len(sorted)-1)]
}

func (r *Report) finish(outcomes []*outcome, started time.Time) {
	r.Took = play.Duration(time.Since(started))
	byName := map[string]*PartnerReport{}
	for _, p := range r.Partners {
		byName[p.Name] = p
	}
	total := newPartner("all", "")
	for _, o := range outcomes {
		name := o.party.Name()
		p := byName[name]
		if p == nil {
			p = newPartner(name, string(o.party.Def.Partner.Role))
			byName[name] = p
			r.Partners = append(r.Partners, p)
		}
		p.add(o)
		total.add(o)
	}
	for _, p := range r.Partners {
		p.summarize()
	}
	total.summarize()
	r.Total = *total
	slices.SortFunc(
		r.Partners,
		func(a, b *PartnerReport) int { return cmp.Compare(a.Name, b.Name) },
	)
	if minutes := time.Duration(r.Took).Minutes(); minutes > 0 {
		r.Throughput = float64(total.Completed) / minutes
	}
}

func (r *Report) Passed() bool {
	if r.Total.Failed > 0 || r.Total.OpenFlows > 0 {
		return false
	}
	for _, p := range r.Partners {
		if p.Failure != "" || p.OpenFlows > 0 || len(p.Lost) > 0 || sum(p.Duplicates) > 0 {
			return false
		}
	}
	return true
}

func sum(values map[string]int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func (r *Report) WriteJSON(path string) error {
	data, err := sonic.ConfigStd.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode the report: %w", err)
	}
	if err = os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func (r *Report) WriteSummary(w io.Writer) {
	fmt.Fprintf(w, "soak run %d: %d conversations at %.2f/s, %d at once, %s (%.1f completed "+
		"per minute)\n", r.Run, r.Conversations, r.Rate, r.Concurrency, r.Took, r.Throughput)
	for _, p := range r.Partners {
		writePartner(w, p)
	}
	writePartner(w, &r.Total)
	if r.Passed() {
		fmt.Fprintln(w, "\nPASS")
		return
	}
	fmt.Fprintln(w, "\nFAIL")
}

func writePartner(w io.Writer, p *PartnerReport) {
	fmt.Fprintf(w, "\n%s", p.Name)
	if p.Role != "" {
		fmt.Fprintf(w, " (%s)", p.Role)
	}
	if p.Failure != "" {
		fmt.Fprintf(w, ": not provisioned: %s\n", p.Failure)
		return
	}
	fmt.Fprintf(w, ": %d conversations, %d completed, %d failed, %d flows left open\n",
		p.Conversations, p.Completed, p.Failed, p.OpenFlows)
	fmt.Fprintf(w, "  %-5s %6s %10s %10s %10s %10s %6s %6s\n", "time", "count", "p50", "p90",
		"p99", "max", "lost", "dups")
	for _, name := range milestoneOrder {
		stats := p.Milestones[name]
		fmt.Fprintf(w, "  %-5s %6d %10s %10s %10s %10s %6d %6d\n", "to "+name, stats.Count,
			stats.P50, stats.P90, stats.P99, stats.Max, p.Lost[name], p.Duplicates[name])
	}
	for _, e := range p.Errors {
		fmt.Fprintf(w, "  error: %s\n", e)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
