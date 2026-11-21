package documentcontrolservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/documentcontrolservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type serviceRig struct {
	service *documentcontrolservice.Service
	repo    *tenantportsmock.MockDocumentControlRepository
	audit   *auditportsmock.MockAuditService
	stored  *documentcontrol.DocumentControl
}

func newServiceRig(t *testing.T) *serviceRig {
	t.Helper()

	stored := documentcontrol.NewDefaultDocumentControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	stored.ID = pulid.MustNew("docc_")
	repo := tenantportsmock.NewMockDocumentControlRepository(t)
	repo.EXPECT().
		GetOrCreate(mock.Anything, stored.OrganizationID, stored.BusinessUnitID).
		Return(stored, nil)
	audit := auditportsmock.NewMockAuditService(t)

	return &serviceRig{
		service: documentcontrolservice.New(documentcontrolservice.Params{
			Logger:       zap.NewNop(),
			Repo:         repo,
			Validator:    documentcontrolservice.NewValidator(),
			AuditService: audit,
		}),
		repo:   repo,
		audit:  audit,
		stored: stored,
	}
}

func (r *serviceRig) update() *documentcontrol.DocumentControl {
	next := *r.stored
	next.ShipmentDraftAllowedResources = nil

	return &next
}

func TestUpdate_StoresResourcesByTheirCanonicalNames(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	entity := rig.update()
	entity.ShipmentDraftAllowedResources = []string{"Shipment_Import", "TRAILER", "shipment"}

	var saved []string
	rig.repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*documentcontrol.DocumentControl")).
		RunAndReturn(func(_ context.Context, dc *documentcontrol.DocumentControl) (*documentcontrol.DocumentControl, error) {
			saved = append([]string(nil), dc.ShipmentDraftAllowedResources...)
			return dc, nil
		}).
		Once()
	rig.audit.EXPECT().
		LogAction(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	updated, err := rig.service.Update(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.Equal(t, []string{"shipment", "trailer"}, saved)
	assert.Equal(t, []string{"shipment", "trailer"}, updated.ShipmentDraftAllowedResources)
}

func TestUpdate_RefusesAnUnknownResourceWithoutSaving(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	entity := rig.update()
	entity.ShipmentDraftAllowedResources = []string{"shipment", "railcar"}

	_, err := rig.service.Update(t.Context(), entity, pulid.MustNew("usr_"))

	multiErr, ok := errortypes.RefusalOf(err)
	require.True(t, ok, "expected a validation error, got %v", err)
	require.Len(t, multiErr.Errors, 1)
	assert.True(t, multiErr.HasErrorAt("shipmentDraftAllowedResources"))
	assert.Equal(t, "unsupported resource type", multiErr.Errors[0].Message)
}

func TestUpdate_RefusesDraftingWithNoResourcesWithoutSaving(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	entity := rig.update()
	entity.EnableShipmentDraftExtraction = true
	entity.ShipmentDraftAllowedResources = []string{}

	_, err := rig.service.Update(t.Context(), entity, pulid.MustNew("usr_"))

	multiErr, ok := errortypes.RefusalOf(err)
	require.True(t, ok, "expected a validation error, got %v", err)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "shipmentDraftAllowedResources", multiErr.Errors[0].Field())
	assert.Equal(t, "Choose at least one kind of record to draft from", multiErr.Errors[0].Message)
}
