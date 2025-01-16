package equipmenttype

type Class string

const (
	ClassTractor   = Class("Tractor")
	ClassTrailer   = Class("Trailer")
	ClassContainer = Class("Container")
	ClassOther     = Class("Other")
)
