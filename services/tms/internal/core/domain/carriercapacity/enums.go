package carriercapacity

type RateMethod string

const (
	RateMethodFlat    = RateMethod("Flat")
	RateMethodPerMile = RateMethod("PerMile")
)

func RateMethods() []RateMethod {
	return []RateMethod{RateMethodFlat, RateMethodPerMile}
}

func (m RateMethod) String() string { return string(m) }

func (m RateMethod) IsValid() bool {
	switch m {
	case RateMethodFlat, RateMethodPerMile:
		return true
	default:
		return false
	}
}

type Source string

const (
	SourceManual = Source("Manual")
	SourceEmail  = Source("Email")
	SourceEDI    = Source("EDI")
)

func Sources() []Source {
	return []Source{SourceManual, SourceEmail, SourceEDI}
}

func (s Source) String() string { return string(s) }

func (s Source) IsValid() bool {
	switch s {
	case SourceManual, SourceEmail, SourceEDI:
		return true
	default:
		return false
	}
}
