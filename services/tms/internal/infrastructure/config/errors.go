package config

import "errors"

var (
	ErrPlatformModeRequiresEdition = errors.New(
		"platform.mode is provided by an edition that is not part of this build",
	)
	ErrSectionRequiresEdition = errors.New(
		"configuration section belongs to an edition that is not part of this build",
	)
	ErrSectionNameRequired      = errors.New("configuration section name is required")
	ErrSectionPathsRequired     = errors.New("configuration section must own at least one path")
	ErrSectionAlreadyRegistered = errors.New("configuration section is already registered")
	ErrSectionPathConflict      = errors.New(
		"configuration section path is already owned by another section or by the base configuration",
	)
)
