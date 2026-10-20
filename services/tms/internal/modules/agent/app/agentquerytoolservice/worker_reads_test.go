package agentquerytoolservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShapeWorkerRow(t *testing.T) {
	t.Parallel()

	t.Run("a NonCompliant worker cannot be assigned", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{
			workerAssignable: true,
			workerProfileField: map[string]any{
				workerComplianceKey: "NonCompliant",
				"licenseExpiry":     int64(1_800_000_000),
			},
		}
		shapeWorkerRow(row, nil)

		assert.Equal(t, false, row[workerAssignable])
		assert.Equal(t, "Credentials are not compliant", row[workerBlockedField])
	})

	t.Run("a stored block keeps its own reason", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{
			workerAssignable:   false,
			workerBlockedField: "On leave",
			workerProfileField: map[string]any{workerComplianceKey: "NonCompliant"},
		}
		shapeWorkerRow(row, nil)

		assert.Equal(t, "On leave", row[workerBlockedField])
	})

	t.Run("a credential date with nothing on file is present and null", func(t *testing.T) {
		t.Parallel()

		profile := map[string]any{workerComplianceKey: "Compliant"}
		row := map[string]any{workerAssignable: true, workerProfileField: profile}
		shapeWorkerRow(row, map[string]struct{}{"profile.medicalCardExpiry": {}})

		assert.Equal(t, true, row[workerAssignable])
		assert.Contains(t, profile, "hazmatExpiry")
		assert.Nil(t, profile["hazmatExpiry"])
		assert.NotContains(t, profile, "medicalCardExpiry", "a withheld date is not shown as missing")
	})
}

func TestRankWorkerRowsPutsTheClosestNameFirst(t *testing.T) {
	t.Parallel()

	rows := []map[string]any{
		{workerFirstNameField: "Juana", workerLastNameField: "Smith"},
		{workerFirstNameField: "Juan", workerLastNameField: "Perez"},
		{workerFirstNameField: "Maria", workerLastNameField: "Perez"},
	}

	ranked := rankWorkerRows("juan perez", rows)

	assert.Equal(t, "Juan", ranked[0][workerFirstNameField])
	assert.Len(t, ranked, 3)
}
