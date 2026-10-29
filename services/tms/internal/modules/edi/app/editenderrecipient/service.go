package editenderrecipient

import (
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	ProfileRepo         edipartnerports.EDICommunicationProfileRepository
	TenderChangeRepo    editenderports.EDITenderChangeRepository
	TenderRecipientRepo editenderports.EDITenderRecipientRepository
}

type Service struct {
	profileRepo         edipartnerports.EDICommunicationProfileRepository
	tenderChangeRepo    editenderports.EDITenderChangeRepository
	tenderRecipientRepo editenderports.EDITenderRecipientRepository
}

func New(p Params) *Service {
	return &Service{
		profileRepo:         p.ProfileRepo,
		tenderChangeRepo:    p.TenderChangeRepo,
		tenderRecipientRepo: p.TenderRecipientRepo,
	}
}
