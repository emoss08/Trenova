package tableconfiguration

type Visibility string

const (
	VisibilityPrivate = Visibility("Private")
	VisibilityPublic  = Visibility("Public")
	VisibilityShared  = Visibility("Shared")
)

type ShareType string

const (
	ShareTypeUser = ShareType("User")
	ShareTypeRole = ShareType("Role")
	ShareTypeTeam = ShareType("Team")
)
