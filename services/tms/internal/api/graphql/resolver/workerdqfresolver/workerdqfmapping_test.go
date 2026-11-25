package workerdqfresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func recordInput(workerID pulid.ID) *gqlmodel.RecordEmploymentVerificationInput {
	return &gqlmodel.RecordEmploymentVerificationInput{
		WorkerID:     workerID.String(),
		EmployerName: "Swift Transportation",
		Status:       ptr(employment.VerificationReceived),
		Method:       ptr(employment.VerificationByEmail),
		RequestedAt:  ptr(1795507200),
	}
}

func TestEmploymentVerificationFromInputKeepsAResponseThatAlreadyArrived(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	input := recordInput(pulid.MustNew("wrk_"))
	input.ResponseReceivedAt = ptr(1795593600)
	input.DrugAlcoholResponseReceivedAt = ptr(1795680000)
	input.HadAccidents = ptr(true)
	input.AccidentCount = ptr(2)
	input.HadDrugAlcoholViolations = ptr(true)
	input.Findings = ptr("Two preventable accidents in 2025")

	entity, err := employmentVerificationFromInput(input, tenant)
	require.NoError(t, err)

	require.NotNil(t, entity.ResponseReceivedAt)
	assert.Equal(t, int64(1795593600), *entity.ResponseReceivedAt)
	require.NotNil(t, entity.DrugAlcoholResponseReceivedAt)
	assert.Equal(t, int64(1795680000), *entity.DrugAlcoholResponseReceivedAt)
	assert.True(t, entity.HadAccidents)
	assert.Equal(t, int32(2), entity.AccidentCount)
	assert.True(t, entity.HadDrugAlcoholViolations)
	assert.Equal(t, "Two preventable accidents in 2025", entity.Findings)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "a received response with its date is valid: %v", multiErr)
}

func TestEmploymentVerificationFromInputLeavesAnUnansweredRequestEmpty(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	input := recordInput(pulid.MustNew("wrk_"))
	input.Status = ptr(employment.VerificationRequested)

	entity, err := employmentVerificationFromInput(input, tenant)
	require.NoError(t, err)

	assert.Nil(t, entity.ResponseReceivedAt)
	assert.Nil(t, entity.DrugAlcoholResponseReceivedAt)
	assert.False(t, entity.HadAccidents)
	assert.Zero(t, entity.AccidentCount)
	assert.False(t, entity.HadDrugAlcoholViolations)
	assert.Empty(t, entity.Findings)
}

func TestEmploymentVerificationFromInputRejectsAnAccidentCountOutsideInt32(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	input := recordInput(pulid.MustNew("wrk_"))
	input.ResponseReceivedAt = ptr(1795593600)
	input.AccidentCount = ptr(1 << 40)

	_, err := employmentVerificationFromInput(input, tenant)
	require.Error(t, err)
}
