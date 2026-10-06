package bootstrap_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/editioninfo"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

type everyProcess struct{}

type apiOnly struct{}

type workerOnly struct{}

type stubSection struct{}

func (stubSection) Name() string                          { return "stub" }
func (stubSection) Paths() []string                       { return []string{"platform.stub"} }
func (stubSection) PlatformModes() []config.PlatformMode  { return nil }
func (stubSection) SetDefaults(*viper.Viper, string)      {}
func (stubSection) Decode(*viper.Viper) (any, error)      { return struct{}{}, nil }
func (stubSection) Validate(*config.Config, string) error { return nil }

type sections struct {
	fx.In

	Sections []config.Section `group:"config_sections"`
}

func fakeEdition() *edition.Edition {
	return &edition.Edition{
		Name: "fake",
		Options: []fx.Option{
			fx.Supply(&everyProcess{}),
			fx.Decorate(func(services.EditionInfo) services.EditionInfo {
				return editioninfo.NewStatic("fake", true)
			}),
		},
		APIOptions:     []fx.Option{fx.Supply(&apiOnly{})},
		WorkerOptions:  []fx.Option{fx.Supply(&workerOnly{})},
		ConfigSections: []config.Section{stubSection{}},
	}
}

func TestEditionOptionsReachTheAPIGraph(t *testing.T) {
	t.Parallel()

	fake := fakeEdition()
	require.NoError(t, fx.ValidateApp(
		bootstrap.OptionsFor(fake),
		bootstrap.APIOptionsFor(fake),
		fx.Invoke(func(*everyProcess, *apiOnly, services.EditionInfo, sections) {}),
	))

	err := fx.ValidateApp(
		bootstrap.OptionsFor(fake),
		bootstrap.APIOptionsFor(fake),
		fx.Invoke(func(*workerOnly) {}),
	)
	require.Error(t, err, "worker options must stay out of the API graph")
}

func TestEditionOptionsReachTheWorkerGraph(t *testing.T) {
	t.Parallel()

	fake := fakeEdition()
	require.NoError(t, fx.ValidateApp(
		bootstrap.OptionsFor(fake),
		bootstrap.WorkerOptionsFor(fake),
		fx.Invoke(func(*everyProcess, *workerOnly) {}),
	))

	err := fx.ValidateApp(
		bootstrap.OptionsFor(fake),
		bootstrap.WorkerOptionsFor(fake),
		fx.Invoke(func(*apiOnly) {}),
	)
	require.Error(t, err, "API options must stay out of the worker graph")
}

func TestCommunityEditionAddsNothing(t *testing.T) {
	t.Parallel()

	community := edition.Community()
	require.NoError(t, fx.ValidateApp(
		bootstrap.OptionsFor(community),
		bootstrap.APIOptionsFor(community),
		fx.Invoke(func(info services.EditionInfo) {}),
	))

	err := fx.ValidateApp(
		bootstrap.OptionsFor(community),
		bootstrap.APIOptionsFor(community),
		fx.Invoke(func(*everyProcess) {}),
	)
	require.Error(t, err)
}
