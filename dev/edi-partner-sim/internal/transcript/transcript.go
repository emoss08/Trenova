package transcript

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
)

type Options struct {
	Judged    bool
	Variation string
	Resend    bool
	Earlier   func(key string) (int64, bool)
}

type keyAt struct {
	segment string
	element int
}

var keyElements = map[string]keyAt{
	"204": {segment: "B2", element: 4},
	"990": {segment: "B1", element: 2},
	"214": {segment: "B10", element: 2},
	"210": {segment: "B3", element: 3},
}

var chargeElements = map[string]keyAt{
	"204": {segment: "L3", element: 5},
	"210": {segment: "B3", element: 7},
}

func KeyOf(rep *oracle.Report) string {
	at, ok := keyElements[rep.Set]
	if !ok {
		return ""
	}
	return strings.TrimSpace(first(rep.Segments, at.segment).Element(at.element))
}

func Build(j *oracle.Judgement, o Options) []journal.Document {
	if j == nil {
		return nil
	}
	var out []journal.Document
	for _, ic := range j.Interchanges {
		for _, g := range ic.Groups {
			for _, rep := range g.Reports {
				out = append(out, document(ic, g, rep, &o))
			}
		}
		for index := range ic.TA1 {
			out = append(out, ta1Document(ic, &ic.TA1[index], &o))
		}
	}
	if len(out) > 0 && len(j.Findings) > 0 {
		out[0].Findings = append(notes(j.Findings, ""), out[0].Findings...)
	}
	return out
}

func document(
	ic *oracle.Interchange,
	g *oracle.Group,
	rep *oracle.Report,
	o *Options,
) journal.Document {
	d := journal.Document{
		Set:          rep.Set,
		Control:      rep.ControlNumber,
		GroupID:      g.FunctionalID,
		GroupControl: g.ControlNumber,
		Interchange:  ic.ControlNumber,
		Header:       rows(&ic.Header, &g.Header),
		Segments:     make([][]string, 0, len(rep.Segments)),
		Trailer:      rows(&g.Trailer, &ic.Trailer),
		Guide:        rep.Guide,
		GuideFile:    relative(rep.GuideFile),
		Judged:       o.Judged,
		ChargeCents:  charge(rep),
	}
	d.Separator, d.Terminator = separators(&ic.Header, rep)
	for index := range rep.Segments {
		d.Segments = append(d.Segments, rep.Segments[index].Elements())
	}
	if key := KeyOf(rep); key != "" {
		d.Keys = []string{key}
	}
	if o.Judged {
		file := relative(rep.GuideFile)
		d.Findings = append(d.Findings, notes(ic.Findings, file)...)
		d.Findings = append(d.Findings, notes(g.Findings, file)...)
		d.Findings = append(d.Findings, notes(rep.Findings, file)...)
		accepted := rep.Accepted() && !hasErrors(ic.Findings) && !hasErrors(g.Findings)
		d.Accepted = &accepted
	}
	if received, ok := acknowledge.Read(rep); ok {
		d.Acknowledges, d.AckCode = acknowledged(received)
	}
	d.Title = title(rep, &d, o)
	return d
}

func ta1Document(ic *oracle.Interchange, segment *oracle.Segment, o *Options) journal.Document {
	read := acknowledge.ReadTA1(&oracle.Interchange{TA1: []oracle.Segment{*segment}})[0]
	d := journal.Document{
		Set:         journal.SetTA1,
		Control:     read.Control,
		Interchange: ic.ControlNumber,
		Header:      rows(&ic.Header),
		Segments:    [][]string{segment.Elements()},
		Trailer:     rows(&ic.Trailer),
		Judged:      o.Judged,
		AckCode:     read.Code,
		Acknowledges: []journal.Acked{{
			Set:         journal.SetTA1,
			Interchange: read.Control,
			Code:        read.Code,
		}},
	}
	d.Separator, d.Terminator = separators(&ic.Header, nil)
	d.Title = interchangeWord(read.Code) + " " + trimControl(read.Control)
	return d
}

func acknowledged(received acknowledge.Received) ([]journal.Acked, string) {
	if len(received.Transactions) == 0 {
		return []journal.Acked{{
			Set:          received.Set,
			GroupID:      received.GroupID,
			GroupControl: received.GroupControl,
			Code:         received.Code,
		}}, received.Code
	}
	out := make([]journal.Acked, 0, len(received.Transactions))
	code := ""
	for _, t := range received.Transactions {
		out = append(out, journal.Acked{
			Set:          t.Set,
			GroupID:      received.GroupID,
			GroupControl: received.GroupControl,
			Control:      t.Control,
			Code:         t.Code,
		})
		code = Worse(code, t.Code)
	}
	if code == "" {
		code = received.Code
	}
	return out, code
}

var ackSeverity = map[string]int{"A": 1, "E": 2, "P": 3, "X": 3, "M": 3, "W": 3, "R": 4}

func Worse(a, b string) string {
	if ackSeverity[b] > ackSeverity[a] {
		return b
	}
	return a
}

func notes(findings []oracle.Finding, file string) []journal.Note {
	out := make([]journal.Note, 0, len(findings))
	for index := range findings {
		f := &findings[index]
		out = append(out, journal.Note{
			Severity:  string(f.Severity),
			Level:     string(f.Level),
			Segment:   f.SegmentIndex,
			SegmentID: f.SegmentID,
			Element:   f.ElementPos,
			Component: f.ComponentPos,
			Value:     f.Value,
			Code:      f.X12Code(),
			Rule:      f.Rule,
			Message:   f.Message,
			File:      file,
		})
	}
	return out
}

func hasErrors(findings []oracle.Finding) bool {
	return slices.ContainsFunc(findings, func(f oracle.Finding) bool {
		return f.Severity == oracle.SeverityError
	})
}

func rows(segments ...*oracle.Segment) [][]string {
	out := make([][]string, 0, len(segments))
	for _, s := range segments {
		if s.ID != "" {
			out = append(out, s.Elements())
		}
	}
	return out
}

func separators(header *oracle.Segment, rep *oracle.Report) (string, string) {
	if header.ID != "" {
		return header.Separator(), header.Terminator()
	}
	if rep != nil && len(rep.Segments) > 0 {
		return rep.Segments[0].Separator(), rep.Segments[0].Terminator()
	}
	return "*", "~"
}

func first(segments []oracle.Segment, id string) *oracle.Segment {
	for index := range segments {
		if segments[index].ID == id {
			return &segments[index]
		}
	}
	return &oracle.Segment{}
}

func charge(rep *oracle.Report) int64 {
	at, ok := chargeElements[rep.Set]
	if !ok {
		return 0
	}
	value := strings.TrimSpace(first(rep.Segments, at.segment).Element(at.element))
	if value == "" {
		return 0
	}
	if whole, fraction, dotted := strings.Cut(value, "."); dotted {
		value = whole + (fraction + "00")[:2]
	}
	cents, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return cents
}

func trimControl(control string) string {
	trimmed := strings.TrimLeft(strings.TrimSpace(control), "0")
	if trimmed == "" {
		return strings.TrimSpace(control)
	}
	return trimmed
}

var (
	workingDir     string
	workingDirOnce sync.Once
)

func relative(file string) string {
	if file == "" || !filepath.IsAbs(file) {
		return filepath.ToSlash(file)
	}
	workingDirOnce.Do(func() { workingDir, _ = os.Getwd() })
	if workingDir == "" {
		return filepath.ToSlash(file)
	}
	rel, err := filepath.Rel(workingDir, file)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(file)
	}
	return filepath.ToSlash(rel)
}
