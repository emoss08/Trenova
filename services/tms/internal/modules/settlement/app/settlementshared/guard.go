package settlementshared

type Guard struct {
	Refusal error
}

func (g Guard) Refused() bool { return g.Refusal != nil }

func Refuse(err error) Guard { return Guard{Refusal: err} }
