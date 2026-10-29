package captureprofiles

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	ProfileRepo captureports.CaptureProfileRepository
	Support     *capturesupport.Service
}

type Service struct {
	profileRepo captureports.CaptureProfileRepository
	support     *capturesupport.Service
}

func New(p Params) *Service {
	return &Service{
		profileRepo: p.ProfileRepo,
		support:     p.Support,
	}
}
