package agentredteam

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SourceInboundEmail = "inbound_email"
	SourceDocument     = "document"
	SourceRecordNote   = "record_note"
	SourceBankMemo     = "bank_memo"
	SourceAttachment   = "attachment"
	SourceWebSearch    = "web_search"
	SourceMemory       = "memory"
	SourceSubject      = "subject"

	AutonomyAuto    = "auto"
	AutonomyDefault = "default"
)

var (
	ErrCaseInvalid = errors.New("red-team case is invalid")

	sourceKinds = []string{
		SourceInboundEmail,
		SourceDocument,
		SourceRecordNote,
		SourceBankMemo,
		SourceAttachment,
		SourceWebSearch,
		SourceMemory,
		SourceSubject,
	}
)

type Case struct {
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description"`
	Attempts    []string                  `yaml:"attempts"`
	Input       string                    `yaml:"input"`
	Source      Source                    `yaml:"source"`
	Agent       AgentSpec                 `yaml:"agent"`
	Delegates   []AgentSpec               `yaml:"delegates"`
	Responses   map[string]map[string]any `yaml:"responses"`
	Script      []Step                    `yaml:"script"`
	Expect      Expect                    `yaml:"expect"`
	KnownGap    *KnownGap                 `yaml:"knownGap"`
	Live        LiveSpec                  `yaml:"live"`
	Path        string                    `yaml:"-"`
}

type Source struct {
	Kind       string          `yaml:"kind"`
	Tool       string          `yaml:"tool"`
	Subject    *SubjectSpec    `yaml:"subject"`
	Attachment *AttachmentSpec `yaml:"attachment"`
	Memory     *MemorySpec     `yaml:"memory"`
}

type SubjectSpec struct {
	Type  string `yaml:"type"`
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
	Notes string `yaml:"notes"`
}

type AttachmentSpec struct {
	DocumentID string `yaml:"documentId"`
	FileName   string `yaml:"fileName"`
	Excerpt    string `yaml:"excerpt"`
}

type MemorySpec struct {
	ID      string `yaml:"id"`
	Kind    string `yaml:"kind"`
	Content string `yaml:"content"`
	Tainted bool   `yaml:"tainted"`
}

type AgentSpec struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Instructions string   `yaml:"instructions"`
	Tools        []string `yaml:"tools"`
	Autonomy     string   `yaml:"autonomy"`
	Unattended   bool     `yaml:"unattended"`
	Extensions   bool     `yaml:"extensions"`
}

type Step struct {
	Calls []Call `yaml:"calls"`
	Reply string `yaml:"reply"`
}

type Call struct {
	Name string         `yaml:"name"`
	Args map[string]any `yaml:"args"`
}

type Expect struct {
	Tainted  bool     `yaml:"tainted"`
	Decided  []string `yaml:"decided"`
	Refused  []string `yaml:"refused"`
	Delegate bool     `yaml:"delegate"`
	Memories int      `yaml:"memories"`
}

type KnownGap struct {
	ID         string   `yaml:"id"`
	Reason     string   `yaml:"reason"`
	Invariants []string `yaml:"invariants"`
}

type LiveSpec struct {
	Forbidden []string `yaml:"forbidden"`
}

func LoadCases(dir string) ([]*Case, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	cases := make([]*Case, 0, len(paths))
	names := make(map[string]string, len(paths))
	for _, path := range paths {
		loaded, loadErr := LoadCase(path)
		if loadErr != nil {
			return nil, loadErr
		}
		if other, taken := names[loaded.Name]; taken {
			return nil, fmt.Errorf("%w: %s and %s are both named %q",
				ErrCaseInvalid, other, path, loaded.Name)
		}
		names[loaded.Name] = path
		cases = append(cases, loaded)
	}

	return cases, nil
}

func LoadCase(path string) (*Case, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var loaded Case
	if err = yaml.Unmarshal(raw, &loaded); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	loaded.Path = path
	if err = loaded.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &loaded, nil
}

func (c *Case) validate() error {
	problems := make([]string, 0, 4)
	if strings.TrimSpace(c.Name) == "" {
		problems = append(problems, "name is required")
	}
	if strings.TrimSpace(c.Input) == "" {
		problems = append(problems, "input is required")
	}
	if !slices.Contains(sourceKinds, c.Source.Kind) {
		problems = append(problems, fmt.Sprintf("source.kind %q is not one of %s",
			c.Source.Kind, strings.Join(sourceKinds, ", ")))
	}
	problems = append(problems, c.Source.problems()...)
	if len(c.Script) == 0 {
		problems = append(problems, "script is required")
	}
	if len(c.Attempts) == 0 {
		problems = append(problems, "attempts names what the compromised model tries")
	}
	if c.KnownGap != nil && (c.KnownGap.ID == "" || c.KnownGap.Reason == "" ||
		len(c.KnownGap.Invariants) == 0) {
		problems = append(problems, "knownGap needs an id, a reason and its invariants")
	}
	if c.KnownGap != nil {
		for _, invariant := range c.KnownGap.Invariants {
			if !slices.Contains(Invariants(), invariant) {
				problems = append(problems, fmt.Sprintf("knownGap names unknown invariant %q",
					invariant))
			}
		}
	}
	for _, delegate := range c.Delegates {
		if len(delegate.ID) < minAgentIDLength {
			problems = append(problems, fmt.Sprintf("delegate %q needs an id of at least %d "+
				"characters", delegate.Name, minAgentIDLength))
		}
	}

	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrCaseInvalid, strings.Join(problems, "; "))
}

func (s Source) problems() []string {
	switch s.Kind {
	case SourceSubject:
		if s.Subject == nil {
			return []string{"a subject source needs source.subject"}
		}
	case SourceAttachment:
		if s.Attachment == nil {
			return []string{"an attachment source needs source.attachment"}
		}
	case SourceMemory:
		if s.Memory == nil {
			return []string{"a memory source needs source.memory"}
		}
	default:
		if s.Tool == "" {
			return []string{"a source read through a tool needs source.tool"}
		}
	}

	return nil
}

func (s Source) ReadAtOpen() bool {
	return s.Tool == ""
}
