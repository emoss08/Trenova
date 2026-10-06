package edition

import (
	iofs "io/fs"
	"testing"
	"testing/fstest"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

type marker struct{ from string }

type stubSection struct{}

func (stubSection) Name() string                         { return "stub" }
func (stubSection) Paths() []string                      { return []string{"platform.stub"} }
func (stubSection) PlatformModes() []config.PlatformMode { return nil }
func (stubSection) SetDefaults(*viper.Viper, string)     {}
func (stubSection) Decode(*viper.Viper) (any, error)     { return struct{}{}, nil }
func (stubSection) Validate(*config.Config, string) error {
	return nil
}

func resetRegistry(t *testing.T) {
	t.Helper()

	mu.Lock()
	registered = nil
	mu.Unlock()

	t.Cleanup(func() {
		mu.Lock()
		registered = nil
		mu.Unlock()
	})
}

func TestCurrentWithoutARegistrationIsTheCommunityEdition(t *testing.T) {
	resetRegistry(t)

	current := Current()
	assert.Equal(t, CommunityName, current.Name)
	assert.True(t, current.IsCommunity())
	assert.False(t, IsRegistered())
	assert.Empty(t, current.Options)
	assert.Empty(t, current.APIOptions)
	assert.Empty(t, current.WorkerOptions)
	assert.Empty(t, current.Commands)
	assert.Empty(t, current.PostgresMigrations)
	assert.Empty(t, current.ConfigSections)

	require.NoError(t, fx.ValidateApp(current.Option(), current.APIOption(), current.WorkerOption()))
}

func TestRegisterMakesTheEditionCurrent(t *testing.T) {
	resetRegistry(t)

	command := &cobra.Command{Use: "operator"}
	migrations := fstest.MapFS{"20990101000000_example.up.sql": {Data: []byte("SELECT 1;")}}
	Register(Edition{
		Name:               "cloud",
		Options:            []fx.Option{fx.Supply(&marker{from: "all"})},
		APIOptions:         []fx.Option{fx.Supply("api")},
		WorkerOptions:      []fx.Option{fx.Supply(42)},
		Commands:           []*cobra.Command{command},
		PostgresMigrations: []iofs.FS{migrations},
		ConfigSections:     []config.Section{stubSection{}},
	})

	current := Current()
	require.True(t, IsRegistered())
	assert.Equal(t, "cloud", current.Name)
	assert.False(t, current.IsCommunity())
	assert.Equal(t, []*cobra.Command{command}, current.Commands)
	require.Len(t, current.PostgresMigrations, 1)
	require.Len(t, current.ConfigSections, 1)
	assert.Len(t, current.LoaderOptions(), 1)

	var (
		all    *marker
		api    string
		worker int
	)
	app := fx.New(
		fx.NopLogger,
		current.Option(),
		current.APIOption(),
		current.WorkerOption(),
		fx.Populate(&all, &api, &worker),
	)
	require.NoError(t, app.Err())
	assert.Equal(t, "all", all.from)
	assert.Equal(t, "api", api)
	assert.Equal(t, 42, worker)
}

func TestCurrentReturnsACopy(t *testing.T) {
	resetRegistry(t)

	Register(Edition{
		Name:     "cloud",
		Commands: []*cobra.Command{{Use: "operator"}},
	})

	first := Current()
	first.Commands[0] = &cobra.Command{Use: "replaced"}
	first.Commands = append(first.Commands, &cobra.Command{Use: "extra"})

	second := Current()
	require.Len(t, second.Commands, 1)
	assert.Equal(t, "operator", second.Commands[0].Use)
}

func TestRegisterTwicePanics(t *testing.T) {
	resetRegistry(t)

	Register(Edition{Name: "cloud"})
	assert.Panics(t, func() { Register(Edition{Name: "other"}) })
	assert.Equal(t, "cloud", Current().Name)
}

func TestRegisterRefusesAnInvalidEdition(t *testing.T) {
	tests := []struct {
		name    string
		edition Edition
	}{
		{name: "unnamed", edition: Edition{}},
		{name: "blank name", edition: Edition{Name: "  "}},
		{name: "the built-in name", edition: Edition{Name: "Community"}},
		{name: "nil option", edition: Edition{Name: "cloud", Options: []fx.Option{nil}}},
		{name: "nil api option", edition: Edition{Name: "cloud", APIOptions: []fx.Option{nil}}},
		{
			name:    "nil worker option",
			edition: Edition{Name: "cloud", WorkerOptions: []fx.Option{nil}},
		},
		{name: "nil command", edition: Edition{Name: "cloud", Commands: []*cobra.Command{nil}}},
		{
			name: "duplicate command",
			edition: Edition{Name: "cloud", Commands: []*cobra.Command{
				{Use: "operator"},
				{Use: "operator [flags]"},
			}},
		},
		{name: "nil migrations", edition: Edition{Name: "cloud", PostgresMigrations: []iofs.FS{nil}}},
		{
			name:    "nil section",
			edition: Edition{Name: "cloud", ConfigSections: []config.Section{nil}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRegistry(t)

			assert.Panics(t, func() { Register(tt.edition) })
			assert.False(t, IsRegistered())
		})
	}
}
