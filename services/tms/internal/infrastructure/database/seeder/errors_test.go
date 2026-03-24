package seeder

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeedError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seedErr *SeedError
		wantStr string
	}{
		{
			name: "validation phase",
			seedErr: &SeedError{
				SeedName: "TestSeed",
				Phase:    PhaseValidation,
				Cause:    errors.New("invalid configuration"),
			},
			wantStr: `seed "TestSeed" failed during validation: invalid configuration`,
		},
		{
			name: "dependency phase",
			seedErr: &SeedError{
				SeedName: "UserSeed",
				Phase:    PhaseDependency,
				Cause:    errors.New("missing dependency"),
			},
			wantStr: `seed "UserSeed" failed during dependency: missing dependency`,
		},
		{
			name: "execution phase",
			seedErr: &SeedError{
				SeedName: "DataSeed",
				Phase:    PhaseExecution,
				Cause:    errors.New("database error"),
			},
			wantStr: `seed "DataSeed" failed during execution: database error`,
		},
		{
			name: "commit phase",
			seedErr: &SeedError{
				SeedName: "CommitSeed",
				Phase:    PhaseCommit,
				Cause:    errors.New("transaction failed"),
			},
			wantStr: `seed "CommitSeed" failed during commit: transaction failed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantStr, tt.seedErr.Error())
		})
	}
}

func TestSeedError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("underlying error")
	seedErr := &SeedError{
		SeedName: "Test",
		Phase:    PhaseExecution,
		Cause:    cause,
	}

	assert.Same(t, cause, seedErr.Unwrap())
	assert.ErrorIs(t, seedErr, cause)
}

func TestNewSeedError(t *testing.T) {
	t.Parallel()

	cause := errors.New("test error")
	seedErr := NewSeedError("MySeed", PhaseExecution, cause)

	assert.Equal(t, "MySeed", seedErr.SeedName)
	assert.Equal(t, PhaseExecution, seedErr.Phase)
	assert.Same(t, cause, seedErr.Cause)
}

func TestDependencyError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		depErr  *DependencyError
		wantStr string
	}{
		{
			name: "circular dependency",
			depErr: &DependencyError{
				CircularPath: []string{"A", "B", "C", "A"},
			},
			wantStr: "circular dependency detected: A -> B -> C -> A",
		},
		{
			name: "missing dependencies single",
			depErr: &DependencyError{
				SeedName:    "UserSeed",
				MissingDeps: []string{"OrgSeed"},
			},
			wantStr: `seed "UserSeed" has missing dependencies: OrgSeed`,
		},
		{
			name: "missing dependencies multiple",
			depErr: &DependencyError{
				SeedName:    "AppSeed",
				MissingDeps: []string{"ConfigSeed", "UserSeed", "OrgSeed"},
			},
			wantStr: `seed "AppSeed" has missing dependencies: ConfigSeed, UserSeed, OrgSeed`,
		},
		{
			name: "empty error",
			depErr: &DependencyError{
				SeedName: "EmptySeed",
			},
			wantStr: `dependency error for seed "EmptySeed"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantStr, tt.depErr.Error())
		})
	}
}

func TestNewMissingDependencyError(t *testing.T) {
	t.Parallel()

	err := NewMissingDependencyError("TestSeed", []string{"Dep1", "Dep2"})

	assert.Equal(t, "TestSeed", err.SeedName)
	assert.Equal(t, []string{"Dep1", "Dep2"}, err.MissingDeps)
	assert.Empty(t, err.CircularPath)
}

func TestNewCircularDependencyError(t *testing.T) {
	t.Parallel()

	path := []string{"A", "B", "C", "A"}
	err := NewCircularDependencyError(path)

	assert.Equal(t, path, err.CircularPath)
	assert.Empty(t, err.MissingDeps)
	assert.Empty(t, err.SeedName)
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		valErr  *ValidationError
		wantStr string
	}{
		{
			name: "with issues",
			valErr: &ValidationError{
				SeedName: "BadSeed",
				Issues:   []string{"invalid name", "missing version"},
			},
			wantStr: `validation error for seed "BadSeed": invalid name; missing version`,
		},
		{
			name: "single issue",
			valErr: &ValidationError{
				SeedName: "SingleIssue",
				Issues:   []string{"only one problem"},
			},
			wantStr: `validation error for seed "SingleIssue": only one problem`,
		},
		{
			name: "no issues",
			valErr: &ValidationError{
				SeedName: "NoIssues",
				Issues:   []string{},
			},
			wantStr: `validation error for seed "NoIssues"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantStr, tt.valErr.Error())
		})
	}
}

func TestNewValidationError(t *testing.T) {
	t.Parallel()

	err := NewValidationError("TestSeed", "issue1", "issue2", "issue3")

	assert.Equal(t, "TestSeed", err.SeedName)
	assert.Equal(t, []string{"issue1", "issue2", "issue3"}, err.Issues)
}

func TestRegistryError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		regErr  *RegistryError
		wantStr string
	}{
		{
			name: "with cause",
			regErr: &RegistryError{
				Message: "registration failed",
				Cause:   errors.New("duplicate seed"),
			},
			wantStr: "registry error: registration failed: duplicate seed",
		},
		{
			name: "without cause",
			regErr: &RegistryError{
				Message: "invalid operation",
				Cause:   nil,
			},
			wantStr: "registry error: invalid operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantStr, tt.regErr.Error())
		})
	}
}

func TestRegistryError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("root cause")
	regErr := &RegistryError{
		Message: "test",
		Cause:   cause,
	}

	assert.Same(t, cause, regErr.Unwrap())
	assert.ErrorIs(t, regErr, cause)
}

func TestNewRegistryError(t *testing.T) {
	t.Parallel()

	cause := errors.New("underlying")
	err := NewRegistryError("operation failed", cause)

	assert.Equal(t, "operation failed", err.Message)
	assert.Same(t, cause, err.Cause)
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("ErrSeedAlreadyRegistered", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "seed already registered", ErrSeedAlreadyRegistered.Error())
	})

	t.Run("ErrSeedNotFound", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "seed not found", ErrSeedNotFound.Error())
	})

	t.Run("ErrNoSeedsToApply", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "no seeds to apply", ErrNoSeedsToApply.Error())
	})

	t.Run("ErrUserCancelled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "operation cancelled by user", ErrUserCancelled.Error())
	})
}

func TestSeedPhase_Values(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SeedPhase("validation"), PhaseValidation)
	assert.Equal(t, SeedPhase("dependency"), PhaseDependency)
	assert.Equal(t, SeedPhase("execution"), PhaseExecution)
	assert.Equal(t, SeedPhase("commit"), PhaseCommit)
}
