package editendermapping

import (
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	MappingProfileRepo edipartnerports.EDIMappingProfileRepository
}

type Service struct {
	mappingProfileRepo edipartnerports.EDIMappingProfileRepository
}

func New(p Params) *Service {
	return &Service{
		mappingProfileRepo: p.MappingProfileRepo,
	}
}
