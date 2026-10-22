package compiler

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func withoutGrant(engine *fakePermissionEngine, resources ...string) *fakePermissionEngine {
	for _, resource := range resources {
		engine.overrides[resource] = resourceGrant{
			dataScope:      permission.DataScopeOrganization,
			maxSensitivity: permission.SensitivityInternal,
		}
	}
	return engine
}

func shipmentReport(columns ...report.ColumnSpec) *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   columns,
	}
}

func requireAuthorizationError(t *testing.T, err error, contains string) {
	t.Helper()

	require.Error(t, err)
	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Contains(t, err.Error(), contains)
}

func TestAuthorize_AReferenceFieldOfALinkedKindNeedsOnlyTheParentsRead(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(withoutGrant(allowAllEngine(), "customer"))

	compiled, err := c.Compile(t.Context(), newRequest(shipmentReport(
		dim("c_pro", "proNumber"),
		dim("c_customer", "name", "customer"),
		dim("c_code", "code", "customer"),
	)))
	require.NoError(t, err, "a customer's name and code are in its reference view")
	assert.Contains(t, compiled.SQL, "name")
}

func TestAuthorize_AFieldOutsideTheViewNeedsTheKindsRead(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(withoutGrant(allowAllEngine(), "customer"))

	_, err := c.Compile(t.Context(), newRequest(shipmentReport(
		dim("c_customer", "name", "customer"),
		dim("c_dot", "dotNumber", "customer"),
	)))
	requireAuthorizationError(t, err, "you do not have read access to Customers")
}

func TestAuthorize_AFilterOutsideTheViewNeedsTheKindsRead(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(withoutGrant(allowAllEngine(), "customer"))

	def := shipmentReport(dim("c_pro", "proNumber"))
	def.Filters = &report.FilterGroup{
		Op: report.BoolOpAnd,
		Filters: []report.FieldFilter{{
			Ref:      report.FieldRef{Path: []string{"customer"}, Field: "dotNumber"},
			Operator: dbtype.OpEqual,
			Value:    "123",
		}},
	}

	_, err := c.Compile(t.Context(), newRequest(def))
	requireAuthorizationError(t, err, "you do not have read access to Customers")
}

func TestAuthorize_ExportIsRequiredOnlyWhereTheKindsReadIs(t *testing.T) {
	t.Parallel()

	engine := allowAllEngine()
	engine.overrides["customer"] = resourceGrant{
		operations:     []permission.Operation{permission.OpRead},
		dataScope:      permission.DataScopeOrganization,
		maxSensitivity: permission.SensitivityConfidential,
	}
	c := newTestCompiler(engine)

	_, err := c.Compile(t.Context(), newRequest(shipmentReport(
		dim("c_customer", "name", "customer"),
	)))
	require.NoError(t, err, "a reference-only join does not export the customer")

	_, err = c.Compile(t.Context(), newRequest(shipmentReport(
		dim("c_dot", "dotNumber", "customer"),
	)))
	requireAuthorizationError(t, err, "you do not have export access to Customers")
}

func TestAuthorize_TheCeilingAppliesInsideAView(t *testing.T) {
	t.Parallel()

	engine := allowAllEngine()
	engine.overrides["customer"] = resourceGrant{
		dataScope:      permission.DataScopeOrganization,
		maxSensitivity: permission.SensitivityPublic,
	}
	c := newTestCompiler(engine)

	_, err := c.Compile(t.Context(), newRequest(shipmentReport(
		dim("c_customer", "name", "customer"),
	)))
	requireAuthorizationError(t, err, `field "Name" on Customer requires internal access`)
}

func TestAuthorize_AnIntermediateRecordIsFollowedOnlyThroughItsView(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(withoutGrant(allowAllEngine(), "worker", "fleet_code", "job_position"))
	moveReport := func(columns ...report.ColumnSpec) *report.Definition {
		return &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment_move",
			Columns:   columns,
		}
	}

	_, err := c.Compile(t.Context(), newRequest(moveReport(
		dim("c_fleet", "code", "assignment", "primaryWorker", "fleetCode"),
	)))
	require.NoError(t, err, "a worker's fleet code is in the worker view, and its code in the fleet code view")

	_, err = c.Compile(t.Context(), newRequest(moveReport(
		dim("c_position", "id", "assignment", "primaryWorker", "position"),
	)))
	requireAuthorizationError(t, err, "you do not have read access to Workers")
}

func TestAuthorize_ALinkedWorkersNameNeedsNoWorkerRead(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(withoutGrant(allowAllEngine(), "worker"))

	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment_move",
		Columns: []report.ColumnSpec{
			dim("c_first", "firstName", "assignment", "primaryWorker"),
			dim("c_last", "lastName", "assignment", "primaryWorker"),
		},
	}
	_, err := c.Compile(t.Context(), newRequest(def))
	require.NoError(t, err)

	def.Columns = append(def.Columns,
		dim("c_phone", "phoneNumber", "assignment", "primaryWorker"))
	_, err = c.Compile(t.Context(), newRequest(def))
	requireAuthorizationError(t, err, "you do not have read access to Workers")
}

func TestAuthorize_AWorkersProfileIsNeverInTheWorkerView(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(withoutGrant(allowAllEngine(), "worker"))

	_, err := c.Compile(t.Context(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment_move",
		Columns: []report.ColumnSpec{
			dim("c_cdl", "cdlClass", "assignment", "primaryWorker", "profile"),
		},
	}))
	requireAuthorizationError(t, err, "you do not have read access to Workers")
}

func TestAuthorize_WithoutReferenceViewsALinkedKindNeedsItsRead(t *testing.T) {
	t.Parallel()

	engine := withoutGrant(allowAllEngine(), "customer")
	c := NewWithCatalog(
		&reportcatalog.Default,
		engine,
		permission.NewRegistry(),
		&config.ReportingConfig{},
		zap.NewNop(),
	)

	_, err := c.Compile(t.Context(), newRequest(shipmentReport(
		dim("c_customer", "name", "customer"),
	)))
	requireAuthorizationError(t, err, "you do not have read access to Customers")
}

func TestAuthorize_AProtectedFieldNeedsItsGroup(t *testing.T) {
	t.Parallel()

	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "worker",
		Columns: []report.ColumnSpec{
			dim("c_first", "firstName"),
			dim("c_license", "licenseNumber", "profile"),
		},
	}

	withheld := newTestCompilerWithGroups(allowAllEngine(),
		permission.FieldGroupSet(0).With(permission.FieldGroupMargins))
	_, err := withheld.Compile(t.Context(), newRequest(def))
	requireAuthorizationError(t, err,
		`field "License Number" on Worker Profile is a protected field and needs protected_personal access`)

	granted := newTestCompilerWithGroups(allowAllEngine(),
		permission.FieldGroupSet(0).With(permission.FieldGroupPersonal))
	_, err = granted.Compile(t.Context(), newRequest(def))
	require.NoError(t, err)

	_, err = withheld.Compile(t.Context(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "worker",
		Columns:   []report.ColumnSpec{dim("c_first", "firstName")},
	}))
	require.NoError(t, err, "a report without a protected field does not need the group")
}

func TestAuthorize_AProtectedMarginIsWithheldOnAgreementsAndVersions(t *testing.T) {
	t.Parallel()

	c := newTestCompilerWithGroups(allowAllEngine(), 0)

	for _, entity := range []string{"rate_agreement", "rate_agreement_version"} {
		_, err := c.ValidateAndAuthorize(t.Context(), newRequest(&report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entity,
			Columns:   []report.ColumnSpec{dim("c_floor", "marginFloorPercent")},
		}))
		requireAuthorizationError(t, err, "protected_margins")
	}
}

func TestAuthorize_EmploymentEventsNeedTheirOwnReadBesideAWorkerRead(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(withoutGrant(allowAllEngine(), "worker_employment_event"))

	_, err := c.Compile(t.Context(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "worker_employment_event",
		Columns: []report.ColumnSpec{
			dim("c_kind", "kind"),
			dim("c_first", "firstName", "worker"),
		},
	}))
	requireAuthorizationError(t, err, "you do not have read access to Employment Events")
}
