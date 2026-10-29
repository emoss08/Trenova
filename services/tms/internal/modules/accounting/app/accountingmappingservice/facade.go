package accountingmappingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingcreate"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingdecisions"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingreads"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingscoring"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingtargets"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Service struct {
	kit       *accountingmappingkit.Kit
	reads     *accountingmappingreads.Service
	scoring   *accountingmappingscoring.Service
	creator   *accountingmappingcreate.Service
	decisions *accountingmappingdecisions.Service
}

var _ accountingmappingports.AccountingMappingService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	l := p.Logger.Named("service.accounting-mapping")
	kit := accountingmappingkit.New(accountingmappingkit.Deps{
		Logger:      l,
		Audit:       p.AuditService,
		Realtime:    p.Realtime,
		SyncRecords: p.SyncRecords,
		Dispatcher:  p.Dispatcher,
	})
	catalog := accountingmappingtargets.New(accountingmappingtargets.Deps{
		Mappings:           p.Mappings,
		AccountingControls: p.AccountingControls,
		GLAccounts:         p.GLAccounts,
		Ledger:             p.Ledger,
		Customers:          p.Customers,
		Carriers:           p.Carriers,
		Accessorials:       p.Accessorials,
		Workers:            p.Workers,
		SettlementControls: p.SettlementControls,
		PayCodes:           p.PayCodes,
	})
	reads := accountingmappingreads.New(accountingmappingreads.Deps{
		Connections:        p.Connections,
		References:         p.References,
		Mappings:           p.Mappings,
		Ledger:             p.Ledger,
		AccountingControls: p.AccountingControls,
	})
	return &Service{
		kit:   kit,
		reads: reads,
		scoring: accountingmappingscoring.New(accountingmappingscoring.Deps{
			Logger:            l,
			Connections:       p.Connections,
			References:        p.References,
			Mappings:          p.Mappings,
			ConnectionService: p.ConnectionService,
			Completion:        p.Completion,
			Kit:               kit,
			Catalog:           catalog,
		}),
		creator: accountingmappingcreate.New(accountingmappingcreate.Deps{
			Logger:            l,
			References:        p.References,
			Mappings:          p.Mappings,
			ConnectionService: p.ConnectionService,
			Accessorials:      p.Accessorials,
			Customers:         p.Customers,
			Carriers:          p.Carriers,
			Kit:               kit,
			Catalog:           catalog,
			Reads:             reads,
		}),
		decisions: accountingmappingdecisions.New(accountingmappingdecisions.Deps{
			Logger:      l,
			DB:          p.DB,
			Connections: p.Connections,
			References:  p.References,
			Mappings:    p.Mappings,
			Refresher:   p.Refresher,
			Audit:       p.AuditService,
			Kit:         kit,
			Catalog:     catalog,
			Reads:       reads,
		}),
	}
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*accountingmappingports.AccountingMappingSummary, error) {
	return s.reads.Summary(ctx, tenantInfo, integrationType)
}

func (s *Service) ListMappings(
	ctx context.Context,
	req *accountingmappingports.ListAccountingMappingsRequest,
) (*pagination.CursorListResult[*accountingmapping.AccountingMapping], error) {
	return s.reads.ListMappings(ctx, req)
}

func (s *Service) GetMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingmapping.AccountingMapping, error) {
	return s.reads.GetMapping(ctx, tenantInfo, id)
}

func (s *Service) GetMappingsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*accountingmapping.AccountingMapping, error) {
	return s.reads.GetMappingsByIDs(ctx, tenantInfo, ids)
}

func (s *Service) FindMapping(
	ctx context.Context,
	req *accountingmappingports.SetAccountingMappingRequest,
) (*accountingmapping.AccountingMapping, error) {
	return s.reads.FindMapping(ctx, req)
}

func (s *Service) SearchReference(
	ctx context.Context,
	req *accountingmappingports.SearchAccountingReferenceRequest,
) ([]*accountingmapping.AccountingReferenceObject, error) {
	return s.reads.SearchReference(ctx, req)
}

func (s *Service) GetReferenceObjects(
	ctx context.Context,
	req *accountingmappingports.GetAccountingReferenceObjectsRequest,
) ([]*accountingmapping.AccountingReferenceObject, error) {
	return s.reads.GetReferenceObjects(ctx, req)
}

func (s *Service) PlanConfirm(
	ctx context.Context,
	req *accountingmappingports.ConfirmAccountingMappingsRequest,
) (*accountingmappingports.AccountingMappingConfirmPlan, error) {
	return s.decisions.PlanConfirm(ctx, req)
}

func (s *Service) Confirm(
	ctx context.Context,
	req *accountingmappingports.ConfirmAccountingMappingsRequest,
) ([]*accountingmapping.AccountingMapping, error) {
	return s.decisions.Confirm(ctx, req)
}

func (s *Service) PlanReject(
	ctx context.Context,
	req *accountingmappingports.AccountingMappingActionRequest,
) (*accountingmappingports.AccountingMappingChange, error) {
	return s.decisions.PlanReject(ctx, req)
}

func (s *Service) Reject(
	ctx context.Context,
	req *accountingmappingports.AccountingMappingActionRequest,
) (*accountingmapping.AccountingMapping, error) {
	return s.decisions.Reject(ctx, req)
}

func (s *Service) Set(
	ctx context.Context,
	req *accountingmappingports.SetAccountingMappingRequest,
) (*accountingmapping.AccountingMapping, error) {
	return s.decisions.Set(ctx, req)
}

func (s *Service) Clear(
	ctx context.Context,
	req *accountingmappingports.AccountingMappingActionRequest,
) (*accountingmapping.AccountingMapping, error) {
	return s.decisions.Clear(ctx, req)
}

func (s *Service) GuardHistory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *accountingmapping.AccountingMapping,
	acknowledged bool,
) (int, error) {
	return s.kit.GuardHistory(ctx, tenantInfo, row, acknowledged)
}

func (s *Service) CreateReferenceRecord(
	ctx context.Context,
	req *accountingmappingports.CreateAccountingReferenceRecordRequest,
) (*accountingmapping.AccountingMapping, error) {
	return s.creator.CreateReferenceRecord(ctx, req)
}

func (s *Service) RequestRefresh(
	ctx context.Context,
	req *accountingmappingports.AccountingSetupRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.decisions.RequestRefresh(ctx, req)
}

func (s *Service) CompleteSetup(
	ctx context.Context,
	req *accountingmappingports.AccountingSetupRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.decisions.CompleteSetup(ctx, req)
}

func (s *Service) PullReference(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	kind accountingprovider.ReferenceKind,
) (*accountingmappingports.AccountingReferencePull, error) {
	return s.scoring.PullReference(ctx, tenantInfo, connectionID, kind)
}

func (s *Service) Rescore(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*accountingmappingports.AccountingRescoreResult, error) {
	return s.scoring.Rescore(ctx, tenantInfo, connectionID)
}

func (s *Service) ModelPass(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	mappingIDs []pulid.ID,
) (int, error) {
	return s.scoring.ModelPass(ctx, tenantInfo, connectionID, mappingIDs)
}

func (s *Service) MarkRefreshStarted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	return s.scoring.MarkRefreshStarted(ctx, tenantInfo, connectionID)
}

func (s *Service) MarkRefreshFinished(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	failure string,
) error {
	return s.scoring.MarkRefreshFinished(ctx, tenantInfo, connectionID, failure)
}

func (s *Service) EnsureMapping(
	ctx context.Context,
	req *accountingmappingports.EnsureAccountingMappingRequest,
) (*accountingmapping.AccountingMapping, error) {
	return s.decisions.EnsureMapping(ctx, req)
}

func (s *Service) CustomerParty(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*accountingconnectorports.AccountingPartyDraft, error) {
	return s.creator.CustomerParty(ctx, tenantInfo, customerID)
}

func (s *Service) VendorParty(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	targetType accountingsync.MappingTargetType,
	objectID pulid.ID,
) (*accountingconnectorports.AccountingPartyDraft, error) {
	return s.creator.VendorParty(ctx, tenantInfo, targetType, objectID)
}
