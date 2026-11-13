package agentturn

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClock_IsTheMomentTheTurnRunsAsOf(t *testing.T) {
	t.Parallel()

	asked := time.Date(2026, time.March, 3, 9, 30, 0, 0, time.UTC).Unix()

	assert.Equal(t, asked, (&RuntimeContext{Now: asked}).Clock(),
		"a replay set back to when the person asked keeps that moment")
}

func TestClock_IsNowForATurnWithNoMomentOfItsOwn(t *testing.T) {
	t.Parallel()

	before := time.Now().Unix()
	clock := (&RuntimeContext{}).Clock()

	assert.GreaterOrEqual(t, clock, before)
	assert.LessOrEqual(t, clock, time.Now().Unix())
}
