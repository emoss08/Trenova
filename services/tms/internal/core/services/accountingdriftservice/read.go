package accountingdriftservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingconnlookup"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
)

const summaryWindow = 7 * 24 * 60 * 60

func (s *Service) Overview(
	ctx context.Context,
	req *services.AccountingDriftOverviewRequest,
) (*services.AccountingDriftOverview, error) {
	conn, err := accountingconnlookup.ByType(
		ctx,
		s.connections,
		req.TenantInfo,
		req.IntegrationType,
	)
	if err != nil {
		return nil, err
	}
	return s.overviewOf(ctx, req.TenantInfo, conn)
}

func (s *Service) overviewOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	conn *accountingsync.AccountingConnection,
) (*services.AccountingDriftOverview, error) {
	summary, err := s.findings.Summarize(ctx, &repositories.SummarizeAccountingDriftRequest{
		TenantInfo:    tenant,
		ConnectionID:  conn.ID,
		ResolvedSince: s.now().Unix() - summaryWindow,
	})
	if err != nil {
		return nil, err
	}
	tolerance, err := s.toleranceMinor(ctx, tenant)
	if err != nil {
		return nil, err
	}
	return &services.AccountingDriftOverview{
		ConnectionID:   conn.ID,
		ProviderName:   accountingsync.ProviderName(conn.IntegrationType),
		CheckedAt:      conn.DriftCheckedAt,
		CheckError:     conn.DriftErrorMessage,
		ToleranceMinor: tolerance,
		CurrencyCode:   money.CurrencyCode(conn.ExternalHomeCurrency),
		Summary:        summary,
	}, nil
}

func (s *Service) List(
	ctx context.Context,
	req *services.ListAccountingDriftFindingsRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingDriftFinding], error) {
	conn, err := accountingconnlookup.ByType(
		ctx,
		s.connections,
		req.TenantInfo,
		req.IntegrationType,
	)
	if err != nil {
		return nil, err
	}
	filter := req.Filter
	if filter == nil {
		filter = &pagination.QueryOptions{}
	}
	filter.TenantInfo = req.TenantInfo
	return s.findings.ListConnection(
		ctx,
		&repositories.ListAccountingDriftFindingsConnectionRequest{
			Filter:       filter,
			Cursor:       req.Cursor,
			ConnectionID: conn.ID,
			Statuses:     req.Statuses,
			Kinds:        req.Kinds,
			ObjectTypes:  req.ObjectTypes,
			ObjectID:     req.ObjectID,
			Search:       req.Search,
		},
	)
}

func (s *Service) Get(
	ctx context.Context,
	req *services.GetAccountingDriftFindingRequest,
) (*accountingsync.AccountingDriftFinding, error) {
	return s.findingByID(ctx, req.TenantInfo, req.ID, false)
}

func (s *Service) CheckNow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*services.AccountingDriftOverview, error) {
	conn, err := accountingconnlookup.ByType(ctx, s.connections, tenantInfo, integrationType)
	if err != nil {
		return nil, err
	}
	if !conn.ChecksDrift() {
		return nil, errortypes.NewBusinessError(
			"{0} is not syncing, so there is nothing to compare",
			accountingsync.ProviderName(conn.IntegrationType),
		)
	}
	if s.checker == nil {
		return nil, errortypes.NewBusinessError(
			"Background work is not available on this server, so the books cannot be checked now",
		)
	}
	if err = s.checker.CheckNow(ctx, tenantInfo, conn.ID); err != nil {
		return nil, err
	}
	return s.overviewOf(ctx, tenantInfo, conn)
}
