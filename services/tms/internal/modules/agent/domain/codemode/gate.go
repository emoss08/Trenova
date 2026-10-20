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
}

type Ledger struct {
	Models []EnabledModel `yaml:"models"`
}

var ErrUnearnedModel = errors.New(
	"codemode: a model is enabled without the bench run that earned it",
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
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	for idx := range l.Models {
		if l.Models[idx].Provider == provider && strings.EqualFold(l.Models[idx].Model, model) {
			return true
		}
	}

	return false
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
	Reason  string
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
		return Decision{Allowed: true}
	case override == OverrideOn:
		return Decision{Reason: "code mode can be forced on only in development and test"}
	case g.Ledger.Enabled(provider, model):
		return Decision{Allowed: true}
	default:
		name := strings.TrimSpace(model)
		if name == "" {
			name = "this model"
		}
		return Decision{Reason: "code mode is not enabled for " + name}
	}
}
