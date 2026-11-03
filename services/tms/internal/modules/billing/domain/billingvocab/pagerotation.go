package billingvocab

const (
	MaxRotatedPage  = 2000
	MaxPageRotation = 270
)

type PageRotation struct {
	Page     int `json:"page"`
	Rotation int `json:"rotation"`
}

func QuarterTurn(degrees int) (int, bool) {
	if degrees%90 != 0 {
		return 0, false
	}
	return ((degrees % 360) + 360) % 360, true
}
