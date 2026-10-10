package briefingservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeDays struct {
	asked  []briefingports.WriteBriefingRequest
	result *briefingports.WriteBriefingResult
}

func (f *fakeDays) WriteForDay(
	_ context.Context,
	req briefingports.WriteBriefingRequest,
) (*briefingports.WriteBriefingResult, error) {
	f.asked = append(f.asked, req)

	return f.result, nil
}

// Rewriting a page spends a model call, so it runs on a worker, for the one
// role asked about, on the day asked about.
func TestRegenerate_WritesTheOneRoleOnAWorker(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	written := &briefing.Briefing{ID: pulid.MustNew("brf_"), RoleKey: briefing.RoleBilling}
	days := &fakeDays{result: &briefingports.WriteBriefingResult{
		Written:   1,
		Briefings: []*briefing.Briefing{written},
	}}
	svc := &Service{l: zap.NewNop(), days: days}

	got, err := svc.Regenerate(t.Context(), briefingports.GetBriefingRequest{
		TenantInfo:   tenant,
		RoleKey:      briefing.RoleBilling,
		BriefingDate: "2026-09-23",
	}, nil)
	require.NoError(t, err)
	assert.Same(t, written, got)

	require.Len(t, days.asked, 1)
	assert.Equal(t, tenant, days.asked[0].TenantInfo)
	assert.Equal(t, []briefing.RoleKey{briefing.RoleBilling}, days.asked[0].Roles)
	assert.Equal(t, "2026-09-23", days.asked[0].BriefingDate)
}

// A role that is not one falls back to the general page rather than to none.
func TestRegenerate_AnUnknownRoleWritesTheGeneralPage(t *testing.T) {
	t.Parallel()

	days := &fakeDays{result: &briefingports.WriteBriefingResult{
		Briefings: []*briefing.Briefing{{ID: pulid.MustNew("brf_")}},
	}}
	svc := &Service{l: zap.NewNop(), days: days}

	_, err := svc.Regenerate(t.Context(), briefingports.GetBriefingRequest{RoleKey: "Nobody"}, nil)
	require.NoError(t, err)
	require.Len(t, days.asked, 1)
	assert.Equal(t, []briefing.RoleKey{briefing.RoleGeneral}, days.asked[0].Roles)
}

// A write that produced no page is a failure the person is told about, not
// an empty answer.
func TestRegenerate_NoPageWrittenIsAnError(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), days: &fakeDays{result: &briefingports.WriteBriefingResult{}}}

	_, err := svc.Regenerate(t.Context(), briefingports.GetBriefingRequest{}, nil)
	require.Error(t, err)
}
