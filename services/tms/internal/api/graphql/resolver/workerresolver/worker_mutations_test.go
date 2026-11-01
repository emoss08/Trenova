package workerresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newWorkerInput() gqlmodel.WorkerInput {
	return gqlmodel.WorkerInput{
		StateID:      pulid.MustNew("us_").String(),
		Status:       domaintypes.StatusActive,
		Type:         worker.WorkerTypeEmployee,
		DriverType:   worker.DriverTypeOTR,
		Gender:       worker.GenderFemale,
		FirstName:    "Ana",
		LastName:     "Lopez",
		AddressLine1: "2 Elm St",
		City:         "Akron",
		PostalCode:   "44301",
	}
}

func TestWorkerMutations_CheckTheWorkerPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("wrk_").String()
	input := newWorkerInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createWorker", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateWorker(ctx, input)
			return err
		}},
		{"updateWorker", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateWorker(ctx, id, input)
			return err
		}},
		{"patchWorker", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PatchWorker(ctx, id, gqlmodel.WorkerPatchInput{})
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, permission.ResourceWorker, call.operation)
		})
	}
}

func TestWorkerFromInput_MapsTheWorkerAndItsProfileIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	input := newWorkerInput()
	managerID := pulid.MustNew("usr_")
	managerRaw := managerID.String()
	phone := "555-0101"
	licenseExpiry, hireDate, medical := 1_900_000_000, 1_600_000_000, 1_800_000_000
	npi := "1234567890"
	input.ManagerID = &managerRaw
	input.PhoneNumber = &phone
	input.CustomFields = map[string]any{"shift": "night"}
	input.Profile = &gqlmodel.WorkerProfileInput{
		CdlClass:           worker.CDLClassA,
		Endorsement:        worker.EndorsementType("N"),
		LicenseExpiry:      &licenseExpiry,
		HireDate:           &hireDate,
		MedicalCardExpiry:  &medical,
		MedicalExaminerNPI: &npi,
		ComplianceStatus:   worker.ComplianceStatus("Compliant"),
		IsQualified:        true,
		ShortHaulExempt:    true,
	}

	entity, err := workerFromInput(&input, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, managerID, entity.ManagerID)
	assert.True(t, entity.FleetCodeID.IsNil())
	assert.Equal(t, "555-0101", entity.PhoneNumber)
	assert.Equal(t, map[string]any{"shift": "night"}, entity.CustomFields)
	require.NotNil(t, entity.Profile)
	assert.Equal(t, authCtx.OrganizationID, entity.Profile.OrganizationID)
	assert.Equal(t, int64(1_900_000_000), entity.Profile.LicenseExpiry)
	assert.Equal(t, int64(1_600_000_000), entity.Profile.HireDate)
	assert.Equal(t, int64(1_800_000_000), *entity.Profile.MedicalCardExpiry)
	assert.Nil(t, entity.Profile.HazmatExpiry)
	assert.Equal(t, "1234567890", entity.Profile.MedicalExaminerNPI)
	assert.True(t, entity.Profile.ShortHaulExempt)
}

func TestApplyWorkerInput_KeepsTheProfilesRollUpsAndIdentity(t *testing.T) {
	t.Parallel()

	profileID := pulid.MustNew("wrkp_")
	existing := &worker.Worker{
		ID: pulid.MustNew("wrk_"),
		Profile: &worker.WorkerProfile{
			ID:                  profileID,
			TrainingHealth:      worker.TrainingHealth("Overdue"),
			SafetyScore:         72,
			LastComplianceCheck: 1_700_000_000,
		},
	}
	input := newWorkerInput()
	input.Profile = &gqlmodel.WorkerProfileInput{
		CdlClass:         worker.CDLClassB,
		Endorsement:      worker.EndorsementType("O"),
		ComplianceStatus: worker.ComplianceStatus("Pending"),
	}

	require.NoError(t, applyWorkerInput(existing, &input))

	assert.Equal(t, profileID, existing.Profile.ID)
	assert.Equal(t, worker.TrainingHealth("Overdue"), existing.Profile.TrainingHealth)
	assert.Equal(t, int16(72), existing.Profile.SafetyScore)
	assert.Equal(t, int64(1_700_000_000), existing.Profile.LastComplianceCheck)
	assert.Equal(t, worker.CDLClassB, existing.Profile.CDLClass)
}

func TestApplyWorkerInput_NamesAMalformedLicenseState(t *testing.T) {
	t.Parallel()

	input := newWorkerInput()
	bad := "nope"
	input.Profile = &gqlmodel.WorkerProfileInput{LicenseStateID: &bad}

	err := applyWorkerInput(&worker.Worker{}, &input)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "profile.licenseStateId", valErr.Field)
}

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
	require.NoError(t, applyWorkerPatch(entity, &gqlmodel.WorkerPatchInput{}))
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
			err := applyWorkerPatch(entity, &tc.input)
			resolvertest.RequireRequiredPatchError(t, err, tc.field)
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
	require.NoError(t, applyWorkerPatch(entity, &gqlmodel.WorkerPatchInput{
		Status:     graphql.OmittableOf(&status),
		Type:       graphql.OmittableOf(&workerType),
		DriverType: graphql.OmittableOf(&driverType),
	}))
	assert.Equal(t, status, entity.Status)
	assert.Equal(t, workerType, entity.Type)
	assert.Equal(t, driverType, entity.DriverType)
}
