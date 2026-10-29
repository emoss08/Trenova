package tenant

type EnforcementLevel string

func (v EnforcementLevel) IsValid() bool {
	switch v {
	case EnforcementLevelIgnore,
		EnforcementLevelWarn,
		EnforcementLevelRequireReview,
		EnforcementLevelBlock:
		return true
	default:
		return false
	}
}

const (
	EnforcementLevelIgnore        = EnforcementLevel("Ignore")
	EnforcementLevelWarn          = EnforcementLevel("Warn")
	EnforcementLevelRequireReview = EnforcementLevel("RequireReview")
	EnforcementLevelBlock         = EnforcementLevel("Block")
)
