package edition

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const CommunityName = "community"

type Edition struct {
	Name               string
	Options            []fx.Option
	APIOptions         []fx.Option
	WorkerOptions      []fx.Option
	Commands           []*cobra.Command
	PostgresMigrations []fs.FS
	ConfigSections     []config.Section
}

var (
	mu         sync.RWMutex
	registered *Edition
)

func Community() *Edition {
	return &Edition{Name: CommunityName}
}

//nolint:gocritic // registered once at init; the registry keeps its own copy
func Register(e Edition) {
	if err := validate(&e); err != nil {
		panic(err)
	}

	mu.Lock()
	defer mu.Unlock()

	if registered != nil {
		panic(fmt.Errorf(
			"edition: %q cannot be registered because %q already is",
			e.Name,
			registered.Name,
		))
	}

	registered = e.clone()
}

func Current() *Edition {
	mu.RLock()
	defer mu.RUnlock()

	if registered == nil {
		return Community()
	}

	return registered.clone()
}

func IsRegistered() bool {
	mu.RLock()
	defer mu.RUnlock()

	return registered != nil
}

func (e *Edition) IsCommunity() bool {
	return e.Name == CommunityName
}

func (e *Edition) Option() fx.Option {
	return fx.Options(e.Options...)
}

func (e *Edition) APIOption() fx.Option {
	return fx.Options(e.APIOptions...)
}

func (e *Edition) WorkerOption() fx.Option {
	return fx.Options(e.WorkerOptions...)
}

func (e *Edition) LoaderOptions() []config.LoaderOption {
	return []config.LoaderOption{config.WithSections(e.ConfigSections...)}
}

func (e *Edition) clone() *Edition {
	return &Edition{
		Name:               e.Name,
		Options:            slices.Clone(e.Options),
		APIOptions:         slices.Clone(e.APIOptions),
		WorkerOptions:      slices.Clone(e.WorkerOptions),
		Commands:           slices.Clone(e.Commands),
		PostgresMigrations: slices.Clone(e.PostgresMigrations),
		ConfigSections:     slices.Clone(e.ConfigSections),
	}
}

func validate(e *Edition) error {
	name := strings.TrimSpace(e.Name)
	if name == "" {
		return errors.New("edition: a registered edition must be named")
	}
	if strings.EqualFold(name, CommunityName) {
		return fmt.Errorf("edition: %q is the built-in edition and cannot be registered", name)
	}

	for idx, option := range slices.Concat(e.Options, e.APIOptions, e.WorkerOptions) {
		if option == nil {
			return fmt.Errorf("edition: %q has a nil fx option at %d", name, idx)
		}
	}

	seen := make(map[string]struct{}, len(e.Commands))
	for _, command := range e.Commands {
		if command == nil {
			return fmt.Errorf("edition: %q registers a nil command", name)
		}

		use := command.Name()
		if _, duplicate := seen[use]; duplicate {
			return fmt.Errorf("edition: %q registers the command %q twice", name, use)
		}
		seen[use] = struct{}{}
	}

	for idx, fsys := range e.PostgresMigrations {
		if fsys == nil {
			return fmt.Errorf("edition: %q has a nil migration set at %d", name, idx)
		}
	}

	for idx, section := range e.ConfigSections {
		if section == nil {
			return fmt.Errorf("edition: %q has a nil configuration section at %d", name, idx)
		}
	}

	return nil
}
