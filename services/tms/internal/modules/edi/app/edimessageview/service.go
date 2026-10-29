package edimessageview

import (
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	MessageRepo edimessageports.EDIMessageRepository
}

type Service struct {
	messageRepo edimessageports.EDIMessageRepository
}

func New(p Params) *Service {
	return &Service{
		messageRepo: p.MessageRepo,
	}
}
