package codemode_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/stretchr/testify/assert"
)

func TestLimitsForClampsToTheCeilings(t *testing.T) {
	t.Parallel()

	limits := codemode.LimitsFor(60, 60)
	assert.Equal(t, 60*time.Second, limits.Wall)
	assert.Equal(t, 60, limits.Calls)
	assert.Equal(t, 5_000, limits.Rows)
	assert.Equal(t, 8<<20, limits.Bytes)
	assert.Equal(t, codemode.MaxConcurrentReads, limits.Concurrent)

	high := codemode.LimitsFor(10_000, 10_000)
	assert.Equal(t, 180*time.Second, high.Wall)
	assert.Equal(t, 200, high.Calls)
	low := codemode.LimitsFor(0, 0)
	assert.Equal(t, 15*time.Second, low.Wall)
	assert.Equal(t, 1, low.Calls)
}

func TestDefaultLimitsAreTheDocumentedOnes(t *testing.T) {
	t.Parallel()

	limits := codemode.DefaultLimits()
	assert.Equal(t, 60*time.Second, limits.Wall)
	assert.Equal(t, 60, limits.Calls)
	assert.Equal(t, int64(256<<20), limits.MemoryBytes)
}
