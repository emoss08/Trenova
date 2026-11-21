package scenario

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/goccy/go-yaml"
)

const (
	Dir             = "scenarios"
	DefaultWithin   = 3 * time.Minute
	MaxSeedsPerCase = 40
	MaxBatch        = 30
)

const (
	KindSend    = "send"
	KindExpect  = "expect"
	KindTrenova = "trenova"
	KindAdvance = "advance"
	KindScript  = "script"
	KindAcks    = "acks"
)

const (
	TotalTendered = "tendered"
	TotalOriginal = "original"
)

type Scenario struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Within      time.Duration `yaml:"within"`
	Steps       []Step        `yaml:"steps"`
	File        string        `yaml:"-"`
	Slug        string        `yaml:"-"`
}

type Step struct {
	Name    string   `yaml:"name"`
	Send    *Send    `yaml:"send"`
	Expect  *Expect  `yaml:"expect"`
	Trenova *Action  `yaml:"trenova"`
	Advance *Advance `yaml:"advance"`
	Script  *Script  `yaml:"script"`
	Acks    *Acks    `yaml:"acks"`
}

type Delta struct {
	RatePercent float64 `yaml:"ratePercent"`
	AddStop     bool    `yaml:"addStop"`
	Move        *Move   `yaml:"move"`
}

type Move struct {
	Stop int           `yaml:"stop"`
	By   time.Duration `yaml:"by"`
}

type Send struct {
	Set       string            `yaml:"set"`
	As        string            `yaml:"as"`
	Of        string            `yaml:"of"`
	Variation string            `yaml:"variation"`
	Seed      uint64            `yaml:"seed"`
	Count     int               `yaml:"count"`
	Sparse    bool              `yaml:"sparse"`
	Delta     Delta             `yaml:"delta"`
	Load      *partner.LoadNeed `yaml:"load"`
}

type Expect struct {
	Wire       string            `yaml:"wire"`
	About      string            `yaml:"about"`
	Ack        string            `yaml:"ack"`
	Flow       string            `yaml:"flow"`
	Trenova    *State            `yaml:"trenova"`
	Reconciled string            `yaml:"reconciled"`
	Within     time.Duration     `yaml:"within"`
	Accepted   *bool             `yaml:"accepted"`
	Elements   map[string]string `yaml:"elements"`
	Statuses   []string          `yaml:"statuses"`
	Total      string            `yaml:"total"`
	Count      int               `yaml:"count"`
	None       bool              `yaml:"none"`
	Status     string            `yaml:"status"`
	Why        string            `yaml:"why"`
	Resent     bool              `yaml:"resent"`
	Set        string            `yaml:"set"`
}

type State struct {
	Shipment string `yaml:"shipment"`
	Exists   *bool  `yaml:"exists"`
	Stops    int    `yaml:"stops"`
	Status   string `yaml:"status"`
	Total    string `yaml:"total"`
}

type Action struct {
	Decide   string `yaml:"decide"`
	Decision string `yaml:"decision"`
	Reason   string `yaml:"reason"`
	Dispatch string `yaml:"dispatch"`
	Arrive   string `yaml:"arrive"`
	Depart   string `yaml:"depart"`
	Stop     int    `yaml:"stop"`
	Deliver  string `yaml:"deliver"`
	Invoice  string `yaml:"invoice"`
	Release  string `yaml:"release"`
	Tender   string `yaml:"tender"`
	Rate     string `yaml:"rate"`
}

type Advance struct {
	About string        `yaml:"about"`
	By    time.Duration `yaml:"by"`
}

type Script struct {
	Answers []Answer `yaml:"answers"`
}

type Answer struct {
	About      string  `yaml:"about"`
	Decline    string  `yaml:"decline"`
	DeltaCents int64   `yaml:"deltaCents"`
	Misfooted  bool    `yaml:"misfooted"`
	Stray      bool    `yaml:"stray"`
	Delay      float64 `yaml:"delay"`
	Silent     bool    `yaml:"silent"`
}

type Acks struct {
	About  string `yaml:"about"`
	Set    string `yaml:"set"`
	Action string `yaml:"action"`
}

func (s *Step) Kind() string {
	switch {
	case s.Send != nil:
		return KindSend
	case s.Expect != nil:
		return KindExpect
	case s.Trenova != nil:
		return KindTrenova
	case s.Advance != nil:
		return KindAdvance
	case s.Script != nil:
		return KindScript
	case s.Acks != nil:
		return KindAcks
	}
	return ""
}

func (s *Step) count() int {
	count := 0
	for _, set := range []bool{
		s.Send != nil, s.Expect != nil, s.Trenova != nil,
		s.Advance != nil, s.Script != nil, s.Acks != nil,
	} {
		if set {
			count++
		}
	}
	return count
}

func (a *Action) Verb() (string, string) {
	for _, pair := range [][2]string{
		{"decide", a.Decide}, {"dispatch", a.Dispatch}, {"arrive", a.Arrive},
		{"depart", a.Depart}, {"deliver", a.Deliver}, {"invoice", a.Invoice},
		{"release", a.Release}, {"tender", a.Tender},
	} {
		if pair[1] != "" {
			return pair[0], pair[1]
		}
	}
	return "", ""
}

func (a *Action) verbs() int {
	count := 0
	for _, value := range []string{
		a.Decide, a.Dispatch, a.Arrive, a.Depart, a.Deliver, a.Invoice, a.Release, a.Tender,
	} {
		if value != "" {
			count++
		}
	}
	return count
}

func (e *Expect) Subject() (string, string) {
	switch {
	case e.Wire != "":
		return "wire", e.About
	case e.Ack != "":
		return "ack", e.Ack
	case e.Flow != "":
		return "flow", e.Flow
	case e.Reconciled != "":
		return "reconciled", e.Reconciled
	case e.Trenova != nil:
		return "trenova", e.Trenova.Shipment
	}
	return "", ""
}

func (e *Expect) subjects() int {
	count := 0
	for _, set := range []bool{
		e.Wire != "", e.Ack != "", e.Flow != "", e.Reconciled != "", e.Trenova != nil,
	} {
		if set {
			count++
		}
	}
	return count
}

func (s *Scenario) WithinOf(step time.Duration) time.Duration {
	if step > 0 {
		return step
	}
	if s.Within > 0 {
		return s.Within
	}
	return DefaultWithin
}

func Load(partnerDir string) ([]*Scenario, error) {
	paths, err := filepath.Glob(filepath.Join(partnerDir, Dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	slices.Sort(paths)
	var out []*Scenario
	var problems []error
	for _, path := range paths {
		s, loadErr := LoadFile(path)
		if loadErr != nil {
			problems = append(problems, loadErr)
			continue
		}
		out = append(out, s)
	}
	return out, errors.Join(problems...)
}

func LoadFile(path string) (*Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	s := &Scenario{File: path, Slug: strings.TrimSuffix(filepath.Base(path), ".yaml")}
	if err = yaml.UnmarshalWithOptions(raw, s, yaml.DisallowUnknownField()); err != nil {
		return nil, fmt.Errorf("%s: %s", path, yaml.FormatError(err, false, true))
	}
	if err = s.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

var (
	elementPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,2}-\d{2}(-\d{1,2})?$`)
	namePattern    = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	sets           = []string{"204", "990", "214", "210", "997", "999", "TA1"}
	variations     = []string{"original", "duplicate", "change", "replace", "cancel"}
	decisions      = []string{"accept", "decline"}
	ackActions     = []string{"withhold", "reject"}
	flowStatuses   = []string{"Open", "Held", "Escalated", "Closed"}
)

type checker struct {
	problems []string
	named    map[string]bool
}

func (c *checker) add(path, format string, args ...any) {
	c.problems = append(c.problems, path+": "+fmt.Sprintf(format, args...))
}

func (c *checker) known(path, name string) {
	if name != "" && !c.named[name] {
		c.add(path, "%q names no conversation an earlier step started", name)
	}
}

func (s *Scenario) Validate() error {
	c := &checker{named: map[string]bool{}}
	if strings.TrimSpace(s.Name) == "" {
		c.add("name", "is required")
	}
	if len(s.Steps) == 0 {
		c.add("steps", "a scenario needs at least one step")
	}
	seeds := map[uint64]string{}
	for index := range s.Steps {
		step := &s.Steps[index]
		path := fmt.Sprintf("steps[%d]", index)
		if step.count() != 1 {
			c.add(path, "has %d of send, expect, trenova, advance, script and acks; "+
				"it takes exactly one", step.count())
			continue
		}
		switch step.Kind() {
		case KindSend:
			c.send(path+".send", step.Send, seeds)
		case KindExpect:
			c.expect(path+".expect", step.Expect)
		case KindTrenova:
			c.action(path+".trenova", step.Trenova)
		case KindAdvance:
			c.known(path+".advance.about", step.Advance.About)
			if step.Advance.About == "" || step.Advance.By <= 0 {
				c.add(path+".advance", "needs about and a positive by, such as 2h")
			}
		case KindScript:
			if len(step.Script.Answers) == 0 {
				c.add(path+".script.answers", "is empty")
			}
			for a, answer := range step.Script.Answers {
				c.known(fmt.Sprintf("%s.script.answers[%d].about", path, a), answer.About)
				if answer.Delay < 0 || answer.Delay > 1 {
					c.add(fmt.Sprintf("%s.script.answers[%d].delay", path, a),
						"is a probability between 0 and 1")
				}
			}
		case KindAcks:
			c.known(path+".acks.about", step.Acks.About)
			if !slices.Contains(ackActions, step.Acks.Action) {
				c.add(path+".acks.action", "%q is not one of %v", step.Acks.Action, ackActions)
			}
		}
	}
	if len(c.problems) > 0 {
		return errors.New(strings.Join(c.problems, "\n"))
	}
	return nil
}

func (c *checker) send(path string, send *Send, seeds map[uint64]string) {
	if send.Set != "204" {
		c.add(path+".set", "%q: a partner sends only generated 204s", send.Set)
	}
	variation := cmpOr(send.Variation, "original")
	if !slices.Contains(variations, variation) {
		c.add(path+".variation", "%q is not one of %v", send.Variation, variations)
	}
	for _, problem := range send.Load.Problems() {
		c.add(path+".load", "%s", problem)
	}
	if variation == "original" {
		if !namePattern.MatchString(send.As) {
			c.add(path+".as", "an original names its conversation (lowercase, such as t1)")
		} else if c.named[send.As] {
			c.add(path+".as", "%q is already a conversation", send.As)
		}
		if send.Of != "" {
			c.add(path+".of", "an original starts a conversation; of is for later variations")
		}
		c.named[send.As] = true
		count := max(send.Count, 1)
		if count > MaxBatch {
			c.add(path+".count", "%d: at most %d tenders in one send", count, MaxBatch)
			return
		}
		if send.Seed == 0 {
			send.Seed = nextSeed(seeds, uint64(count))
		}
		if send.Seed+uint64(count) > MaxSeedsPerCase {
			c.add(path+".seed", "%d: a scenario's seeds run 1..%d", send.Seed, MaxSeedsPerCase-1)
			return
		}
		for offset := range uint64(count) {
			if owner, taken := seeds[send.Seed+offset]; taken {
				c.add(path+".seed", "seed %d is already %s's", send.Seed+offset, owner)
			}
			seeds[send.Seed+offset] = send.As
		}
		return
	}
	if send.As != "" {
		c.add(path+".as", "only an original names a conversation; use of")
	}
	if send.Count > 1 {
		c.add(path+".count", "only an original is batched")
	}
	if send.Load != nil {
		c.add(path+".load", "only an original states its load; a %s keeps its original's",
			variation)
	}
	c.known(path+".of", send.Of)
	if send.Of == "" {
		c.add(path+".of", "a %s names the conversation it varies", variation)
	}
}

func (c *checker) expect(path string, e *Expect) {
	if e.subjects() != 1 {
		c.add(path, "takes exactly one of wire, ack, flow, reconciled and trenova")
		return
	}
	kind, about := e.Subject()
	if kind == "wire" {
		if !slices.Contains(sets, e.Wire) {
			c.add(path+".wire", "%q is not one of %v", e.Wire, sets)
		}
		if e.About == "" {
			c.add(path+".about", "a wire expectation names its conversation")
		}
	}
	c.known(path+"."+kind, about)
	for element := range e.Elements {
		if !elementPattern.MatchString(element) {
			c.add(path+".elements", "%q is not a segment and position such as B1-04", element)
		}
	}
	if len(e.Statuses) > 0 && e.Wire != "214" {
		c.add(path+".statuses", "statuses check 214s")
	}
	if e.Total != "" && e.Wire != "210" {
		c.add(path+".total", "total checks a 210")
	}
	if e.Total != "" {
		c.total(path+".total", e.Total)
	}
	if e.Status != "" && kind == "flow" && !slices.Contains(flowStatuses, e.Status) {
		c.add(path+".status", "%q is not one of %v", e.Status, flowStatuses)
	}
	if (e.Status != "" || e.Why != "") && kind != "flow" {
		c.add(path, "status and why check a flow")
	}
	if e.None && (e.Count > 0 || len(e.Elements) > 0 || len(e.Statuses) > 0) {
		c.add(path+".none", "an expectation of nothing checks nothing about it")
	}
	if e.Set != "" && kind != "ack" {
		c.add(path+".set", "set picks which of the partner's messages an ack expectation checks")
	}
	if e.Count > 0 && kind != "wire" && kind != "flow" {
		c.add(path+".count", "count checks the wire or the flow instances")
	}
	if e.Resent && kind != "wire" {
		c.add(path+".resent", "resent checks the wire")
	}
	if kind == "trenova" {
		if e.Trenova.Total != "" {
			c.total(path+".trenova.total", e.Trenova.Total)
		}
	}
}

func (c *checker) total(path, total string) {
	switch {
	case total == TotalTendered || total == TotalOriginal:
	case strings.HasSuffix(total, "%"):
		if _, err := strconv.ParseFloat(strings.TrimSuffix(total, "%"), 64); err != nil {
			c.add(path, "%q is not a percentage such as +12%%", total)
		}
	default:
		if _, err := strconv.ParseFloat(total, 64); err != nil {
			c.add(path, "%q is not tendered, original, a percentage or an amount", total)
		}
	}
}

func (c *checker) action(path string, a *Action) {
	if a.verbs() != 1 {
		c.add(path, "takes exactly one of decide, dispatch, arrive, depart, deliver, invoice, "+
			"release and tender")
		return
	}
	verb, about := a.Verb()
	if verb == "tender" {
		if !namePattern.MatchString(about) || c.named[about] {
			c.add(path+".tender", "a tender names a new conversation (lowercase, such as c1)")
		}
		c.named[about] = true
		if a.Rate != "" {
			if _, err := strconv.ParseFloat(a.Rate, 64); err != nil {
				c.add(path+".rate", "%q is not an amount", a.Rate)
			}
		}
		return
	}
	c.known(path+"."+verb, about)
	if verb == "decide" && !slices.Contains(decisions, a.Decision) {
		c.add(path+".decision", "%q is not one of %v", a.Decision, decisions)
	}
	if (verb == "arrive" || verb == "depart") && a.Stop < 1 {
		c.add(path+".stop", "names the stop by its place on the load, from 1")
	}
}

func nextSeed(taken map[uint64]string, count uint64) uint64 {
	for seed := uint64(1); ; seed++ {
		free := true
		for offset := range count {
			if _, used := taken[seed+offset]; used {
				free = false
				break
			}
		}
		if free {
			return seed
		}
	}
}

func cmpOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
