package oracle

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

const goodGuide = `set: "990"
version: "004010"
group: GF
segments:
  ST:
    elements:
      - { pos: 1, type: ID, min: 3, max: 3, usage: M }
      - { pos: 2, type: AN, min: 4, max: 9, usage: M }
  B1:
    elements:
      - { pos: 1, type: ID, min: 2, max: 4, usage: M }
      - { pos: 4, type: ID, min: 1, max: 1, usage: M, codes: [A, D] }
  SE:
    elements:
      - { pos: 1, type: N0, min: 1, max: 10, usage: M }
      - { pos: 2, type: AN, min: 4, max: 9, usage: M }
structure:
  - { seg: ST, usage: M }
  - { seg: B1, usage: M }
  - { seg: SE, usage: M }
`

func TestGuideProblems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		old  string
		new  string
		path string
		want string
	}{
		{
			name: "unknown segment in structure",
			old:  "{ seg: B1, usage: M }",
			new:  "{ seg: B9, usage: M }",
			path: "structure[1].seg",
			want: "segment B9 is not defined",
		},
		{
			name: "duplicate element position",
			old:  "{ pos: 4, type: ID",
			new:  "{ pos: 1, type: ID",
			path: "segments.B1.elements[1].pos",
			want: "listed twice",
		},
		{
			name: "min over max",
			old:  "{ pos: 1, type: ID, min: 2, max: 4",
			new:  "{ pos: 1, type: ID, min: 5, max: 4",
			path: "segments.B1.elements[0].min",
			want: "greater than max",
		},
		{
			name: "bad syntax note letter",
			old:  "  SE:\n",
			new:  "    syntax: [Q0104]\n  SE:\n",
			path: "segments.B1.syntax[0]",
			want: "does not start with",
		},
		{
			name: "syntax note too short",
			old:  "  SE:\n",
			new:  "    syntax: [P01]\n  SE:\n",
			path: "segments.B1.syntax[0]",
			want: "is not a syntax note",
		},
		{
			name: "syntax note names an unlisted element",
			old:  "  SE:\n",
			new:  "    syntax: [P0104, C0302]\n  SE:\n",
			path: "segments.B1.syntax[1]",
			want: "names element 03",
		},
		{
			name: "unknown type",
			old:  "{ pos: 1, type: ID, min: 2",
			new:  "{ pos: 1, type: XX, min: 2",
			path: "segments.B1.elements[0].type",
			want: "is not one of AN",
		},
		{
			name: "bad usage",
			old:  "{ seg: B1, usage: M }",
			new:  "{ seg: B1, usage: Q }",
			path: "structure[1].usage",
			want: "is not one of M, O, X, N",
		},
		{
			name: "code longer than max",
			old:  "codes: [A, D]",
			new:  "codes: [A, DD]",
			path: "segments.B1.elements[1].codes[1]",
			want: "does not fit",
		},
		{
			name: "fixed outside codes",
			old:  "codes: [A, D] }",
			new:  "codes: [A, D], fixed: X }",
			path: "segments.B1.elements[1].fixed",
			want: "is not one of the codes",
		},
		{
			name: "loop without trigger",
			old:  "  - { seg: SE, usage: M }",
			new:  "  - { loop: \"0100\", usage: O, body: [] }\n  - { seg: SE, usage: M }",
			path: "structure[2].body",
			want: "trigger segment",
		},
		{
			name: "structure without SE",
			old:  "  - { seg: SE, usage: M }",
			new:  "",
			path: "structure",
			want: "must start with ST and end with SE",
		},
		{
			name: "unknown key",
			old:  "group: GF",
			new:  "group: GF\nowner: someone",
			want: "owner",
		},
		{
			name: "missing set",
			old:  "set: \"990\"",
			new:  "",
			path: "set",
			want: "is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			text := strings.Replace(goodGuide, tt.old, tt.new, 1)
			if text == goodGuide {
				t.Fatal("the edit did not apply")
			}
			_, errs := parseGuide("partners/x/guide/990.yaml", []byte(text))
			joined := errors.Join(errs...)
			if joined == nil {
				t.Fatal("want a guide error")
			}
			msg := joined.Error()
			if !strings.Contains(msg, "partners/x/guide/990.yaml") ||
				!strings.Contains(msg, tt.path) || !strings.Contains(msg, tt.want) {
				t.Fatalf("want %q at %q, got:\n%s", tt.want, tt.path, msg)
			}
		})
	}
}

func TestGoodGuideLoads(t *testing.T) {
	t.Parallel()

	g, errs := parseGuide("990.yaml", []byte(goodGuide))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errors.Join(errs...))
	}
	if g.Segments["B1"].byPos[4].codeSet["D"] != true {
		t.Fatal("codes not indexed")
	}
}

func TestLoadGuidesFS(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"990.yaml":   {Data: []byte(goodGuide)},
		"notes.md":   {Data: []byte("ignored")},
		"other.yaml": {Data: []byte(goodGuide)},
	}
	if _, err := LoadGuidesFS(fsys); err == nil || !strings.Contains(err.Error(), "also in") {
		t.Fatalf("want a duplicate guide error, got %v", err)
	}
	delete(fsys, "other.yaml")
	gs, err := LoadGuidesFS(fsys)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := gs.Sets(); len(got) != 1 || got[0] != "990/004010" {
		t.Fatalf("sets: %v", got)
	}
}

func TestPickPrefersTheLongestVersion(t *testing.T) {
	t.Parallel()

	gs := &GuideSet{guides: []*guide{
		{Set: "999", Version: "005010"},
		{Set: "999", Version: "005010X231A1"},
	}}
	if g := gs.pick("999", "005010X231A1"); g == nil || g.Version != "005010X231A1" {
		t.Fatalf("picked %+v", g)
	}
	if g := gs.pick("999", "005010X999"); g == nil || g.Version != "005010" {
		t.Fatalf("picked %+v", g)
	}
	if g := gs.pick("999", "004010"); g != nil {
		t.Fatalf("picked %+v", g)
	}
}

func TestScalarsKeepTheirText(t *testing.T) {
	t.Parallel()

	text := strings.NewReplacer(
		`version: "004010"`, "version: 004010",
		"codes: [A, D] }", "codes: [A, D], fixed: null }\n"+
			"      - { pos: 5, type: ID, min: 2, max: 3, usage: O, codes: [01, 400, '07'], fixed: 01 }",
	).Replace(goodGuide)
	g, errs := parseGuide("990.yaml", []byte(text))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errors.Join(errs...))
	}
	el := g.Segments["B1"].byPos[5]
	if g.Segments["B1"].byPos[4].Fixed != nil {
		t.Fatal("fixed: null must leave the element unfixed")
	}
	if g.Version != "004010" || scalarsText(el.Codes) != "01 400 07" || *el.Fixed != "01" {
		t.Fatalf("version %q codes %q fixed %q", g.Version, el.Codes, *el.Fixed)
	}
}
