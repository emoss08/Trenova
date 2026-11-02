package bootstrap

import (
	"github.com/emoss08/trenova/internal/bootstrap/apiapp"
	"github.com/emoss08/trenova/internal/bootstrap/core"
	"github.com/emoss08/trenova/internal/bootstrap/workerapp"
	"go.uber.org/fx"
)

type App struct {
	*fx.App
}

func Options() fx.Option {
	return fx.Options(
		core.Options(),
		workerapp.Modules(),
	)
}

func NewApp(opts ...fx.Option) *App {
	return &App{
		App: fx.New(append([]fx.Option{Options()}, opts...)...),
	}
}

func APIOptions() fx.Option {
	return apiapp.Options()
}

func WorkerOptions() fx.Option {
	return workerapp.Options()
}

func UnscheduledWorkerOptions() fx.Option {
	return workerapp.UnscheduledOptions()
}
