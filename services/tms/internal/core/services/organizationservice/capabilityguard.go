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

const (
	brokerageEnabledField       = "brokerageEnabled"
	assetOperationsEnabledField = "assetOperationsEnabled"
)

type capabilityBlocker struct {
	count    int
	singular string
	plural   string
}

type capabilityDisableGuard struct {
	field    string
	noun     string
	enabled  func(*tenant.Organization) bool
	describe func(context.Context, pagination.TenantInfo) (string, error)
}

func (s *service) capabilityDisableGuards() [2]capabilityDisableGuard {
	return [...]capabilityDisableGuard{
		{
			field:    brokerageEnabledField,
			noun:     "brokerage",
			enabled:  func(o *tenant.Organization) bool { return o.BrokerageEnabled },
			describe: s.describeBrokerageDependencies,
		},
		{
			field:    assetOperationsEnabledField,
			noun:     "asset operations",
			enabled:  func(o *tenant.Organization) bool { return o.AssetOperationsEnabled },
			describe: s.describeAssetDependencies,
		},
	}
}

func (s *service) guardCapabilityDisable(
	ctx context.Context,
	entity *tenant.Organization,
) error {
	guards := s.capabilityDisableGuards()

	pending := make([]capabilityDisableGuard, 0, len(guards))
	for _, guard := range guards {
		if !guard.enabled(entity) {
			pending = append(pending, guard)
		}
	}
	if len(pending) == 0 {
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

	multiErr := errortypes.NewMultiError()
	for _, guard := range pending {
		if !guard.enabled(current) {
			continue
		}

		description, dErr := guard.describe(ctx, tenantInfo)
		if dErr != nil {
			return dErr
		}
		if description == "" {
			continue
		}

		multiErr.Add(
			guard.field,
			errortypes.ErrResourceInUse,
			"Cannot disable "+guard.noun+": "+description+
				". Resolve or close this work before turning "+guard.noun+" off",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *service) describeBrokerageDependencies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	counts, err := s.repo.CountBrokerageDependencies(ctx, tenantInfo)
	if err != nil {
		return "", err
	}
	if !counts.HasOutstandingWork() {
		return "", nil
	}

	return describeBlockers([]capabilityBlocker{
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
	}), nil
}

func (s *service) describeAssetDependencies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	counts, err := s.repo.CountAssetDependencies(ctx, tenantInfo)
	if err != nil {
		return "", err
	}
	if !counts.HasOutstandingWork() {
		return "", nil
	}

	return describeBlockers([]capabilityBlocker{
		{
			count:    counts.ActiveDriverAssignments,
			singular: "active driver assignment",
			plural:   "active driver assignments",
		},
		{
			count:    counts.UnpaidDriverSettlements,
			singular: "unpaid driver settlement",
			plural:   "unpaid driver settlements",
		},
		{
			count:    counts.LinkedPortalUsers,
			singular: "linked driver portal user",
			plural:   "linked driver portal users",
		},
	}), nil
}

func describeBlockers(blockers []capabilityBlocker) string {
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
