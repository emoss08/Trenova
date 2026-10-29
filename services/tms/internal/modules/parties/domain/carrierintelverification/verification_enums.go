package carrierintelverification

type VerificationResult string

const (
	VerificationResultMatch         = VerificationResult("Match")
	VerificationResultMismatch      = VerificationResult("Mismatch")
	VerificationResultNotFound      = VerificationResult("NotFound")
	VerificationResultUnverifiable  = VerificationResult("Unverifiable")
	VerificationResultProviderError = VerificationResult("ProviderError")
)

func (r VerificationResult) String() string { return string(r) }

func (r VerificationResult) IsValid() bool {
	switch r {
	case VerificationResultMatch, VerificationResultMismatch, VerificationResultNotFound,
		VerificationResultUnverifiable, VerificationResultProviderError:
		return true
	default:
		return false
	}
}
