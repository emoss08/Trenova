package seeder

import (
	"errors"
	"fmt"
	"strings"
)

type SeedPhase string

const (
	PhaseValidation = SeedPhase("validation")
	PhaseDependency = SeedPhase("dependency")
	PhaseExecution  = SeedPhase("execution")
	PhaseCommit     = SeedPhase("commit")
)

type SeedError struct {
	SeedName string
	Phase    SeedPhase
	Cause    error
}

func (e *SeedError) Error() string {
	return fmt.Sprintf("seed %q failed during %s: %v", e.SeedName, e.Phase, e.Cause)
}

func (e *SeedError) Unwrap() error {
	return e.Cause
}

func NewSeedError(seedName string, phase SeedPhase, cause error) *SeedError {
	return &SeedError{
		SeedName: seedName,
		Phase:    phase,
		Cause:    cause,
	}
}

type DependencyError struct {
	SeedName     string
	MissingDeps  []string
	CircularPath []string
}

func (e *DependencyError) Error() string {
	if len(e.CircularPath) > 0 {
		return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.CircularPath, " -> "))
	}
	if len(e.MissingDeps) > 0 {
		return fmt.Sprintf(
			"seed %q has missing dependencies: %s",
			e.SeedName,
			strings.Join(e.MissingDeps, ", "),
		)
	}
	return fmt.Sprintf("dependency error for seed %q", e.SeedName)
}

func NewMissingDependencyError(seedName string, missingDeps []string) *DependencyError {
	return &DependencyError{
		SeedName:    seedName,
		MissingDeps: missingDeps,
	}
}

func NewCircularDependencyError(path []string) *DependencyError {
	return &DependencyError{
		CircularPath: path,
	}
}

type ValidationError struct {
	SeedName string
	Issues   []string
}

func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return fmt.Sprintf("validation error for seed %q", e.SeedName)
	}
	return fmt.Sprintf("validation error for seed %q: %s", e.SeedName, strings.Join(e.Issues, "; "))
}

func NewValidationError(seedName string, issues ...string) *ValidationError {
	return &ValidationError{
		SeedName: seedName,
		Issues:   issues,
	}
}

type RegistryError struct {
	Message string
	Cause   error
}

func (e *RegistryError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("registry error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("registry error: %s", e.Message)
}

func (e *RegistryError) Unwrap() error {
	return e.Cause
}

func NewRegistryError(message string, cause error) *RegistryError {
	return &RegistryError{
		Message: message,
		Cause:   cause,
	}
}

var (
	ErrSeedAlreadyRegistered = errors.New("seed already registered")
	ErrSeedNotFound          = errors.New("seed not found")
	ErrNoSeedsToApply        = errors.New("no seeds to apply")
	ErrUserCancelled         = errors.New("operation cancelled by user")
)
