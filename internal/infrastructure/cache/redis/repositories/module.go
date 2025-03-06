package repositories

import "go.uber.org/fx"

var Module = fx.Module("redis-repositories", fx.Provide(
	NewPermissionRepository,
	NewSessionRepository,
	NewStateRepository,
	NewOrganizationRepository,
))
