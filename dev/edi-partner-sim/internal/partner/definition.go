package partner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

const (
	partnerFile   = "partner.yaml"
	generateDir   = "generate"
	ladingDir     = "lading"
	guideDir      = "guide"
	samplesFile   = "samples.yaml"
	yamlExtension = ".yaml"
)

type Definition struct {
	Dir       string
	Partner   Partner
	Templates map[string]*Template
}

func OpenAll(root string) ([]*Definition, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read the partners folder: %w", err)
	}
	var definitions []*Definition
	var all Problems
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		if _, statErr := os.Stat(filepath.Join(dir, partnerFile)); statErr != nil {
			continue
		}
		definition, openErr := Open(dir)
		var problems Problems
		switch {
		case errors.As(openErr, &problems):
			all = append(all, problems...)
		case openErr != nil:
			return nil, openErr
		default:
			definitions = append(definitions, definition)
		}
	}
	if len(all) > 0 {
		return nil, all
	}
	return definitions, nil
}

func Open(dir string) (*Definition, error) {
	definition := &Definition{Dir: dir, Templates: map[string]*Template{}}
	file := filepath.Join(dir, partnerFile)
	if err := decodeFile(file, &definition.Partner); err != nil {
		return nil, err
	}
	if err := definition.Partner.validate(file, filepath.Base(dir)); err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(dir, generateDir, "*"+yamlExtension))
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", filepath.Join(dir, generateDir), err)
	}
	slices.Sort(paths)
	var all Problems
	for _, path := range paths {
		if isReplyTemplate(&definition.Partner, path) {
			continue
		}
		template := &Template{}
		if err := decodeFile(path, template); err != nil {
			return nil, err
		}
		check := &problems{file: path}
		if want := strings.TrimSuffix(filepath.Base(path), yamlExtension); template.Set != want {
			check.add("set", "%q must match the file name %q", template.Set, want)
		}
		template.validate(check, &definition.Partner)
		definition.Templates[template.Set] = template
		if len(check.list) == 0 {
			definition.dryRun(check, template)
		}
		all = append(all, check.list...)
	}
	for _, set := range definition.Partner.Sends {
		if _, ok := definition.Templates[set]; !ok && slices.Contains(generatedSets, set) {
			all = append(all, Problem{
				File:    filepath.Join(dir, generateDir, set+yamlExtension),
				Message: "is missing: partner.yaml says the partner sends " + set,
			})
		}
	}
	if len(all) > 0 {
		return nil, all
	}
	return definition, nil
}

func (d *Definition) dryRun(check *problems, template *Template) {
	for kind := range template.Purposes {
		for _, sparse := range []bool{false, true} {
			variation := Variation{Kind: kind, Sparse: sparse}
			if _, _, err := d.Transaction(template.Set, 1, variation); err != nil {
				check.add("segments", "%s: %v", variation, err)
				return
			}
		}
	}
}

func (d *Definition) Name() string {
	return d.Partner.Name
}

func (d *Definition) Load(set string, seed uint64) (*Load, error) {
	template, ok := d.Templates[set]
	if !ok {
		return nil, fmt.Errorf("%s does not send %s", d.Partner.Name, set)
	}
	return template.Load.Generate(d.Partner.Name, d.Partner.Location(), seed), nil
}

func (d *Definition) Transaction(
	set string,
	seed uint64,
	variation Variation,
) (Transaction, *Load, error) {
	template, ok := d.Templates[set]
	if !ok {
		return Transaction{}, nil, fmt.Errorf("%s does not send %s", d.Partner.Name, set)
	}
	purpose, ok := template.Purposes[variation.Kind]
	if !ok {
		return Transaction{}, nil, fmt.Errorf(
			"%s does not send a %s %s",
			d.Partner.Name,
			variation.Kind,
			set,
		)
	}
	profile, err := template.Load.Narrow(variation.Need)
	if err != nil {
		return Transaction{}, nil, err
	}
	base := profile.Generate(d.Partner.Name, d.Partner.Location(), seed)
	load, err := profile.apply(base, variation.Delta)
	if err != nil {
		return Transaction{}, nil, err
	}
	view := tenderView(&d.Partner, load, variation.Kind, purpose)
	segments, err := template.render(view, variation.Kind, variation.Sparse)
	if err != nil {
		return Transaction{}, nil, err
	}
	return Transaction{Set: set, Segments: segments}, load, nil
}

func (d *Definition) Interchange(
	set string,
	seed uint64,
	variation Variation,
	controls Controls,
) (Written, *Load, error) {
	written, loads, err := d.Batch(set, []Send{{Seed: seed, Variation: variation}}, controls)
	if err != nil {
		return Written{}, nil, err
	}
	return written[0], loads[0], nil
}

type Send struct {
	Seed      uint64
	Variation Variation
}

func (d *Definition) Batch(
	set string,
	sends []Send,
	controls Controls,
) ([]Written, []*Load, error) {
	if len(sends) == 0 {
		return nil, nil, fmt.Errorf("a batch of %s needs at least one send", set)
	}
	transactions := make([]Transaction, 0, len(sends))
	loads := make([]*Load, 0, len(sends))
	var at time.Time
	for _, send := range sends {
		variation := d.written(set, send.Variation)
		transaction, load, err := d.Transaction(set, send.Seed, variation)
		if err != nil {
			return nil, nil, err
		}
		transactions = append(transactions, transaction)
		loads = append(loads, load)
		if sent := variation.SentAt(load); sent.After(at) {
			at = sent
		}
	}
	written, err := d.Partner.Writer(controls).Write(at, transactions)
	if err != nil {
		return nil, nil, err
	}
	return written, loads, nil
}

func (d *Definition) Resends(set string) bool {
	template, ok := d.Templates[set]
	return ok && template.Duplicates == DuplicatesResend
}

func (d *Definition) written(set string, variation Variation) Variation {
	if variation.Kind == KindDuplicate && d.Resends(set) {
		variation.Kind = KindOriginal
	}
	return variation
}

var sendDelay = map[Kind]time.Duration{
	KindOriginal:  0,
	KindDuplicate: 30 * time.Minute,
	KindChange:    3 * time.Hour,
	KindReplace:   4 * time.Hour,
	KindCancel:    6 * time.Hour,
}

func (v Variation) SentAt(load *Load) time.Time {
	return load.TenderedAt.Add(sendDelay[v.Kind])
}

func decodeFile(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.UnmarshalWithOptions(raw, out, yaml.DisallowUnknownField()); err != nil {
		return Problems{{File: path, Message: yaml.FormatError(err, false, true)}}
	}
	return nil
}

func (d *Definition) GuidePaths() ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(d.Dir, guideDir, "*"+yamlExtension))
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", filepath.Join(d.Dir, guideDir), err)
	}
	slices.Sort(paths)
	return paths, nil
}
