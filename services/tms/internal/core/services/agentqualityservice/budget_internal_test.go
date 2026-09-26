package agentqualityservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type budgetControls struct {
	repositories.AgentQualityControlRepository
	control *agentquality.Control
}

func (f *budgetControls) Get(context.Context, pagination.TenantInfo) (*agentquality.Control, error) {
	if f.control == nil {
		return nil, errortypes.NewNotFoundError("agent quality control not found")
	}
	return f.control, nil
}

type windowedUsage struct {
	repositories.AIUsageRepository
	since []int64
	spent decimal.Decimal
}

func (f *windowedUsage) EvaluationCost(
	_ context.Context,
	req repositories.AIUsageEvaluationCostRequest,
) (*repositories.AIUsageCost, error) {
	f.since = append(f.since, req.Since)
	return &repositories.AIUsageCost{CostUSD: f.spent}, nil
}

func budgetService(control *agentquality.Control, usage *windowedUsage, now int64) *Service {
	return &Service{
		l:        zap.NewNop(),
		controls: &budgetControls{control: control},
		usage:    usage,
		now:      func() int64 { return now },
	}
}

func TestCheckEvaluationBudget_StopsOnceTonightsBudgetIsSpent(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}
	control := agentquality.DefaultControl(tenant.OrgID, tenant.BuID)
	control.NightlyBudgetUSD = decimal.RequireFromString("0.40")
	usage := &windowedUsage{spent: decimal.RequireFromString("0.42")}
	now := time.Date(2026, time.March, 18, 15, 30, 0, 0, time.UTC).Unix()

	decision, err := budgetService(control, usage, now).CheckEvaluationBudget(t.Context(), tenant)
	require.NoError(t, err)
	assert.True(t, decision.Stop)
	assert.NotEmpty(t, decision.Reason)
	require.Len(t, usage.since, 2)
	assert.Equal(t, time.Date(2026, time.March, 18, 0, 0, 0, 0, time.UTC).Unix(), usage.since[0])
	assert.Equal(t, time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC).Unix(), usage.since[1])
}

func TestCheckEvaluationBudget_UsesDefaultsWhenNoControlsAreSaved(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}
	usage := &windowedUsage{spent: decimal.RequireFromString("0.10")}

	decision, err := budgetService(nil, usage, time.Now().Unix()).
		CheckEvaluationBudget(t.Context(), tenant)
	require.NoError(t, err)
	assert.False(t, decision.Stop)
}
