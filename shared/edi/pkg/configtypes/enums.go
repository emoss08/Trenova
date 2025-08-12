package configtypes

type Strictness string

const (
	Strict  Strictness = "strict"
	Lenient Strictness = "lenient"
)

func (s Strictness) String() string {
	return string(s)
}
