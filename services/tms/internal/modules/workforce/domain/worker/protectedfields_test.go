package worker

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func storedProfile() *WorkerProfile {
	return &WorkerProfile{
		DOB:            315532800,
		LicenseNumber:  "D1234567",
		TWICCardNumber: "TWIC-998877",
		LastDrugTest:   1767225600,
		LicenseExpiry:  1893456000,
	}
}

func TestWorkerKeepsStoredPersonalDetailsForAWriterWithoutTheGroup(t *testing.T) {
	t.Parallel()

	stored := &Worker{Profile: storedProfile()}
	incoming := &Worker{Profile: &WorkerProfile{LicenseExpiry: 1924992000}}

	incoming.KeepFields(stored, 0)

	assert.Equal(t, int64(315532800), incoming.Profile.DOB)
	assert.Equal(t, "D1234567", incoming.Profile.LicenseNumber)
	assert.Equal(t, "TWIC-998877", incoming.Profile.TWICCardNumber)
	assert.Equal(t, int64(1767225600), incoming.Profile.LastDrugTest)
	assert.Equal(t, int64(1924992000), incoming.Profile.LicenseExpiry)
}

func TestWorkerTakesPersonalDetailsFromAWriterWithTheGroup(t *testing.T) {
	t.Parallel()

	stored := &Worker{Profile: storedProfile()}
	incoming := &Worker{Profile: &WorkerProfile{LicenseNumber: "N7654321"}}

	incoming.KeepFields(stored, permission.AllFieldGroups())

	assert.Equal(t, "N7654321", incoming.Profile.LicenseNumber)
	assert.Zero(t, incoming.Profile.DOB)
}

func TestWorkerWithholdsPersonalDetailsOnly(t *testing.T) {
	t.Parallel()

	entity := &Worker{FirstName: "Ana", Profile: storedProfile()}

	entity.WithholdFields(permission.FieldGroupSet(0).With(permission.FieldGroupMargins))

	assert.Equal(t, "Ana", entity.FirstName)
	assert.Equal(t, int64(1893456000), entity.Profile.LicenseExpiry)
	assert.Zero(t, entity.Profile.DOB)
	assert.Empty(t, entity.Profile.LicenseNumber)
	assert.Empty(t, entity.Profile.TWICCardNumber)
	assert.Zero(t, entity.Profile.LastDrugTest)
}

func TestWorkerWithoutAProfileIsLeftAlone(t *testing.T) {
	t.Parallel()

	entity := &Worker{FirstName: "Ana"}
	entity.WithholdFields(0)
	entity.KeepFields(&Worker{Profile: storedProfile()}, 0)

	assert.Nil(t, entity.Profile)
}
