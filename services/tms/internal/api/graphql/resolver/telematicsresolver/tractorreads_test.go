package telematicsresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/telematicsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeTelematicsRepo struct {
	telematicsports.TelematicsRepository

	positions          []*telematics.VehiclePosition
	positionRequest    *telematicsports.ListVehiclePositionsRequest
	inspections        []*telematics.VehicleInspection
	inspectionRequests []*telematicsports.ListVehicleInspectionsRequest
	workerMappings     []telematicsports.WorkerTelematicsMapping
}

func (f *fakeTelematicsRepo) ListVehiclePositions(
	_ context.Context,
	req *telematicsports.ListVehiclePositionsRequest,
) ([]*telematics.VehiclePosition, error) {
	f.positionRequest = req
	return f.positions, nil
}

func (f *fakeTelematicsRepo) ListVehicleInspections(
	_ context.Context,
	req *telematicsports.ListVehicleInspectionsRequest,
) ([]*telematics.VehicleInspection, error) {
	f.inspectionRequests = append(f.inspectionRequests, req)
	return f.inspections, nil
}

func (f *fakeTelematicsRepo) ListWorkerMappings(
	context.Context,
	pagination.TenantInfo,
) ([]telematicsports.WorkerTelematicsMapping, error) {
	return f.workerMappings, nil
}

type tractorReadFixture struct {
	repo  *fakeTelematicsRepo
	root  *Deps
	orgID pulid.ID
	buID  pulid.ID
}

func newTractorReadFixture(grants ...string) *tractorReadFixture {
	repo := &fakeTelematicsRepo{}
	engine := &resolvertest.GrantingPermissionEngine{Granted: make(map[string]bool, len(grants))}
	for _, grant := range grants {
		engine.Granted[grant] = true
	}

	return &tractorReadFixture{
		repo: repo,
		root: &Deps{
			Core: &resolvercore.Core{PermissionEngine: engine},
			TelematicsService: telematicsservice.New(telematicsservice.Params{
				Repo:   repo,
				Logger: zap.NewNop(),
			}),
		},
		orgID: pulid.MustNew("org_"),
		buID:  pulid.MustNew("bu_"),
	}
}

func (f *tractorReadFixture) ctx(t *testing.T) context.Context {
	t.Helper()

	ctx := graphql.WithOperationContext(t.Context(), &graphql.OperationContext{})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{})
	userID := pulid.MustNew("usr_")

	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: f.orgID,
		BusinessUnitID: f.buID,
	})
}

func tractorRead() string {
	return permission.ResourceTractor.String() + "|" + string(permission.OpRead)
}

func TestVehiclePosition_ReadsOneTractorWithoutAnAgeLimit(t *testing.T) {
	t.Parallel()

	f := newTractorReadFixture(tractorRead())
	tractorID := pulid.MustNew("tr_")
	fuel := 61.5
	f.repo.positions = []*telematics.VehiclePosition{{
		TractorID:         tractorID,
		Provider:          "Samsara",
		Latitude:          32.7767,
		Longitude:         -96.797,
		SpeedMph:          58,
		FuelPercent:       &fuel,
		FormattedLocation: "I-35E near Waxahachie, TX",
		RecordedAt:        1_800_000_000,
	}}

	got, err := (&QueryResolver{f.root}).VehiclePosition(f.ctx(t), tractorID.String())
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, tractorID.String(), got.TractorID)
	assert.InDelta(t, 58.0, got.SpeedMph, 0.001)
	require.NotNil(t, got.FuelPercent)
	assert.InDelta(t, 61.5, *got.FuelPercent, 0.001)
	require.NotNil(t, f.repo.positionRequest)
	assert.Equal(t, []pulid.ID{tractorID}, f.repo.positionRequest.TractorIDs)
	assert.Zero(t, f.repo.positionRequest.MaxAgeSeconds)
	assert.Equal(t, f.orgID, f.repo.positionRequest.TenantInfo.OrgID)
	assert.Equal(t, f.buID, f.repo.positionRequest.TenantInfo.BuID)
}

func TestVehiclePosition_NoReportIsNullNotAnError(t *testing.T) {
	t.Parallel()

	f := newTractorReadFixture(tractorRead())

	got, err := (&QueryResolver{f.root}).VehiclePosition(f.ctx(t), pulid.MustNew("tr_").String())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestVehiclePosition_RequiresTractorRead(t *testing.T) {
	t.Parallel()

	f := newTractorReadFixture()

	_, err := (&QueryResolver{f.root}).VehiclePosition(f.ctx(t), pulid.MustNew("tr_").String())
	require.Error(t, err)
	assert.Nil(t, f.repo.positionRequest)
}

func TestVehicleInspections_PassesTheKeysetCursor(t *testing.T) {
	t.Parallel()

	f := newTractorReadFixture(tractorRead())
	tractorID := pulid.MustNew("tr_")
	workerID := pulid.MustNew("wrk_")
	inspectionID := pulid.MustNew("vinsp_")
	cursorID := pulid.MustNew("vinsp_")
	f.repo.workerMappings = []telematicsports.WorkerTelematicsMapping{
		{WorkerID: workerID, FirstName: "Rosa", LastName: "Diaz"},
	}
	f.repo.inspections = []*telematics.VehicleInspection{{
		ID:                    inspectionID,
		TractorID:             tractorID,
		WorkerID:              workerID,
		InspectionType:        "preTrip",
		SafetyStatus:          "unsafe",
		StartedAt:             1_799_990_000,
		EndedAt:               1_799_990_600,
		DefectCount:           2,
		UnresolvedDefectCount: 1,
	}}

	rawTractor := tractorID.String()
	limit := 6
	before := 1_800_000_000
	beforeID := cursorID.String()
	got, err := (&QueryResolver{f.root}).VehicleInspections(
		f.ctx(t),
		&rawTractor,
		nil,
		nil,
		&limit,
		&before,
		&beforeID,
	)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, inspectionID.String(), got[0].ID)
	require.NotNil(t, got[0].WorkerName)
	assert.Equal(t, "Rosa Diaz", *got[0].WorkerName)

	require.Len(t, f.repo.inspectionRequests, 1)
	req := f.repo.inspectionRequests[0]
	assert.Equal(t, tractorID, req.TractorID)
	assert.True(t, req.WorkerID.IsNil())
	assert.Equal(t, 6, req.Limit)
	assert.Equal(t, int64(1_800_000_000), req.Before)
	assert.Equal(t, cursorID, req.BeforeID)
}

func TestVehicleInspections_FirstPageHasNoCursor(t *testing.T) {
	t.Parallel()

	f := newTractorReadFixture(tractorRead())
	rawTractor := pulid.MustNew("tr_").String()

	_, err := (&QueryResolver{f.root}).VehicleInspections(
		f.ctx(t),
		&rawTractor,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.Len(t, f.repo.inspectionRequests, 1)
	assert.Zero(t, f.repo.inspectionRequests[0].Before)
	assert.True(t, f.repo.inspectionRequests[0].BeforeID.IsNil())
}

func TestVehicleInspections_RejectsAMalformedCursor(t *testing.T) {
	t.Parallel()

	f := newTractorReadFixture(tractorRead())
	rawTractor := pulid.MustNew("tr_").String()
	before := 1_800_000_000
	bad := "not-an-id"

	_, err := (&QueryResolver{f.root}).VehicleInspections(
		f.ctx(t),
		&rawTractor,
		nil,
		nil,
		nil,
		&before,
		&bad,
	)
	require.Error(t, err)
	assert.Empty(t, f.repo.inspectionRequests)
}
