package oracle

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

const testISA = "ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*SIMPARTNER     " +
	"*261119*1200*U*00401*000000101*0*T*>"

func testGuides(t testing.TB) *GuideSet {
	t.Helper()
	gs, err := LoadGuides("testdata/guides")
	if err != nil {
		t.Fatalf("load guides: %v", err)
	}
	return gs
}

func readFixture(t testing.TB, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func transaction(set, control string, body ...string) []string {
	segs := append([]string{"ST*" + set + "*" + control}, body...)
	return append(segs, "SE*"+strconv.Itoa(len(segs)+1)+"*"+control)
}

func interchange(group string, segs ...string) string {
	all := []string{testISA, "GS*" + group + "*TRENOVA*SIMPARTNER*20261119*1200*101*X*004010"}
	all = append(all, segs...)
	sets := 0
	for _, s := range segs {
		if strings.HasPrefix(s, "ST*") {
			sets++
		}
	}
	all = append(all, "GE*"+strconv.Itoa(sets)+"*101", "IEA*1*000000101")
	return strings.Join(all, "~\n") + "~\n"
}

func judge(t testing.TB, text string) *Judgement {
	t.Helper()
	j, err := testGuides(t).Judge([]byte(text))
	if err != nil {
		t.Fatalf("judge: %v", err)
	}
	return j
}

func allFindings(j *Judgement) []Finding {
	out := append([]Finding(nil), j.Findings...)
	for _, ic := range j.Interchanges {
		out = append(out, ic.Findings...)
		for _, g := range ic.Groups {
			out = append(out, g.Findings...)
			for _, r := range g.Reports {
				out = append(out, r.Findings...)
			}
		}
	}
	return out
}

func findFinding(fs []Finding, level Level, code string) *Finding {
	for i := range fs {
		if fs[i].Level == level && fs[i].Code == code {
			return &fs[i]
		}
	}
	return nil
}

func errorSummary(fs []Finding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(string(f.Severity) + " " + string(f.Level) + " " + f.Code + " " +
			f.SegmentID + " " + f.Rule + ": " + f.Message + "\n")
	}
	return b.String()
}
