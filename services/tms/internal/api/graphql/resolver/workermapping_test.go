package resolver

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/mappers"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newWorkerPatchFixture() *worker.Worker {
	return &worker.Worker{
		Status:     domaintypes.StatusActive,
		Type:       worker.WorkerTypeEmployee,
		DriverType: worker.DriverTypeOTR,
	}
}

func TestApplyWorkerPatch_AbsentLeavesUnchanged(t *testing.T) {
	t.Parallel()

	entity := newWorkerPatchFixture()
	require.NoError(t, mappers.ApplyWorkerPatch(entity, gqlmodel.WorkerPatchInput{}))
	assert.Equal(t, newWorkerPatchFixture(), entity)
}

func TestApplyWorkerPatch_NullRejectedOnRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		input gqlmodel.WorkerPatchInput
	}{
		{
			field: "status",
			input: gqlmodel.WorkerPatchInput{Status: graphql.OmittableOf[*domaintypes.Status](nil)},
		},
		{
			field: "type",
			input: gqlmodel.WorkerPatchInput{Type: graphql.OmittableOf[*worker.WorkerType](nil)},
		},
		{
			field: "driverType",
			input: gqlmodel.WorkerPatchInput{
				DriverType: graphql.OmittableOf[*worker.DriverType](nil),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()

			entity := newWorkerPatchFixture()
			err := mappers.ApplyWorkerPatch(entity, tc.input)
			requireRequiredPatchError(t, err, tc.field)
			assert.Equal(t, newWorkerPatchFixture(), entity)
		})
	}
}

func TestApplyWorkerPatch_ValuesAreSet(t *testing.T) {
	t.Parallel()

	status := domaintypes.StatusInactive
	workerType := worker.WorkerTypeContractor
	driverType := worker.DriverTypeLocal
	entity := newWorkerPatchFixture()
	require.NoError(t, mappers.ApplyWorkerPatch(entity, gqlmodel.WorkerPatchInput{
		Status:     graphql.OmittableOf(&status),
		Type:       graphql.OmittableOf(&workerType),
		DriverType: graphql.OmittableOf(&driverType),
	}))
	assert.Equal(t, status, entity.Status)
	assert.Equal(t, workerType, entity.Type)
	assert.Equal(t, driverType, entity.DriverType)
}
