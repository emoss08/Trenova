package triageconflictservice

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/shared/addressutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type draftedLoad struct {
	origin      string
	destination string
	pickup      string
}

func (l *draftedLoad) empty() bool {
	return l.origin == "" && l.destination == "" && l.pickup == ""
}

func loadsInDraft(draft *inboundmessage.LoadDraft) []draftedLoad {
	if draft == nil {
		return nil
	}

	loads := make([]draftedLoad, 0, 1)
	current := draftedLoad{}
	for _, field := range draft.Fields {
		switch field.Key {
		case inboundmessage.LoadFieldOrigin:
			if current.origin != "" {
				loads = append(loads, current)
				current = draftedLoad{}
			}
			current.origin = field.Value
		case inboundmessage.LoadFieldDestination:
			current.destination = field.Value
		case inboundmessage.LoadFieldPickup:
			current.pickup = field.Value
		case inboundmessage.LoadFieldCustomer, inboundmessage.LoadFieldDelivery,
			inboundmessage.LoadFieldEquipment, inboundmessage.LoadFieldWeight,
			inboundmessage.LoadFieldRate, inboundmessage.LoadFieldLoads,
			inboundmessage.LoadFieldWeekOf, inboundmessage.LoadFieldRespondBy,
			inboundmessage.LoadFieldReference, inboundmessage.LoadFieldCommodity,
			inboundmessage.LoadFieldInstructions:
		}
	}
	if !current.empty() {
		loads = append(loads, current)
	}

	return loads
}

func tenderLoads(
	draft *inboundmessage.LoadDraft,
	loc *time.Location,
	now int64,
) []triageports.TenderLoad {
	drafted := loadsInDraft(draft)
	loads := make([]triageports.TenderLoad, 0, len(drafted))
	for i, load := range drafted {
		origin, okOrigin := addressutils.ParseCityState(load.origin)
		destination, okDestination := addressutils.ParseCityState(load.destination)
		day, okDay := timeutils.FindDocumentDate(load.pickup)
		if !okOrigin || !okDestination || !okDay {
			continue
		}

		start := day.On(time.Unix(now, 0).In(loc))
		loads = append(loads, triageports.TenderLoad{
			Index:            i + 1,
			OriginCity:       origin.City,
			OriginState:      origin.State,
			DestinationCity:  destination.City,
			DestinationState: destination.State,
			PickupFrom:       start.Unix(),
			PickupTo:         timeutils.NextDayStart(start.Unix(), loc),
		})
	}

	return loads
}
