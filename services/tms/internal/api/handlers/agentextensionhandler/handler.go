package agentextensionhandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/agent/app/extensionwebwatchservice"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	WebWatches   *extensionwebwatchservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	webWatches *extensionwebwatchservice.Service
	eh         *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{webWatches: p.WebWatches, eh: p.ErrorHandler}
}
