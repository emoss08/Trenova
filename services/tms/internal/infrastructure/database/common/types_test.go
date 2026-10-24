package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentConstants(t *testing.T) {
	t.Parallel()

	t.Run("EnvDevelopment", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Environment("development"), EnvDevelopment)
	})

	t.Run("EnvTest", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Environment("test"), EnvTest)
	})

	t.Run("EnvStaging", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Environment("staging"), EnvStaging)
	})

	t.Run("EnvProduction", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Environment("production"), EnvProduction)
	})
}

func TestOperationOptions(t *testing.T) {
	t.Parallel()

	t.Run("fields can be set and read", func(t *testing.T) {
		t.Parallel()
		opts := OperationOptions{
			DryRun:      true,
			Force:       true,
			Interactive: false,
			Verbose:     true,
			Target:      "20240101_init",
			Environment: EnvProduction,
		}

		assert.True(t, opts.DryRun)
		assert.True(t, opts.Force)
		assert.False(t, opts.Interactive)
		assert.True(t, opts.Verbose)
		assert.Equal(t, "20240101_init", opts.Target)
		assert.Equal(t, EnvProduction, opts.Environment)
	})
}
