package accountingmappingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ledgerRequirement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	conn *accountingsync.AccountingConnection,
	summary *services.AccountingMappingSummary,
) error {
	var since *int64
	if conn.SyncStartDate != nil && !conn.SentOpeningBalances() {
		since = conn.SyncStartDate
	}
	accounts, err := s.ledger.ListActiveAccounts(ctx, &repositories.ListLedgerAccountsRequest{
		TenantInfo: tenantInfo,
		Since:      since,
	})
	if err != nil {
		return err
	}
	mappings, err := s.mappings.ListByConnection(ctx, &repositories.ListAccountingMappingsRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: conn.ID,
		TargetTypes: []accountingsync.MappingTargetType{
			accountingsync.TargetGLAccount,
			accountingsync.TargetAccountRole,
		},
	})
	if err != nil {
		return err
	}
	control, err := s.accountingControls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}

	byAccount := make(map[pulid.ID]struct{}, len(mappings))
	byRole := make(map[string]struct{}, 8)
	for _, mapping := range mappings {
		if mapping.State != accountingsync.MappingStateConfirmed || mapping.ExternalID == "" {
			continue
		}
		if mapping.TargetType == accountingsync.TargetGLAccount {
			byAccount[mapping.TrenovaObjectID] = struct{}{}
			continue
		}
		byRole[mapping.TrenovaKey] = struct{}{}
	}

	confirmed := 0
	for idx := range accounts {
		if _, ok := byAccount[accounts[idx].ID]; ok {
			confirmed++
			continue
		}
		if role := accountingsync.LedgerAccountRole(accounts[idx].ID, control); role != "" {
			if _, ok := byRole[role]; ok {
				confirmed++
			}
		}
	}
	summary.RequiredTotal = len(accounts)
	summary.RequiredConfirmed = confirmed
	summary.CanCompleteSetup = conn.IsActive() && confirmed == len(accounts)
	return nil
}
