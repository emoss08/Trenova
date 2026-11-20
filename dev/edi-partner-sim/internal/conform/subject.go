package conform

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/pkg/lading"
)

type subject struct {
	def     *partner.Definition
	project *lading.Encoded
	check   *lading.Check
	guides  *oracle.GuideSet
	report  *PartnerReport
	runner  *runner
}

func (s *subject) name() string {
	return s.def.Name()
}

func (s *subject) fail(rule, message string) {
	s.report.Failures = append(s.report.Failures, Failure{
		Step:    "project",
		Source:  SourceProject,
		Rule:    rule,
		Message: message,
		Replay:  s.name(),
	})
}

func encodeProject(def *partner.Definition) (*lading.Encoded, error) {
	files, samples, err := def.LadingProject()
	if err != nil {
		return nil, err
	}
	project := &lading.Project{}
	for _, file := range files {
		project.Files = append(project.Files, lading.File{Path: file.Path, Source: file.Content})
	}
	for _, sample := range samples {
		project.Samples = append(
			project.Samples,
			lading.Sample{Path: sample.Path, Content: sample.Content},
		)
	}
	return lading.Encode(project, &lading.Context{Org: lading.Org{
		Name: "Trenova",
		SCAC: def.Partner.Trenova.SCAC,
		Zone: def.Partner.Zone,
	}})
}

func (r *runner) prepare(ctx context.Context, def *partner.Definition) *subject {
	s := &subject{
		def:    def,
		runner: r,
		report: &PartnerReport{Name: def.Name(), Role: string(def.Partner.Role)},
	}
	guides, err := oracle.LoadGuides(filepath.Join(def.Dir, "guide"))
	if err != nil {
		s.fail("partner guide", err.Error())
		return s
	}
	s.guides = guides
	if s.project, err = encodeProject(def); err != nil {
		s.fail("lading project", err.Error())
		return s
	}
	if s.check, err = r.client.Check(ctx, s.project); err != nil {
		s.fail("lading check", err.Error())
		return s
	}
	s.recordCheck()
	if s.check.Errors > 0 {
		return s
	}
	s.runTests(ctx)
	s.checkReceives()
	return s
}

func (s *subject) ready() bool {
	return s.check != nil && s.check.Errors == 0 && s.guides != nil
}

func (s *subject) recordCheck() {
	p := &s.report.Project
	p.Errors, p.Warnings = s.check.Errors, s.check.Warnings
	for fi := range s.check.Files {
		file := &s.check.Files[fi]
		for di := range file.Diagnostics {
			d := &file.Diagnostics[di]
			if d.Severity != lading.SeverityError {
				continue
			}
			text := fmt.Sprintf("%s %d..%d: %s: %s",
				file.Path, d.Range.From, d.Range.To, d.Code, d.Message.String())
			p.Diagnostics = append(p.Diagnostics, text)
			s.fail("lading check", text)
		}
	}
}

func (s *subject) runTests(ctx context.Context) {
	report, err := s.runner.client.Test(ctx, s.project)
	if err != nil {
		s.fail("lading test", err.Error())
		return
	}
	p := &s.report.Project
	p.Tests, p.Ready = len(report.Tests), report.Readiness.Ready
	for i := range report.Tests {
		result := &report.Tests[i]
		if result.Passed {
			continue
		}
		var text strings.Builder
		fmt.Fprintf(&text, "%s: %q", result.Path, result.Name)
		for _, failure := range result.Failures {
			text.WriteString("; " + failure.Message.String())
		}
		p.FailedTests = append(p.FailedTests, text.String())
		s.fail("lading test", text.String())
	}
}

func (s *subject) checkReceives() {
	for _, set := range s.def.Partner.Receives {
		if set == "997" || set == "999" {
			continue
		}
		if !s.hasGuide(set) {
			s.fail("partner guide", fmt.Sprintf(
				"partner.yaml says the partner receives %s but guide/ has no guide for it", set))
		}
		if s.outboundMap(set) == "" {
			s.fail("lading project", fmt.Sprintf(
				"partner.yaml says the partner receives %s but no map writes it", set))
		}
	}
}

func (s *subject) hasGuide(set string) bool {
	return slices.ContainsFunc(s.guides.Sets(), func(entry string) bool {
		return strings.HasPrefix(entry, set+"/")
	})
}

func (s *subject) outboundMap(set string) string {
	for fi := range s.check.Flows {
		for ci := range s.check.Flows[fi].Clauses {
			clause := &s.check.Flows[fi].Clauses[ci]
			if clause.Set == set && clause.Map != "" &&
				(clause.Kind == lading.ClauseSend || clause.Kind == lading.ClauseRespond) {
				return clause.Map
			}
		}
	}
	for i := range s.check.Maps {
		if m := &s.check.Maps[i]; m.Direction == lading.DirectionOutbound && m.Set == set {
			return m.Name
		}
	}
	return ""
}

func (s *subject) inboundMap(set string) string {
	if _, clause := s.check.Receiving(set); clause != nil && clause.Map != "" {
		return clause.Map
	}
	for i := range s.check.Maps {
		if m := &s.check.Maps[i]; m.Direction == lading.DirectionInbound && m.Set == set {
			return m.Name
		}
	}
	return ""
}

func (s *subject) sendsOn(set string) func(event string) bool {
	var events []string
	clauses := 0
	for fi := range s.check.Flows {
		for ci := range s.check.Flows[fi].Clauses {
			clause := &s.check.Flows[fi].Clauses[ci]
			if clause.Kind == lading.ClauseSend && clause.Set == set {
				clauses++
				events = append(events, clause.Events...)
			}
		}
	}
	return func(event string) bool {
		return clauses == 0 || slices.Contains(events, event)
	}
}
