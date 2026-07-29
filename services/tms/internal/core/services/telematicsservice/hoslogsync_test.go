package telematicsservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeDutyStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, telematics.DutyStatusSleeperBed, normalizeDutyStatus("sleeperBed"))
	assert.Equal(t, telematics.DutyStatusDriving, normalizeDutyStatus("driving"))
	assert.Equal(t, telematics.DutyStatusOnDuty, normalizeDutyStatus("somethingNew"),
		"unknown provider statuses must count as on-duty, never as rest")
	assert.Equal(t, telematics.DutyStatusOnDuty, normalizeDutyStatus(""))
}
