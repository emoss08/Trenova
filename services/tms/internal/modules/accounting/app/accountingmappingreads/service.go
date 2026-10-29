package accountingmappingreads

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnlookup"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ledgerRequirement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	conn *accountingconnection.AccountingConnection,
	summary *accountingmappingports.AccountingMappingSummary,
) error {
	var since *int64
	if conn.SyncStartDate != nil && !conn.SentOpeningBalances() {
		since = conn.SyncStartDate
	}
	accounts, err := s.ledger.ListActiveAccounts(ctx, &journalentryports.ListLedgerAccountsRequest{
		TenantInfo: tenantInfo,
		Since:      since,
	})
	if err != nil {
		return err
	}
	mappings, err := s.mappings.ListByConnection(
		ctx,
		&accountingmappingports.ListAccountingMappingsQuery{
			TenantInfo:   tenantInfo,
			ConnectionID: conn.ID,
			TargetTypes: []accountingsync.MappingTargetType{
				accountingsync.TargetGLAccount,
				accountingsync.TargetAccountRole,
			},
		},
	)
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
		if mapping.State != accountingmapping.MappingStateConfirmed || mapping.ExternalID == "" {
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

func (s *Service) ConnectionFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingconnection.AccountingConnection, error) {
	return accountingconnlookup.ByType(ctx, s.connections, tenantInfo, typ)
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*accountingmappingports.AccountingMappingSummary, error) {
	profile := accountingprovider.MustProfile(integrationType)
	summary := &accountingmappingports.AccountingMappingSummary{
		IntegrationType: integrationType,
		ProviderName:    profile.Name,
		RequiredTotal:   len(profile.RequiredTargets()),
	}
	conn, err := s.ConnectionFor(ctx, tenantInfo, integrationType)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return summary, nil
		}
		return nil, err
	}
	summary.Connection = conn

	counts, err := s.mappings.CountByState(ctx, tenantInfo, conn.ID)
	if err != nil {
		return nil, err
	}
	summary.Groups = groupCounts(counts)

	if conn.SendsLedger() && s.ledger != nil {
		if err = s.ledgerRequirement(ctx, tenantInfo, conn, summary); err != nil {
			return nil, err
		}
		return summary, nil
	}

	required, err := s.mappings.ListConnection(
		ctx,
		&accountingmappingports.ListAccountingMappingsConnectionRequest{
			Filter:       &pagination.QueryOptions{TenantInfo: tenantInfo},
			Cursor:       pagination.CursorInfo{Limit: pagination.MaxLimit},
			ConnectionID: conn.ID,
			RequiredOnly: true,
			States:       []accountingmapping.MappingState{accountingmapping.MappingStateConfirmed},
		},
	)
	if err != nil {
		return nil, err
	}
	summary.RequiredConfirmed = len(required.Items)
	summary.CanCompleteSetup = conn.IsActive() &&
		summary.RequiredConfirmed == summary.RequiredTotal

	return summary, nil
}

func groupCounts(
	counts []accountingmappingports.AccountingMappingCount,
) []accountingmappingports.AccountingMappingGroup {
	byType := make(
		map[accountingsync.MappingTargetType]*accountingmappingports.AccountingMappingGroup,
	)
	groups := make(
		[]accountingmappingports.AccountingMappingGroup,
		0,
		len(accountingsync.AllMappingTargetTypes()),
	)
	for _, targetType := range accountingsync.AllMappingTargetTypes() {
		groups = append(
			groups,
			accountingmappingports.AccountingMappingGroup{TargetType: targetType},
		)
	}
	for idx := range groups {
		byType[groups[idx].TargetType] = &groups[idx]
	}
	for _, count := range counts {
		group, ok := byType[count.TargetType]
		if !ok {
			continue
		}
		switch count.State {
		case accountingmapping.MappingStateUnmatched:
			group.Unmatched += count.Count
		case accountingmapping.MappingStateProposed:
			group.Proposed += count.Count
		case accountingmapping.MappingStateConfirmed:
			group.Confirmed += count.Count
		}
	}
	return groups
}

func (s *Service) ListMappings(
	ctx context.Context,
	req *accountingmappingports.ListAccountingMappingsRequest,
) (*pagination.CursorListResult[*accountingmapping.AccountingMapping], error) {
	conn, err := s.ConnectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	return s.mappings.ListConnection(
		ctx,
		&accountingmappingports.ListAccountingMappingsConnectionRequest{
			Filter:       &pagination.QueryOptions{TenantInfo: req.TenantInfo},
			Cursor:       req.Cursor,
			ConnectionID: conn.ID,
			TargetTypes:  req.TargetTypes,
			States:       req.States,
			RequiredOnly: req.RequiredOnly,
			Search:       req.Search,
		},
	)
}

func (s *Service) GetMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingmapping.AccountingMapping, error) {
	return s.mappings.GetByID(ctx, accountingmappingports.GetAccountingMappingRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (s *Service) GetMappingsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*accountingmapping.AccountingMapping, error) {
	return s.mappings.GetByIDs(ctx, accountingmappingports.GetAccountingMappingsByIDsRequest{
		TenantInfo: tenantInfo,
		IDs:        ids,
	})
}

func (s *Service) FindMapping(
	ctx context.Context,
	req *accountingmappingports.SetAccountingMappingRequest,
) (*accountingmapping.AccountingMapping, error) {
	if err := req.ValidateTarget(); err != nil {
		return nil, err
	}
	if !req.MappingID.IsNil() {
		return s.GetMapping(ctx, req.TenantInfo, req.MappingID)
	}
	conn, err := s.ConnectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	return s.mappings.GetByTarget(ctx, &accountingmappingports.GetAccountingMappingByTargetRequest{
		TenantInfo:      req.TenantInfo,
		ConnectionID:    conn.ID,
		TargetType:      req.TargetType,
		TrenovaObjectID: req.TrenovaObjectID,
		TrenovaKey:      req.TrenovaKey,
	})
}

func (s *Service) SearchReference(
	ctx context.Context,
	req *accountingmappingports.SearchAccountingReferenceRequest,
) ([]*accountingmapping.AccountingReferenceObject, error) {
	if !req.Kind.IsValid() {
		return nil, errortypes.NewValidationError(
			"kind",
			errortypes.ErrInvalid,
			"Record kind is not recognized",
		)
	}
	conn, err := s.ConnectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	return s.references.Search(ctx, &accountingmappingports.SearchAccountingReferenceObjectsRequest{
		TenantInfo:   req.TenantInfo,
		ConnectionID: conn.ID,
		Kind:         req.Kind,
		Query:        req.Query,
		UsableOnly:   req.UsableOnly,
		Limit:        req.Limit,
	})
}

func (s *Service) GetReferenceObjects(
	ctx context.Context,
	req *accountingmappingports.GetAccountingReferenceObjectsRequest,
) ([]*accountingmapping.AccountingReferenceObject, error) {
	return s.references.GetByExternalIDs(
		ctx,
		&accountingmappingports.GetAccountingReferenceObjectsRequest{
			TenantInfo:   req.TenantInfo,
			ConnectionID: req.ConnectionID,
			Kind:         req.Kind,
			ExternalIDs:  req.ExternalIDs,
		},
	)
}

type Deps struct {
	Connections        accountingconnectionports.AccountingConnectionRepository
	References         accountingmappingports.AccountingReferenceObjectRepository
	Mappings           accountingmappingports.AccountingMappingRepository
	Ledger             journalentryports.AccountingLedgerSource
	AccountingControls tenantports.AccountingControlRepository
}

type Service struct {
	connections        accountingconnectionports.AccountingConnectionRepository
	references         accountingmappingports.AccountingReferenceObjectRepository
	mappings           accountingmappingports.AccountingMappingRepository
	ledger             journalentryports.AccountingLedgerSource
	accountingControls tenantports.AccountingControlRepository
}

func New(d Deps) *Service {
	return &Service{
		connections:        d.Connections,
		references:         d.References,
		mappings:           d.Mappings,
		ledger:             d.Ledger,
		accountingControls: d.AccountingControls,
	}
}
