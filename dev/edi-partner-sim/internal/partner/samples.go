package partner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var sampleFilePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*\.x12$`)

type SampleSpec struct {
	File      string    `yaml:"file"`
	Set       string    `yaml:"set"`
	Seed      uint64    `yaml:"seed"`
	With      []uint64  `yaml:"with"`
	Control   int64     `yaml:"control"`
	Variation Variation `yaml:"variation"`
}

func (s SampleSpec) sends() []Send {
	sends := []Send{{Seed: s.Seed, Variation: s.Variation}}
	for _, seed := range s.With {
		sends = append(sends, Send{Seed: seed, Variation: s.Variation})
	}
	return sends
}

type GeneratedSample struct {
	Path    string
	Content []byte
}

type SourceFile struct {
	Path    string
	Content string
}

func (d *Definition) SampleSpecs() ([]SampleSpec, error) {
	path := filepath.Join(d.Dir, samplesFile)
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	var specs []SampleSpec
	if err := decodeFile(path, &specs); err != nil {
		return nil, err
	}
	check := &problems{file: path}
	seen := map[string]bool{}
	for index, spec := range specs {
		where := fmt.Sprintf("[%d]", index)
		if !sampleFilePattern.MatchString(spec.File) {
			check.add(
				where+".file",
				"%q is not a sample file name such as tender-original.x12",
				spec.File,
			)
		}
		if seen[spec.File] {
			check.add(where+".file", "%q is listed twice", spec.File)
		}
		seen[spec.File] = true
		if _, ok := d.Templates[spec.Set]; !ok {
			check.add(where+".set", "%q has no generate template", spec.Set)
		}
		if spec.Control < 1 || spec.Control > 999_999_999 {
			check.add(where+".control", "%d is not an interchange control number", spec.Control)
		}
		spec.Variation.validate(check, where+".variation")
	}
	if err := check.err(); err != nil {
		return nil, err
	}
	return specs, nil
}

func (d *Definition) GenerateSamples() ([]GeneratedSample, error) {
	specs, err := d.SampleSpecs()
	if err != nil {
		return nil, err
	}
	out := make([]GeneratedSample, 0, len(specs))
	for _, spec := range specs {
		controls := Controls{Interchange: NewCounter(spec.Control), Group: NewCounter(spec.Control)}
		where := filepath.Join(d.Dir, samplesFile)
		written, _, err := d.Batch(spec.Set, spec.sends(), controls)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", where, spec.File, err)
		}
		if len(written) != 1 {
			return nil, fmt.Errorf(
				"%s: %s: the batch spans %d interchanges; the partner's batching allows fewer",
				where,
				spec.File,
				len(written),
			)
		}
		out = append(out, GeneratedSample{
			Path:    filepath.Join(d.Dir, ladingDir, "samples", spec.File),
			Content: written[0].Bytes,
		})
	}
	return out, nil
}

func (d *Definition) WriteSamples() error {
	samples, err := d.GenerateSamples()
	if err != nil {
		return err
	}
	for _, sample := range samples {
		if err := os.MkdirAll(filepath.Dir(sample.Path), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", filepath.Dir(sample.Path), err)
		}
		if err := os.WriteFile(sample.Path, sample.Content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", sample.Path, err)
		}
	}
	return nil
}

func (d *Definition) LadingProject() (files, samples []SourceFile, err error) {
	root := filepath.Join(d.Dir, ladingDir)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		file := SourceFile{Path: filepath.ToSlash(relative), Content: string(raw)}
		if strings.HasSuffix(relative, ".lading") {
			files = append(files, file)
		} else {
			samples = append(samples, file)
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("read the lading project %s: %w", root, err)
	}
	slices.SortFunc(files, func(a, b SourceFile) int { return strings.Compare(a.Path, b.Path) })
	slices.SortFunc(samples, func(a, b SourceFile) int { return strings.Compare(a.Path, b.Path) })
	return files, samples, nil
}
