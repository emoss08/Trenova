package bootstrap

import (
	"github.com/emoss08/trenova/internal/bootstrap/apiapp"
	"github.com/emoss08/trenova/internal/bootstrap/core"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/bootstrap/workerapp"
	"go.uber.org/fx"
)

type App struct {
	*fx.App
}

func Options() fx.Option {
	return OptionsFor(edition.Current())
}

func OptionsFor(e *edition.Edition) fx.Option {
	return fx.Options(
		core.OptionsFor(e),
		workerapp.Modules(),
	)
}

func NewApp(opts ...fx.Option) *App {
	baseOpts := Options()
	allOpts := append([]fx.Option{baseOpts}, opts...)

	return &App{
		App: fx.New(allOpts...),
	}
}

func APIOptions() fx.Option {
	return APIOptionsFor(edition.Current())
}

func APIOptionsFor(e *edition.Edition) fx.Option {
	return apiapp.OptionsFor(e)
}

func WorkerOptions() fx.Option {
	return WorkerOptionsFor(edition.Current())
}

func WorkerOptionsFor(e *edition.Edition) fx.Option {
	return workerapp.OptionsFor(e)
}

func UnscheduledWorkerOptions() fx.Option {
	return workerapp.UnscheduledOptions()
}
