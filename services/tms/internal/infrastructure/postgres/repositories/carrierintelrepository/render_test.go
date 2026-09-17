package carrierintelrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func renderDB() *bun.DB {
	return bun.NewDB(nil, pgdialect.New())
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestUpdateCarrierSummaryLeavesVersionAndUpdatedAtAlone(t *testing.T) {
	t.Parallel()

	sql := buildCarrierSummaryUpdate(renderDB(), &repositories.UpdateCarrierIntelSummaryRequest{
		TenantInfo:     testTenant(),
		CarrierID:      pulid.MustNew("car_"),
		RiskLevel:      string(carrierintel.RiskLevelHigh),
		ReviewRequired: true,
		BlockingCount:  2,
	}).String()

	assert.NotContains(t, sql, "version", sql)
	assert.NotContains(t, sql, "updated_at", sql)
	assert.Contains(t, sql, "intel_risk_level = 'High'", sql)
	assert.Contains(t, sql, "IS DISTINCT FROM 'High'", sql)
	assert.Contains(t, sql, "IS DISTINCT FROM TRUE", sql)
	assert.Contains(t, sql, "IS DISTINCT FROM 2", sql)
}

func TestUpdateCarrierSummaryClearsAnEmptyRiskLevel(t *testing.T) {
	t.Parallel()

	sql := buildCarrierSummaryUpdate(renderDB(), &repositories.UpdateCarrierIntelSummaryRequest{
		TenantInfo: testTenant(),
		CarrierID:  pulid.MustNew("car_"),
	}).String()

	assert.Contains(t, sql, "intel_risk_level = NULL", sql)
	assert.Contains(t, sql, "IS DISTINCT FROM NULL", sql)
}

func TestInsertIgnoreDuplicateEventsTargetsTheFingerprintIndex(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenant()
	rows := []*carrierintel.CarrierIntelEvent{{
		ID:             pulid.MustNew("cievt_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		SubjectType:    carrierintel.SubjectTypeCarrier,
		SubjectID:      "car_1",
		DOTNumber:      "1234567",
		Category:       carrierintel.Section("Authority"),
		Severity:       carrierintel.SeverityHigh,
		Summary:        "Authority revoked",
		DetectedAt:     1,
		Fingerprint:    "abc",
		Status:         carrierintel.EventStatusOpen,
	}}

	sql := buildInsertIgnoreDuplicateEvents(renderDB(), &rows).String()

	assert.Contains(
		t,
		sql,
		"ON CONFLICT (organization_id, business_unit_id, provider, subject_type, subject_id, fingerprint) DO NOTHING",
		sql,
	)
	assert.Contains(t, sql, "RETURNING *", sql)
}

func TestDefaultControlInsertWritesFalseBooleans(t *testing.T) {
	t.Parallel()

	cols := buncolgen.CarrierIntelControlColumns
	falseColumns := []string{
		cols.IncludeOpenTenders.Bare(),
		cols.ExclusiveWatchlist.Bare(),
		cols.PreTenderRefreshEnabled.Bare(),
		cols.AutoDisqualifyOnBlock.Bare(),
	}

	full := buildDefaultControlInsert(renderDB(), testTenant()).String()
	assert.Contains(t, full, "ON CONFLICT (organization_id, business_unit_id) DO NOTHING", full)

	sql := buildDefaultControlInsert(renderDB(), testTenant()).Column(falseColumns...).String()
	start := strings.LastIndex(sql, "VALUES (")
	require.NotEqual(t, -1, start, sql)
	end := strings.Index(sql[start:], ")")
	require.NotEqual(t, -1, end, sql)

	values := strings.Split(sql[start+len("VALUES ("):start+end], ", ")
	require.Len(t, values, len(falseColumns), sql)
	for i, value := range values {
		assert.Equal(t, "FALSE", strings.ToUpper(value), "%s: %s", falseColumns[i], sql)
	}
}

func TestPruneHistoryDeletesOnlyRankedNonCurrentSnapshots(t *testing.T) {
	t.Parallel()

	sql := buildPruneHistoryDelete(renderDB(), testTenant(), 12).String()

	assert.Contains(
		t,
		sql,
		"row_number() OVER (PARTITION BY cisnap.subject_type, cisnap.subject_id",
		sql,
	)
	assert.Contains(t, sql, "ranked.rn > 12", sql)
	assert.Contains(t, sql, "cisnap.is_current = FALSE", sql)
}

func TestEnrollmentUpsertBumpsTheExistingRowVersion(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenant()
	rows := []*carrierintel.CarrierMonitoringEnrollment{{
		ID:             pulid.MustNew("cienr_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		SubjectType:    carrierintel.SubjectTypeCarrier,
		SubjectID:      "car_1",
		DOTNumber:      "1234567",
		Mode:           carrierintel.EnrollmentModeNative,
		DesiredState:   carrierintel.DesiredStateEnrolled,
		VendorState:    carrierintel.VendorStateUnknown,
	}}

	sql := buildEnrollmentUpsert(renderDB(), &rows).String()

	assert.Contains(
		t,
		sql,
		"ON CONFLICT (organization_id, business_unit_id, provider, subject_type, subject_id) DO UPDATE",
		sql,
	)
	assert.Contains(t, sql, "version = cienr.version + 1", sql)
	assert.Contains(t, sql, "desired_state = EXCLUDED.desired_state", sql)
}
