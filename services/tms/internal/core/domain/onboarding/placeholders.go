package onboarding

const (
	PlaceholderSCAC       = "TBDX"
	PlaceholderDOTNumber  = "0"
	PlaceholderCity       = "Pending"
	PlaceholderPostalCode = "00000"
)

func IsPlaceholderSCAC(value string) bool {
	return value == PlaceholderSCAC
}

func IsPlaceholderDOTNumber(value string) bool {
	return value == PlaceholderDOTNumber
}
