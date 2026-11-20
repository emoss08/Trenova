package reportload

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/conform"
	"github.com/emoss08/trenova/edi-partner-sim/internal/ladder"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
)

const (
	partnerEnvelope = "ISA*00*          *00*          *ZZ*ACMESHIP       *ZZ*TRENOVA        " +
		"*260302*0900*^*00401*000000501*0*T*>~\nGS*SM*ACMESHIP*TRENOVA*20260302*0900*501*X*004010~\n"
	tender = partnerEnvelope + "ST*204*0001~\nB2**TRNV**ACM-1**PP~\nB2A*00*LT~\n" +
		"L3*42000*G***185000******24*L~\nSE*5*0001~\nGE*1*501~\nIEA*1*000000501~\n"
	ack = "ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*ACMESHIP       " +
		"*260302*0901*^*00401*000000912*0*T*>~\nGS*FA*TRENOVA*ACMESHIP*20260302*0901*912*X*004010~\n" +
		"ST*997*0001~\nAK1*SM*501~\nAK2*204*0001~\nAK5*A~\nAK9*A*1*1*1~\nSE*6*0001~\n" +
		"GE*1*912~\nIEA*1*000000912~\n"
	answer = "ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*ACMESHIP       " +
		"*260302*0902*^*00401*000000913*0*T*>~\nGS*GF*TRENOVA*ACMESHIP*20260302*0902*913*X*004010~\n" +
		"ST*990*0001~\nB1*TRNV*ACM-1*20260302*D~\nSE*3*0001~\nGE*1*913~\nIEA*1*000000913~\n"
)

func newServer(t *testing.T) *sim.Server {
	t.Helper()
	definition, err := partner.Open("../play/testdata/partners/acme")
	if err != nil {
		t.Fatalf("open the fixture: %v", err)
	}
	server, err := sim.New(sim.Options{
		Partners:    []*partner.Definition{definition},
		IdentityDir: t.TempDir(),
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}
	return server
}

func write(t *testing.T, name string, report any) string {
	t.Helper()
	data, err := sonic.Marshal(report)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err = os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestARunReportBecomesARunWithItsWire(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 3, 2, 15, 0, 0, 0, time.UTC)
	report := &play.Report{Run: 12, API: "http://localhost:8080/api/v1", StartedAt: started,
		Partners: []*play.PartnerReport{
			{Name: "acme", Scenarios: []*play.ScenarioReport{{
				Name: "happy", Slug: "happy", Status: play.StatusPassed, StartedAt: started,
				Replay: "edi-sim run --partner acme --scenario happy --run 12",
				Steps: []*play.StepReport{
					{Index: 0, Kind: "send", Text: "send 204 original as t1",
						Status:  play.StatusPassed,
						Matches: []runlog.Match{{MessageID: "acme-000041", Key: "ACM-1"}}},
					{Index: 1, Kind: "expect", Text: "expect 997 for t1",
						Status:  play.StatusPassed,
						Matches: []runlog.Match{{MessageID: "acme-000042"}}},
				},
				Conversations: []*play.ConversationReport{{Key: "ACM-1", Wire: []play.WireEntry{
					{MessageID: "acme-000042", From: "trenova", Purpose: "ack",
						State: "received", At: started.Add(time.Minute), X12: ack},
					{MessageID: "acme-000041", From: "partner", Purpose: "document",
						State: "delivered", At: started, X12: tender},
				}}},
			}}},
			{Name: "elsewhere"},
		}}
	path := write(t, "run-report.json", report)
	server := newServer(t)
	result, err := Load(server, path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if result.Kind != runlog.KindRun || result.Messages != 2 || result.Scenarios != 1 ||
		len(result.Skipped) != 1 {
		t.Fatalf("result %+v", result)
	}
	current, _ := server.Runs().Current()
	if current.Source != runlog.SourceReport || current.File != "run-report.json" ||
		current.Number != 12 || current.Tally.Passed != 1 {
		t.Fatalf("the run %+v", current)
	}
	detail, err := server.Conversation("acme", "ACM-1")
	if err != nil || len(detail.Rows) != 2 {
		t.Fatalf("the conversation: %v %+v", err, detail.Rows)
	}
	if row := detail.Rows[0]; row.Set != "204" || row.Expect != ladder.ToneOK ||
		row.Verdict != ladder.VerdictRead {
		t.Fatalf("the 204 row %+v", row)
	}
	if row := detail.Rows[1]; row.Set != "997" || row.Expect != ladder.ToneOK || !row.Ack {
		t.Fatalf("the 997 row %+v", row)
	}
	again, err := Load(server, path)
	if err != nil || !again.Already {
		t.Fatalf("a second load: %+v %v", again, err)
	}
}

func TestAConformReportBecomesOneScenarioPerCase(t *testing.T) {
	t.Parallel()

	report := &conform.Report{Seed: 4821, Count: 1, Partners: []*conform.PartnerReport{{
		Name: "acme",
		Cases: []*conform.Case{{
			Partner: "acme", Seed: 4821, Variation: "original", Title: "Original tender",
			Steps: []*conform.Step{
				{Name: "204 in", Text: "Read Acme's 204", Direction: "inbound", Set: "204",
					Status: conform.StatusRead,
					X12:    tender},
				{Name: "990", Direction: "outbound", Set: "990", Status: conform.StatusFailed,
					X12: answer},
			},
			Failures: []conform.Failure{{Step: "990", Set: "990", Source: conform.SourceOracle,
				Rule: "B1-04", Message: "wanted A"}},
		}},
	}}}
	server := newServer(t)
	result, err := Load(server, write(t, "conform.json", report))
	if err != nil || result.Kind != runlog.KindConform || result.Messages != 2 {
		t.Fatalf("load: %+v %v", result, err)
	}
	current, _ := server.Runs().Current()
	if current.Kind != runlog.KindConform || current.Seed != 4821 || current.Tally.Failed != 1 {
		t.Fatalf("the run %+v", current)
	}
	detail, _ := server.Conversation("acme", "ACM-1")
	if detail.Scenario == nil || detail.Scenario.Name != "Original tender · seed 4821" ||
		detail.Scenario.Steps[0].Text != "Read Acme's 204" || detail.Scenario.Replay != "edi-sim conform --replay acme:4821" ||
		detail.Scenario.Steps[1].Why != "B1-04: wanted A" || len(detail.Rows) != 2 ||
		detail.Rows[1].Expect != ladder.ToneBad || detail.Rows[0].Expect != ladder.ToneOK {
		t.Fatalf("the case: %+v %+v", detail.Scenario, detail.Rows)
	}
}

func TestASoakReportIsRefused(t *testing.T) {
	t.Parallel()

	path := write(t, "soak.json", map[string]any{"run": 3, "partners": []any{
		map[string]any{"name": "acme", "milestones": map[string]any{}},
	}})
	if _, err := Load(newServer(t), path); !errors.Is(err, ErrSoakReport) {
		t.Fatalf("a soak report: %v", err)
	}
	path = write(t, "other.json", map[string]any{"hello": "world"})
	if _, err := Load(newServer(t), path); !errors.Is(err, ErrUnknownReport) {
		t.Fatalf("another file: %v", err)
	}
}
