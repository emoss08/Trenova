package auditservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type state struct {
	Width float64 `json:"width"`
}

func capture(t *testing.T) (*mocks.MockAuditService, func() (*services.LogActionParams, int)) {
	t.Helper()

	auditService := mocks.NewMockAuditService(t)

	var (
		params   *services.LogActionParams
		optCount int
	)
	auditService.EXPECT().
		LogAction(mock.Anything, mock.Anything).
		RunAndReturn(func(p *services.LogActionParams, opts ...services.LogOption) error {
			params = p
			optCount = len(opts)
			return nil
		}).
		Maybe()

	return auditService, func() (*services.LogActionParams, int) { return params, optCount }
}

func TestRecord_CarriesTheActorAndScope(t *testing.T) {
	auditService, read := capture(t)
	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")

	auditservice.Record(auditService, zap.NewNop(), &auditservice.RecordParams{
		Resource:       permission.ResourcePermit,
		ResourceID:     "pmt_1",
		Operation:      permission.OpCreate,
		Actor:          services.AuditActor{UserID: userID, PrincipalType: services.PrincipalTypeUser},
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Critical:       true,
		Current:        state{Width: 8.5},
	})

	params, _ := read()
	require.NotNil(t, params)
	assert.Equal(t, permission.ResourcePermit, params.Resource)
	assert.Equal(t, "pmt_1", params.ResourceID)
	assert.Equal(t, userID, params.UserID)
	assert.Equal(t, orgID, params.OrganizationID)
	assert.True(t, params.Critical)
	assert.NotNil(t, params.CurrentState)
	assert.Nil(t, params.PreviousState)
}

// A globally scoped resource leaves the tenant unset. Stamping the acting
// user's organization onto a change every tenant sees would imply the change
// was scoped to them.
func TestRecord_LeavesTenantUnsetWhenNotSupplied(t *testing.T) {
	auditService, read := capture(t)

	auditservice.Record(auditService, zap.NewNop(), &auditservice.RecordParams{
		Resource:   permission.ResourceJurisdictionRule,
		ResourceID: "jrl_1",
		Operation:  permission.OpUpdate,
	})

	params, _ := read()
	require.NotNil(t, params)
	assert.True(t, params.OrganizationID.IsNil())
	assert.True(t, params.BusinessUnitID.IsNil())
}

// The diff option is what makes an update entry answer "what changed", so it
// has to be attached exactly when both sides are present.
func TestRecord_AttachesADiffOnlyWithBothStates(t *testing.T) {
	cases := map[string]struct {
		previous, current any
		wantOpts          int
	}{
		"both":          {state{8.5}, state{8.0}, 1},
		"current only":  {nil, state{8.0}, 0},
		"previous only": {state{8.5}, nil, 0},
		"neither":       {nil, nil, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			auditService, read := capture(t)

			auditservice.Record(auditService, zap.NewNop(), &auditservice.RecordParams{
				Resource:   permission.ResourcePermit,
				ResourceID: "pmt_1",
				Operation:  permission.OpUpdate,
				Previous:   tc.previous,
				Current:    tc.current,
			})

			_, optCount := read()
			assert.Equal(t, tc.wantOpts, optCount)
		})
	}
}

func TestRecord_AttachesCommentAndMetadataOnlyWhenPresent(t *testing.T) {
	auditService, read := capture(t)

	auditservice.Record(auditService, zap.NewNop(), &auditservice.RecordParams{
		Resource:   permission.ResourcePermit,
		ResourceID: "pmt_1",
		Operation:  permission.OpCreate,
		Comment:    "Permit recorded",
		Metadata:   map[string]any{"shipmentId": "shp_1"},
	})

	_, optCount := read()
	assert.Equal(t, 2, optCount)
}

// A nil audit service is how a service running without auditing is wired, and a
// nil params is a caller bug. Neither should panic on a path that runs after
// the audited change already succeeded.
func TestRecord_IsSafeWithoutAnAuditServiceOrParams(t *testing.T) {
	assert.NotPanics(t, func() {
		auditservice.Record(nil, zap.NewNop(), &auditservice.RecordParams{ResourceID: "x"})
	})

	auditService := mocks.NewMockAuditService(t)
	assert.NotPanics(t, func() {
		auditservice.Record(auditService, zap.NewNop(), nil)
	})
}
