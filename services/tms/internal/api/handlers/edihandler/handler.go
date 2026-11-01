package edihandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundas2"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	AS2Service   *ediinboundas2.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	as2Service *ediinboundas2.Service
	eh         *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		as2Service: p.AS2Service,
		eh:         p.ErrorHandler,
	}
}
