package tableconfiguration

import "errors"

type Visibility string

const (
	VisibilityPrivate = Visibility("Private")
	VisibilityPublic  = Visibility("Public")
	VisibilityShared  = Visibility("Shared")
)

func (v Visibility) String() string {
	return string(v)
}

func VisibilityFromString(s string) (Visibility, error) {
	switch s {
	case "Private":
		return VisibilityPrivate, nil
	case "Public":
		return VisibilityPublic, nil
	case "Shared":
		return VisibilityShared, nil
	default:
		return "", errors.New("invalid visibility")
	}
}
