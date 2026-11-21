package oracle

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

type guide struct {
	Set       scalar                 `yaml:"set"`
	Version   scalar                 `yaml:"version"`
	Group     string                 `yaml:"group"`
	Title     string                 `yaml:"title"`
	Segments  map[string]*segmentDef `yaml:"segments"`
	Structure []*entry               `yaml:"structure"`
	file      string
	inSet     map[string]bool
}

type segmentDef struct {
	Name     string        `yaml:"name"`
	Elements []*elementDef `yaml:"elements"`
	Syntax   []string      `yaml:"syntax"`
	id       string
	notes    []syntaxNote
	byPos    map[int]*elementDef
	lastPos  int
}

type elementDef struct {
	Pos        int           `yaml:"pos"`
	Name       string        `yaml:"name"`
	Type       string        `yaml:"type"`
	Min        int           `yaml:"min"`
	Max        int           `yaml:"max"`
	Usage      string        `yaml:"usage"`
	Codes      []scalar      `yaml:"codes"`
	Fixed      *scalar       `yaml:"fixed"`
	Repeat     int           `yaml:"repeat"`
	Components []*elementDef `yaml:"components"`
	ref        string
	dataType   dataType
	codeSet    map[string]bool
	byPos      map[int]*elementDef
	lastPos    int
}

type entry struct {
	Seg      string             `yaml:"seg"`
	Loop     string             `yaml:"loop"`
	Usage    string             `yaml:"usage"`
	Max      int                `yaml:"max"`
	Elements []*elementOverride `yaml:"elements"`
	Body     []*entry           `yaml:"body"`
	def      *segmentDef
}

type elementOverride struct {
	Pos    int      `yaml:"pos"`
	Usage  string   `yaml:"usage"`
	Min    *int     `yaml:"min"`
	Max    *int     `yaml:"max"`
	Codes  []scalar `yaml:"codes"`
	Fixed  *scalar  `yaml:"fixed"`
	Repeat *int     `yaml:"repeat"`
}

type GuideError struct {
	File    string
	Path    string
	Message string
}

func (e *GuideError) Error() string {
	if e.Path == "" {
		return e.File + ": " + e.Message
	}
	return e.File + ": " + e.Path + ": " + e.Message
}

type GuideSet struct {
	guides []*guide
}

func LoadGuides(dir string) (*GuideSet, error) {
	return loadGuides(os.DirFS(dir), dir)
}

func LoadGuidesFS(fsys fs.FS) (*GuideSet, error) {
	return loadGuides(fsys, "")
}

func loadGuides(fsys fs.FS, prefix string) (*GuideSet, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read guides: %w", err)
	}
	set := &GuideSet{}
	var errs []error
	for _, e := range entries {
		ext := path.Ext(e.Name())
		if e.IsDir() || (ext != ".yaml" && ext != ".yml") {
			continue
		}
		data, readErr := fs.ReadFile(fsys, e.Name())
		if readErr != nil {
			errs = append(errs, readErr)
			continue
		}
		g, gErrs := parseGuide(path.Join(prefix, e.Name()), data)
		if len(gErrs) > 0 {
			errs = append(errs, gErrs...)
			continue
		}
		if other := set.exact(string(g.Set), string(g.Version)); other != nil {
			errs = append(errs, &GuideError{
				File: g.file,
				Message: fmt.Sprintf(
					"set %s version %s is also in %s",
					g.Set,
					g.Version,
					other.file,
				),
			})
			continue
		}
		set.guides = append(set.guides, g)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return set, nil
}

func parseGuide(file string, data []byte) (*guide, []error) {
	g := &guide{}
	if err := yaml.UnmarshalWithOptions(data, g, yaml.Strict()); err != nil {
		return nil, []error{&GuideError{File: file, Message: yaml.FormatError(err, false, false)}}
	}
	g.file = file
	c := &guideChecker{g: g}
	c.check()
	if len(c.errs) > 0 {
		return nil, c.errs
	}
	return g, nil
}

func (s *GuideSet) exact(set, version string) *guide {
	for _, g := range s.guides {
		if g.Set == scalar(set) && g.Version == scalar(version) {
			return g
		}
	}
	return nil
}

func (s *GuideSet) pick(set, version string) *guide {
	if s == nil {
		return nil
	}
	var best *guide
	for _, g := range s.guides {
		if string(g.Set) != set || !strings.HasPrefix(version, string(g.Version)) {
			continue
		}
		if best == nil || len(g.Version) > len(best.Version) {
			best = g
		}
	}
	return best
}

func (s *GuideSet) Sets() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.guides))
	for _, g := range s.guides {
		out = append(out, string(g.Set+"/"+g.Version))
	}
	slices.Sort(out)
	return out
}

func (s *GuideSet) ElementMax(set, segment string, position int) int {
	if s == nil {
		return 0
	}
	for _, g := range s.guides {
		if string(g.Set) != set {
			continue
		}
		def := g.Segments[segment]
		if def == nil {
			return 0
		}
		limit := 0
		for _, element := range def.Elements {
			if element.Pos == position {
				limit = element.Max
			}
		}
		walkEntries(g.Structure, func(e *entry) {
			if e.Seg != segment {
				return
			}
			for _, override := range e.Elements {
				if override.Pos == position && override.Max != nil {
					limit = min(limit, *override.Max)
				}
			}
		})
		return limit
	}
	return 0
}

func walkEntries(entries []*entry, visit func(*entry)) {
	for _, e := range entries {
		visit(e)
		walkEntries(e.Body, visit)
	}
}
