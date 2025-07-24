/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
