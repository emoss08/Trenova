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
