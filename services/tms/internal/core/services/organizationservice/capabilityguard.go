package organizationservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
)

const brokerageEnabledField = "brokerageEnabled"

type brokerageBlocker struct {
	count    int
	singular string
	plural   string
}

// guardBrokerageDisable refuses to turn brokerage off while dependent work is
// still outstanding. Enabling is always allowed: it is purely additive and can
// strand nothing.
func (s *service) guardBrokerageDisable(
	ctx context.Context,
	entity *tenant.Organization,
) error {
	if entity.BrokerageEnabled {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.ID,
		BuID:  entity.BusinessUnitID,
	}

	current, err := s.repo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if !current.BrokerageEnabled {
		return nil
	}

	counts, err := s.repo.CountBrokerageDependencies(ctx, tenantInfo)
	if err != nil {
		return err
	}

	if !counts.HasOutstandingWork() {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		brokerageEnabledField,
		errortypes.ErrResourceInUse,
		"Cannot disable brokerage: "+describeBrokerageDependencies(counts)+
			". Resolve or close this work before turning brokerage off",
	)

	return multiErr
}

func describeBrokerageDependencies(counts *repositories.BrokerageDependencyCounts) string {
	blockers := [...]brokerageBlocker{
		{
			count:    counts.ActiveTenders,
			singular: "active tender",
			plural:   "active tenders",
		},
		{
			count:    counts.ActiveRateConfirmations,
			singular: "unvoided rate confirmation",
			plural:   "unvoided rate confirmations",
		},
		{
			count:    counts.UnpaidCarrierSettlements,
			singular: "unpaid carrier settlement",
			plural:   "unpaid carrier settlements",
		},
		{
			count:    counts.OpenCarrierInvoiceMatches,
			singular: "open carrier invoice match",
			plural:   "open carrier invoice matches",
		},
		{
			count:    counts.ActiveCarrierAssignments,
			singular: "active carrier assignment",
			plural:   "active carrier assignments",
		},
	}

	parts := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		if blocker.count == 0 {
			continue
		}

		parts = append(parts, strconv.Itoa(blocker.count)+" "+
			stringutils.Pluralize(blocker.singular, blocker.plural, blocker.count))
	}

	return strings.Join(parts, ", ")
}
