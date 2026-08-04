package permitservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A hand-rolled stub rather than a generated mock: the repository interface has
// eight methods and this test drives three of them. Mockery is not run against
// this repo.
type permitRepoStub struct {
	repositories.PermitRepository

	getByID func(*repositories.GetPermitByIDRequest) (*permit.Permit, error)
	create  func(*permit.Permit) (*permit.Permit, error)
	update  func(*permit.Permit) (*permit.Permit, error)
	waive   func(*repositories.WaiveRequirementRepoRequest) (*permit.Requirement, error)
}

func (s *permitRepoStub) GetByID(
	_ context.Context, req *repositories.GetPermitByIDRequest,
) (*permit.Permit, error) {
	return s.getByID(req)
}

func (s *permitRepoStub) Create(
	_ context.Context, entity *permit.Permit,
) (*permit.Permit, error) {
	return s.create(entity)
}

func (s *permitRepoStub) Update(
	_ context.Context, entity *permit.Permit,
) (*permit.Permit, error) {
	return s.update(entity)
}

func (s *permitRepoStub) WaiveRequirement(
	_ context.Context, req *repositories.WaiveRequirementRepoRequest,
) (*permit.Requirement, error) {
	return s.waive(req)
}

func (s *permitRepoStub) FindHoldReasonByCode(
	context.Context, pagination.TenantInfo, string,
) (*holdreason.HoldReason, error) {
	return nil, nil
}

func activePermit() *permit.Permit {
	expires := int64(2_000_000_000)
	return &permit.Permit{
		ID:             pulid.MustNew("pmt_"),
		ShipmentID:     pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StateID:        pulid.MustNew("us_"),
		PermitNumber:   "GA-1234",
		Status:         permit.StatusActive,
		ExpiresAt:      &expires,
	}
}

func actor() *services.RequestActor {
	return &services.RequestActor{
		UserID:        pulid.MustNew("usr_"),
		PrincipalType: services.PrincipalTypeUser,
	}
}

// shipmentRepo is left nil so resync short-circuits: this test is about the
// audit entry, and reloading the shipment is a separate concern already covered
// by the sync tests.
func newAuditingService(
	t *testing.T,
	repo repositories.PermitRepository,
) (*service, *mocks.MockAuditService) {
	t.Helper()

	auditService := mocks.NewMockAuditService(t)
	return &service{
		permitRepo:   repo,
		auditService: auditService,
		l:            zap.NewNop(),
	}, auditService
}

func TestCreatePermit_RecordsAnAuditEntry(t *testing.T) {
	entity := activePermit()

	svc, auditService := newAuditingService(t, &permitRepoStub{
		create: func(e *permit.Permit) (*permit.Permit, error) { return e, nil },
	})

	var captured *services.LogActionParams
	auditService.EXPECT().
		LogAction(mock.Anything, mock.Anything).
		RunAndReturn(func(p *services.LogActionParams, _ ...services.LogOption) error {
			captured = p
			return nil
		})

	_, err := svc.CreatePermit(t.Context(), entity, actor())
	require.NoError(t, err)

	require.NotNil(t, captured)
	assert.Equal(t, permission.ResourcePermit, captured.Resource)
	assert.Equal(t, permission.OpCreate, captured.Operation)
	assert.Equal(t, entity.ID.String(), captured.ResourceID)
	assert.Equal(t, entity.OrganizationID, captured.OrganizationID)
	assert.True(t, captured.Critical,
		"permit evidence must outlive ordinary audit retention")
}

// The waiver is the entry that matters most: it is a named person accepting the
// compliance risk of moving an oversize load without a permit.
func TestWaiveRequirement_RecordsWhoAcceptedTheRiskAndWhy(t *testing.T) {
	requirement := &permit.Requirement{
		ID:             pulid.MustNew("pmr_"),
		ShipmentID:     pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         permit.RequirementWaived,
	}
	waiver := pulid.MustNew("usr_")
	reason := "Escort booked and route surveyed with the state"

	svc, auditService := newAuditingService(t, &permitRepoStub{
		waive: func(*repositories.WaiveRequirementRepoRequest) (*permit.Requirement, error) {
			return requirement, nil
		},
	})

	var captured *services.LogActionParams
	auditService.EXPECT().
		LogAction(mock.Anything, mock.Anything).
		RunAndReturn(func(p *services.LogActionParams, _ ...services.LogOption) error {
			captured = p
			return nil
		})

	_, err := svc.WaiveRequirement(t.Context(), &services.WaiveRequirementRequest{
		RequirementID: requirement.ID,
		WaivedByID:    waiver,
		Reason:        reason,
		Actor:         &services.RequestActor{UserID: waiver, PrincipalType: services.PrincipalTypeUser},
	})
	require.NoError(t, err)

	require.NotNil(t, captured)
	// OpApprove, not OpUpdate — a waiver is an authorization act, and the
	// permission registry gates it on approve.
	assert.Equal(t, permission.OpApprove, captured.Operation)
	assert.Equal(t, permission.ResourcePermit, captured.Resource)
	assert.Equal(t, waiver, captured.UserID)
	assert.True(t, captured.Critical)
}

// A rejected waiver never happened, so it must not leave an audit entry
// suggesting someone accepted a risk they did not.
func TestWaiveRequirement_DoesNotAuditARejectedWaiver(t *testing.T) {
	svc, auditService := newAuditingService(t, &permitRepoStub{})

	_, err := svc.WaiveRequirement(t.Context(), &services.WaiveRequirementRequest{
		RequirementID: pulid.MustNew("pmr_"),
		WaivedByID:    pulid.MustNew("usr_"),
		Reason:        "too short",
		Actor:         actor(),
	})

	require.Error(t, err)
	auditService.AssertNotCalled(t, "LogAction", mock.Anything, mock.Anything)
}

// The service must stay usable where no audit service is wired, matching how
// the other optional collaborators are guarded.
func TestCreatePermit_SurvivesWithoutAnAuditService(t *testing.T) {
	entity := activePermit()

	svc := &service{
		permitRepo: &permitRepoStub{
			create: func(e *permit.Permit) (*permit.Permit, error) { return e, nil },
		},
		l: zap.NewNop(),
	}

	created, err := svc.CreatePermit(t.Context(), entity, actor())
	require.NoError(t, err)
	assert.Equal(t, entity.ID, created.ID)
}
