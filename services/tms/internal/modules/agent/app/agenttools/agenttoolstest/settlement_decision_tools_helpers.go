package agenttoolstest

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/providertest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

var ErrPerformedDuringPreview = errors.New("a preview performed a settlement action")

func BuildTool(t *testing.T, provider any) agentports.AgentTool {
	t.Helper()

	tool, ok := providertest.Build(t, provider, nil).(agentports.AgentTool)
	require.Truef(t, ok, "%T does not build a tool", provider)

	return tool
}

func PendingDriverSettlement() *driversettlement.Settlement {
	entity := &driversettlement.Settlement{
		ID:               pulid.MustNew("dstl_"),
		SettlementNumber: "DS-2041",
		Status:           driversettlement.StatusApproved,
		CurrencyCode:     "USD",
		Version:          4,
		Worker:           &worker.Worker{FirstName: "Ana", LastName: "Ruiz"},
		Lines: []*driversettlement.SettlementLine{
			{
				ID:          pulid.MustNew("dstll_"),
				Category:    driversettlement.LineCategoryEarning,
				Description: "Linehaul",
				AmountMinor: 180000,
			},
			{
				ID:          pulid.MustNew("dstll_"),
				Category:    driversettlement.LineCategoryDeduction,
				Description: "Insurance",
				AmountMinor: -6000,
			},
		},
	}
	entity.SyncTotals()

	return entity
}
