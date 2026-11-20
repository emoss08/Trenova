package oracle

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

const overrideGuide = `set: "214"
version: "004010"
group: QM
segments:
  ST:
    elements:
      - { pos: 1, type: ID, min: 3, max: 3, usage: M }
      - { pos: 2, type: AN, min: 4, max: 9, usage: M }
  B10:
    elements:
      - { pos: 1, type: AN, min: 1, max: 30, usage: M }
      - { pos: 2, type: AN, min: 1, max: 30, usage: M }
      - { pos: 3, type: ID, min: 2, max: 4, usage: O }
  L11:
    elements:
      - { pos: 1, type: AN, min: 1, max: 30, usage: M }
      - { pos: 2, type: ID, min: 2, max: 3, usage: O, codes: [BM, PO, QN] }
  LX:
    elements:
      - { pos: 1, type: N0, min: 1, max: 6, usage: M }
  AT7:
    elements:
      - { pos: 1, type: ID, min: 2, max: 2, usage: M }
      - { pos: 5, type: DT, min: 8, max: 8, usage: M }
  SE:
    elements:
      - { pos: 1, type: N0, min: 1, max: 10, usage: M }
      - { pos: 2, type: AN, min: 4, max: 9, usage: M }
structure:
  - { seg: ST, usage: M }
  - { seg: B10, usage: M }
  - seg: L11
    usage: O
    max: 2
    elements:
      - { pos: 2, usage: M, codes: [BM, PO] }
  - loop: "0205"
    usage: M
    max: 5
    body:
      - { seg: LX, usage: M }
      - { seg: AT7, usage: M }
      - seg: L11
        usage: O
        elements:
          - { pos: 2, codes: [QN] }
  - { seg: SE, usage: M }
`

func TestOverrideProblems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		old  string
		new  string
		path string
		want string
	}{
		{
			name: "unknown position",
			old:  "{ pos: 2, codes: [QN] }",
			new:  "{ pos: 7, codes: [QN] }",
			path: "structure[3].body[2].elements[0].pos",
			want: "L11 lists no element 07 to override",
		},
		{
			name: "position overridden twice",
			old:  "{ pos: 2, codes: [QN] }",
			new:  "{ pos: 2, codes: [QN] }\n          - { pos: 2, max: 2 }",
			path: "structure[3].body[2].elements[1].pos",
			want: "overridden twice",
		},
		{
			name: "code outside the segment's codes",
			old:  "codes: [QN]",
			new:  "codes: [QN, ZZ]",
			path: "structure[3].body[2].elements[0].codes[1]",
			want: `"ZZ" is not one of L11-02's codes`,
		},
		{
			name: "relaxing a mandatory element",
			old:  "{ pos: 2, codes: [QN] }",
			new:  "{ pos: 1, usage: O }",
			path: "structure[3].body[2].elements[0].usage",
			want: "cannot relax it to O",
		},
		{
			name: "max outside the segment's",
			old:  "{ pos: 2, codes: [QN] }",
			new:  "{ pos: 1, max: 40 }",
			path: "structure[3].body[2].elements[0].max",
			want: "40 is outside L11-01's 1..30",
		},
		{
			name: "a code that no longer fits",
			old:  "{ pos: 2, codes: [QN] }",
			new:  "{ pos: 2, min: 3, codes: [QN] }",
			path: "structure[3].body[2].elements[0].codes[0]",
			want: "does not fit min 3 max 3",
		},
		{
			name: "elements on a loop",
			old:  "    max: 5\n",
			new:  "    max: 5\n    elements: [{ pos: 1 }]\n",
			path: "structure[3].elements",
			want: "a loop has no elements to override",
		},
		{
			name: "unknown key",
			old:  "{ pos: 2, codes: [QN] }",
			new:  "{ pos: 2, type: AN }",
			want: "type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			text := strings.Replace(overrideGuide, tt.old, tt.new, 1)
			if text == overrideGuide {
				t.Fatal("the edit did not apply")
			}
			_, errs := parseGuide("214.yaml", []byte(text))
			joined := errors.Join(errs...)
			if joined == nil {
				t.Fatal("want a guide error")
			}
			if msg := joined.Error(); !strings.Contains(msg, tt.path) ||
				!strings.Contains(msg, tt.want) {
				t.Fatalf("want %q at %q, got:\n%s", tt.want, tt.path, msg)
			}
		})
	}
}

func TestOverridesNarrowEachUse(t *testing.T) {
	t.Parallel()

	gs, err := LoadGuidesFS(mapFS("214.yaml", overrideGuide))
	if err != nil {
		t.Fatal(err)
	}
	judgeOne := func(t *testing.T, body ...string) []Finding {
		t.Helper()
		j, err := gs.Judge([]byte(interchange("QM", transaction("214", "0001", body...)...)))
		if err != nil {
			t.Fatal(err)
		}
		return allFindings(j)
	}

	t.Run("each use takes its own codes", func(t *testing.T) {
		t.Parallel()
		fs := judgeOne(t, "B10*PRO1*L1", "L11*8100*BM", "L11*4500*PO",
			"LX*1", "AT7*X1****20261119", "L11*2*QN")
		if len(fs) > 0 {
			t.Fatalf("want no findings, got:\n%s", errorSummary(fs))
		}
	})

	t.Run("a heading code inside the loop is a finding", func(t *testing.T) {
		t.Parallel()
		fs := judgeOne(t, "B10*PRO1*L1", "LX*1", "AT7*X1****20261119", "L11*8100*BM")
		f := findFinding(fs, LevelElement, codeInvalidCode)
		if f == nil || f.Rule != "L11-02 codes [QN]" || f.LoopID != "0205" {
			t.Fatalf("want L11-02 codes [QN] in 0205, got:\n%s", errorSummary(fs))
		}
	})

	t.Run("a loop code in the heading is a finding", func(t *testing.T) {
		t.Parallel()
		fs := judgeOne(t, "B10*PRO1*L1", "L11*2*QN", "LX*1", "AT7*X1****20261119")
		f := findFinding(fs, LevelElement, codeInvalidCode)
		if f == nil || f.Rule != "L11-02 codes [BM PO]" {
			t.Fatalf("want L11-02 codes [BM PO], got:\n%s", errorSummary(fs))
		}
	})

	t.Run("a tightened usage applies to its use only", func(t *testing.T) {
		t.Parallel()
		fs := judgeOne(t, "B10*PRO1*L1", "L11*8100", "LX*1", "AT7*X1****20261119", "L11*2")
		f := findFinding(fs, LevelElement, codeElementMissing)
		if f == nil || f.Rule != "L11-02 usage M" || f.SegmentIndex != 3 {
			t.Fatalf("want L11-02 usage M on the heading L11, got:\n%s", errorSummary(fs))
		}
		if n := countCode(fs, codeElementMissing); n != 1 {
			t.Fatalf("want one missing element, got %d:\n%s", n, errorSummary(fs))
		}
	})
}

func TestTheFormatExampleIsAValidGuide(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../partners/GUIDE-FORMAT.md")
	if err != nil {
		t.Fatal(err)
	}
	example := firstGuideBlock(string(raw))
	if example == "" {
		t.Fatal("GUIDE-FORMAT.md has no yaml block starting with set:")
	}
	g, errs := parseGuide("GUIDE-FORMAT.md", []byte(example))
	if len(errs) > 0 {
		t.Fatalf("the example fails the guide check:\n%v", errors.Join(errs...))
	}
	if len(g.Structure) == 0 {
		t.Fatal("the example has no structure")
	}
}

func TestEveryPartnerGuideLoads(t *testing.T) {
	t.Parallel()

	dirs, err := filepath.Glob("../../partners/*/guide")
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) == 0 {
		t.Fatal("no partner guides")
	}
	for _, dir := range dirs {
		if _, err := LoadGuides(dir); err != nil {
			t.Errorf("%s:\n%v", dir, err)
		}
	}
}

func firstGuideBlock(markdown string) string {
	for _, block := range strings.Split(markdown, "```yaml\n")[1:] {
		end := strings.Index(block, "```")
		if end < 0 {
			continue
		}
		if body := block[:end]; strings.HasPrefix(body, "set:") {
			return body
		}
	}
	return ""
}

func countCode(fs []Finding, code string) int {
	n := 0
	for i := range fs {
		if fs[i].Code == code && fs[i].Severity == SeverityError {
			n++
		}
	}
	return n
}

func mapFS(name, text string) fstest.MapFS {
	return fstest.MapFS{name: {Data: []byte(text)}}
}
