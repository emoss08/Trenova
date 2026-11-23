package board

import (
	"slices"
)

const MaxConversationBlocks = 4

type Place string

const (
	PlaceConversation = Place("conversation")
	PlaceBeside       = Place("beside")
)

func AllPlaces() []Place {
	return []Place{PlaceConversation, PlaceBeside}
}

func (p Place) IsValid() bool {
	return slices.Contains(AllPlaces(), p)
}

func PlaceFor(asked Place, blocks int) Place {
	if asked == PlaceConversation && blocks <= MaxConversationBlocks {
		return PlaceConversation
	}

	return PlaceBeside
}

func (b *Board) Placed() Place {
	if b.Place == PlaceConversation {
		return PlaceConversation
	}

	return PlaceBeside
}
