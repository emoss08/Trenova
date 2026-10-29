package agentreach

type ReachWarningKind string

const (
	ReachOpenWithSensitiveTools = ReachWarningKind("OpenWithSensitiveTools")
	ReachNoAudience             = ReachWarningKind("NoAudience")
)

func (k ReachWarningKind) IsValid() bool {
	switch k {
	case ReachOpenWithSensitiveTools, ReachNoAudience:
		return true
	default:
		return false
	}
}

func (k ReachWarningKind) String() string { return string(k) }
