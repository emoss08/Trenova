package oracle

import (
	"strings"
	"testing"
)

func cleanSegments(t testing.TB) []string {
	t.Helper()
	text := strings.TrimSuffix(readFixture(t, "clean-4010.x12"), "~\n")
	return strings.Split(text, "~\n")
}

func squeezeISA(isa string) string {
	fields := strings.Split(isa, "*")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return strings.Join(fields, "*")
}

func render(segs []string, element, component, terminator string) string {
	var b strings.Builder
	for _, s := range segs {
		s = strings.ReplaceAll(s, ">", component)
		b.WriteString(strings.ReplaceAll(s, "*", element))
		b.WriteString(terminator)
	}
	return b.String()
}

func TestCleanFileIsAccepted(t *testing.T) {
	t.Parallel()

	j := judge(t, readFixture(t, "clean-4010.x12"))
	if !j.Accepted() {
		t.Fatalf("clean file rejected:\n%s", errorSummary(allFindings(j)))
	}
	if len(j.Interchanges) != 1 || len(j.Interchanges[0].Groups) != 3 {
		t.Fatalf("want 1 interchange with 3 groups, got %+v", j.Interchanges)
	}
	reports := j.Reports()
	want := []struct{ set, control, guide string }{
		{"990", "0001", "Test partner 990, v1"},
		{"990", "0002", "Test partner 990, v1"},
		{"214", "0003", "Test partner 214, v1"},
		{"210", "0004", "Test partner 210, v1"},
	}
	if len(reports) != len(want) {
		t.Fatalf("want %d reports, got %d", len(want), len(reports))
	}
	for i, w := range want {
		r := reports[i]
		if r.Set != w.set || r.ControlNumber != w.control || r.Guide != w.guide || !r.Accepted() {
			t.Errorf(
				"report %d: got %s %s %q accepted=%v",
				i,
				r.Set,
				r.ControlNumber,
				r.Guide,
				r.Accepted(),
			)
		}
	}
	at7 := reports[2].Segments[4]
	if at7.ID != "AT7" || at7.Index != 5 || at7.LoopID != "0205" || at7.Element(1) != "X3" {
		t.Errorf("214 AT7 placement: %+v", at7)
	}
	n9 := reports[0].Segments[3]
	if comps := n9.Components(7); len(comps) != 4 || comps[1] != "12345" {
		t.Errorf("N9-07 components: %q", comps)
	}
	if lx := reports[3].Segments[9]; lx.ID != "LX" || lx.LoopID != "0400" {
		t.Errorf("210 LX placement: %+v", lx)
	}
}

func TestSeparatorVariants(t *testing.T) {
	t.Parallel()

	segs := cleanSegments(t)
	squeezed := append([]string{squeezeISA(segs[0])}, segs[1:]...)
	tests := []struct {
		name     string
		text     string
		squeezed bool
	}{
		{name: "tilde newline", text: render(segs, "*", ">", "~\n")},
		{name: "tilde crlf", text: render(segs, "*", ">", "~\r\n")},
		{name: "no line breaks", text: render(segs, "*", ">", "~")},
		{name: "pipe and caret", text: render(segs, "|", "^", "~\r\n")},
		{name: "newline terminator", text: render(segs, "*", ">", "\n")},
		{name: "crlf terminator", text: render(segs, "*", ":", "\r\n")},
		{name: "indented with blank lines", text: "\n  " + render(segs, "*", ">", "~\n\n  ")},
		{name: "squeezed isa", text: render(squeezed, "*", ">", "~"), squeezed: true},
		{
			name:     "squeezed isa pipes crlf",
			text:     render(squeezed, "|", "^", "~\r\n"),
			squeezed: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			j := judge(t, tt.text)
			fs := allFindings(j)
			if !j.Accepted() {
				t.Fatalf("rejected:\n%s", errorSummary(fs))
			}
			if got := len(j.Reports()); got != 4 {
				t.Fatalf("want 4 transactions, got %d", got)
			}
			warned := false
			for _, f := range fs {
				warned = warned || (f.Severity == SeverityWarning && f.SegmentID == "ISA")
			}
			if warned != tt.squeezed {
				t.Fatalf("squeezed warning = %v, want %v", warned, tt.squeezed)
			}
		})
	}
}

func TestRepetitionSeparator5010(t *testing.T) {
	t.Parallel()

	text := readFixture(t, "clean-5010.x12")
	j := judge(t, text)
	if !j.Accepted() {
		t.Fatalf("rejected:\n%s", errorSummary(allFindings(j)))
	}
	k1 := j.Reports()[0].Segments[2]
	if reps := k1.Repetitions(1); len(reps) != 2 || reps[1] != "SECOND" {
		t.Fatalf("K1-01 repetitions: %q", reps)
	}

	over := judge(t, strings.Replace(text, "FIRST^SECOND", "A^B^C", 1))
	f := findFinding(allFindings(over), LevelElement, codeTooManyRepetitions)
	if f == nil || f.Rule != "K1-01 repeat 2" {
		t.Fatalf("want too many repetitions, got:\n%s", errorSummary(allFindings(over)))
	}

	fixed := judge(t, strings.Replace(text, "B1*TRNV", "B1*ABCD", 1))
	f = findFinding(allFindings(fixed), LevelElement, codeInvalidCode)
	if f == nil || f.Rule != "B1-01 fixed TRNV" {
		t.Fatalf("want fixed value finding, got:\n%s", errorSummary(allFindings(fixed)))
	}

	in4010 := strings.Replace(strings.Replace(text, "00501", "00401", 1), "005010", "004010", 1)
	in4010 = strings.Replace(in4010, "*^*", "*U*", 1)
	j = judge(t, in4010)
	if k1 := j.Reports()[0].Segments[2]; len(k1.Repetitions(1)) != 1 {
		t.Fatalf("4010 must not split repetitions: %q", k1.Repetitions(1))
	}
}

func TestNotX12(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"", "hello", "ISA", "GS*FA*X~", "ISA*00*"} {
		if _, err := testGuides(t).Judge([]byte(text)); err == nil {
			t.Errorf("%q: want an error", text)
		}
	}
}

func FuzzJudge(f *testing.F) {
	gs := testGuides(f)
	segs := cleanSegments(f)
	f.Add([]byte(readFixture(f, "clean-4010.x12")))
	f.Add([]byte(readFixture(f, "clean-5010.x12")))
	f.Add([]byte(render(segs, "|", "^", "~\r\n")))
	f.Add([]byte(render(append([]string{squeezeISA(segs[0])}, segs[1:]...), "*", ">", "\n")))
	f.Add([]byte("ISA*00**00**ZZ*A*ZZ*B*261119*1200*^*00501*1*0*T*:~ST*990~SE~IEA~"))
	f.Add([]byte("ISA*1*2*3*4*5*6*7*8*9*0*1*2*3*4*5*6~"))
	f.Add([]byte("ISA~~~~~~~~~~~~~~~~~~~~~"))
	f.Add(
		[]byte(
			"ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*SIMPARTNER     *261119*1200*U*00401*000000101*0*T*>",
		),
	)
	f.Fuzz(func(t *testing.T, data []byte) {
		j, err := gs.Judge(data)
		if err != nil {
			return
		}
		for _, r := range j.Reports() {
			for i := range r.Segments {
				s := &r.Segments[i]
				for p := -1; p <= s.ElementCount()+1; p++ {
					_ = s.Repetitions(p)
					_ = s.Components(p)
				}
			}
			_ = r.Accepted()
		}
		_ = j.Accepted()
	})
}
