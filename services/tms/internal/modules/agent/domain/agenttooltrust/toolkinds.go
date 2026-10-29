package agenttooltrust

type ToolKind string

const (
	ToolKindQuery   = ToolKind("query")
	ToolKindAction  = ToolKind("action")
	ToolKindRuntime = ToolKind("runtime")
)

func (k ToolKind) IsValid() bool {
	switch k {
	case ToolKindQuery, ToolKindAction, ToolKindRuntime:
		return true
	default:
		return false
	}
}

func (k ToolKind) String() string { return string(k) }

type ToolScope string

const (
	ToolScopeTenant = ToolScope("tenant")
	ToolScopeSelf   = ToolScope("self")
	ToolScopeRun    = ToolScope("run")
)

func (s ToolScope) IsValid() bool {
	switch s {
	case ToolScopeTenant, ToolScopeSelf, ToolScopeRun:
		return true
	default:
		return false
	}
}

func (s ToolScope) String() string { return string(s) }

type ExternalRead string

const (
	ExternalReadNever  = ExternalRead("never")
	ExternalReadAlways = ExternalRead("always")
	ExternalReadMarked = ExternalRead("marked")
)

func (r ExternalRead) IsValid() bool {
	switch r {
	case ExternalReadNever, ExternalReadAlways, ExternalReadMarked:
		return true
	default:
		return false
	}
}

func (r ExternalRead) String() string { return string(r) }
