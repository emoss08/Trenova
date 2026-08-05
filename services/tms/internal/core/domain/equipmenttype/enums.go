package equipmenttype

type Class string

const (
	ClassTractor   = Class("Tractor")
	ClassTrailer   = Class("Trailer")
	ClassContainer = Class("Container")
	ClassOther     = Class("Other")
)

func (c Class) String() string {
	return string(c)
}

func (c Class) IsValid() bool {
	switch c {
	case ClassTractor, ClassTrailer, ClassContainer, ClassOther:
		return true
	default:
		return false
	}
}

type DeckType string

const (
	DeckTypeFlatbed    = DeckType("Flatbed")
	DeckTypeStepDeck   = DeckType("StepDeck")
	DeckTypeDoubleDrop = DeckType("DoubleDrop")
	DeckTypeRGN        = DeckType("RGN")
	DeckTypeLowboy     = DeckType("Lowboy")
	DeckTypeConestoga  = DeckType("Conestoga")
)

func (d DeckType) String() string {
	return string(d)
}

func (d DeckType) IsValid() bool {
	switch d {
	case DeckTypeFlatbed,
		DeckTypeStepDeck,
		DeckTypeDoubleDrop,
		DeckTypeRGN,
		DeckTypeLowboy,
		DeckTypeConestoga:
		return true
	default:
		return false
	}
}

func (d DeckType) IsOpenDeck() bool {
	return d.IsValid()
}

func (d DeckType) HasWell() bool {
	switch d {
	case DeckTypeDoubleDrop, DeckTypeRGN, DeckTypeLowboy:
		return true
	case DeckTypeFlatbed, DeckTypeStepDeck, DeckTypeConestoga:
		return false
	default:
		return false
	}
}
