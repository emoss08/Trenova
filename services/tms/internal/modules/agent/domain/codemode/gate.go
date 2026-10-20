package codemode

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"gopkg.in/yaml.v3"
)

//go:embed models.yaml
var modelsYAML []byte

type Override string

const (
	OverrideDefault = Override("")
	OverrideOn      = Override("on")
	OverrideOff     = Override("off")
)

func (o Override) IsValid() bool {
	switch o {
	case OverrideDefault, OverrideOn, OverrideOff:
		return true
	default:
		return false
	}
}

type EnabledModel struct {
	Provider aiprovider.Kind `yaml:"provider"`
	Model    string          `yaml:"model"`
	Run      string          `yaml:"run"`
	Decided  string          `yaml:"decided"`
	// Writes lets the model's scripts prepare changes, which are proposed as
	// one batch and never run before a person approves it. It needs its own
	// bench run on the batch-write pairs, WritesRun.
	Writes    bool   `yaml:"writes"`
	WritesRun string `yaml:"writesRun"`
}

type Ledger struct {
	Models []EnabledModel `yaml:"models"`
}

var (
	ErrUnearnedModel = errors.New(
		"codemode: a model is enabled without the bench run that earned it",
	)
	ErrUnearnedWrites = errors.New(
		"codemode: a model's scripts may write without the batch-write run that earned it",
	)
)

func ParseLedger(data []byte) (Ledger, error) {
	var ledger Ledger
	if err := yaml.Unmarshal(data, &ledger); err != nil {
		return Ledger{}, fmt.Errorf("codemode: read the model ledger: %w", err)
	}
	for idx := range ledger.Models {
		entry := ledger.Models[idx]
		if strings.TrimSpace(entry.Model) == "" || !entry.Provider.IsValid() {
			return Ledger{}, fmt.Errorf("codemode: ledger entry %d names no model", idx)
		}
		if strings.TrimSpace(entry.Run) == "" || strings.TrimSpace(entry.Decided) == "" {
			return Ledger{}, fmt.Errorf("%w: %s", ErrUnearnedModel, entry.Model)
		}
		if entry.Writes && strings.TrimSpace(entry.WritesRun) == "" {
			return Ledger{}, fmt.Errorf("%w: %s", ErrUnearnedWrites, entry.Model)
		}
	}

	return ledger, nil
}

var shipped = sync.OnceValues(func() (Ledger, error) { return ParseLedger(modelsYAML) })

func Shipped() Ledger {
	ledger, err := shipped()
	if err != nil {
		return Ledger{}
	}

	return ledger
}

func ShippedError() error {
	_, err := shipped()
	return err
}

func (l Ledger) Enabled(provider aiprovider.Kind, model string) bool {
	_, found := l.entry(provider, model)

	return found
}

// Writes reports a model whose scripts may prepare changes.
func (l Ledger) Writes(provider aiprovider.Kind, model string) bool {
	entry, found := l.entry(provider, model)

	return found && entry.Writes
}

func (l Ledger) entry(provider aiprovider.Kind, model string) (EnabledModel, bool) {
	model = strings.TrimSpace(model)
	if model == "" {
		return EnabledModel{}, false
	}
	for idx := range l.Models {
		if l.Models[idx].Provider == provider && strings.EqualFold(l.Models[idx].Model, model) {
			return l.Models[idx], true
		}
	}

	return EnabledModel{}, false
}

func (l Ledger) Any() bool {
	return len(l.Models) > 0
}

type Gate struct {
	Ledger Ledger
	Forced bool
}

type Decision struct {
	Allowed bool
	// Writes says the script may prepare changes as well as read.
	Writes bool
	Reason string
}

func (g Gate) Offered(override Override) bool {
	switch override {
	case OverrideOff:
		return false
	case OverrideOn:
		return g.Forced
	case OverrideDefault:
		return g.Ledger.Any()
	default:
		return false
	}
}

func (g Gate) Decide(override Override, provider aiprovider.Kind, model string) Decision {
	switch {
	case override == OverrideOff:
		return Decision{Reason: "code mode is turned off for this turn"}
	case override == OverrideOn && g.Forced:
		return Decision{Allowed: true, Writes: true}
	case override == OverrideOn:
		return Decision{Reason: "code mode can be forced on only in development and test"}
	case g.Ledger.Enabled(provider, model):
		return Decision{Allowed: true, Writes: g.Ledger.Writes(provider, model)}
	default:
		name := strings.TrimSpace(model)
		if name == "" {
			name = "this model"
		}
		return Decision{Reason: "code mode is not enabled for " + name}
	}
}
