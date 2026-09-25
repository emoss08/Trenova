package accountingmappingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxConfirmBatch = 200

type Params struct {
	fx.In

	Logger             *zap.Logger
	DB                 ports.DBConnection
	Connections        repositories.AccountingConnectionRepository
	References         repositories.AccountingReferenceObjectRepository
	Mappings           repositories.AccountingMappingRepository
	ConnectionService  services.AccountingConnectionService
	AccountingControls repositories.AccountingControlRepository
	GLAccounts         repositories.GLAccountRepository
	Customers          repositories.CustomerRepository
	Carriers           repositories.CarrierRepository
	Accessorials       repositories.AccessorialChargeRepository
	AuditService       services.AuditService
	Completion         services.CompletionService                      `optional:"true"`
	Refresher          services.AccountingReferenceRefresher           `optional:"true"`
	Realtime           services.RealtimeService                        `optional:"true"`
	SyncRecords        repositories.AccountingSyncRecordRepository     `optional:"true"`
	Dispatcher         services.AccountingSyncDispatcher               `optional:"true"`
	Workers            repositories.WorkerRepository                   `optional:"true"`
	SettlementControls repositories.CarrierSettlementControlRepository `optional:"true"`
	PayCodes           repositories.PayCodeRepository                  `optional:"true"`
}

type Service struct {
	l                  *zap.Logger
	db                 ports.DBConnection
	connections        repositories.AccountingConnectionRepository
	references         repositories.AccountingReferenceObjectRepository
	mappings           repositories.AccountingMappingRepository
	connectionService  services.AccountingConnectionService
	accountingControls repositories.AccountingControlRepository
	glAccounts         repositories.GLAccountRepository
	customers          repositories.CustomerRepository
	carriers           repositories.CarrierRepository
	accessorials       repositories.AccessorialChargeRepository
	audit              services.AuditService
	completion         services.CompletionService
	refresher          services.AccountingReferenceRefresher
	realtime           services.RealtimeService
	syncRecords        repositories.AccountingSyncRecordRepository
	dispatcher         services.AccountingSyncDispatcher
	workers            repositories.WorkerRepository
	settlementControls repositories.CarrierSettlementControlRepository
	payCodes           repositories.PayCodeRepository
}

var _ services.AccountingMappingService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.accounting-mapping"),
		db:                 p.DB,
		connections:        p.Connections,
		references:         p.References,
		mappings:           p.Mappings,
		connectionService:  p.ConnectionService,
		accountingControls: p.AccountingControls,
		glAccounts:         p.GLAccounts,
		customers:          p.Customers,
		carriers:           p.Carriers,
		accessorials:       p.Accessorials,
		audit:              p.AuditService,
		completion:         p.Completion,
		refresher:          p.Refresher,
		realtime:           p.Realtime,
		syncRecords:        p.SyncRecords,
		dispatcher:         p.Dispatcher,
		workers:            p.Workers,
		settlementControls: p.SettlementControls,
		payCodes:           p.PayCodes,
	}
}

func (s *Service) connectionFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingsync.AccountingConnection, error) {
	if !accountingsync.SupportsAccountingSync(typ) {
		return nil, errortypes.NewValidationError(
			"integrationType",
			errortypes.ErrInvalid,
			"{0} is not an accounting system Trenova can sync with",
			string(typ),
		)
	}
	conn, err := s.connections.GetByType(ctx, repositories.GetAccountingConnectionRequest{
		TenantInfo:      tenantInfo,
		IntegrationType: typ,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError(
				"{0} has not been connected yet",
				accountingsync.ProviderName(typ),
			)
		}
		return nil, err
	}
	return conn, nil
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*services.AccountingMappingSummary, error) {
	summary := &services.AccountingMappingSummary{
		IntegrationType: integrationType,
		ProviderName:    accountingsync.ProviderName(integrationType),
		RequiredTotal:   requiredTargetCount(),
	}
	conn, err := s.connectionFor(ctx, tenantInfo, integrationType)
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

	required, err := s.mappings.ListConnection(
		ctx,
		&repositories.ListAccountingMappingsConnectionRequest{
			Filter:       &pagination.QueryOptions{TenantInfo: tenantInfo},
			Cursor:       pagination.CursorInfo{Limit: pagination.MaxLimit},
			ConnectionID: conn.ID,
			RequiredOnly: true,
			States:       []accountingsync.MappingState{accountingsync.MappingStateConfirmed},
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

func requiredTargetCount() int {
	count := 0
	for _, targetType := range accountingsync.AllMappingTargetTypes() {
		for _, key := range targetType.Keys() {
			if accountingsync.IsRequiredTarget(targetType, key) {
				count++
			}
		}
	}
	return count
}

func groupCounts(counts []repositories.AccountingMappingCount) []services.AccountingMappingGroup {
	byType := make(map[accountingsync.MappingTargetType]*services.AccountingMappingGroup)
	groups := make(
		[]services.AccountingMappingGroup,
		0,
		len(accountingsync.AllMappingTargetTypes()),
	)
	for _, targetType := range accountingsync.AllMappingTargetTypes() {
		groups = append(groups, services.AccountingMappingGroup{TargetType: targetType})
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
		case accountingsync.MappingStateUnmatched:
			group.Unmatched += count.Count
		case accountingsync.MappingStateProposed:
			group.Proposed += count.Count
		case accountingsync.MappingStateConfirmed:
			group.Confirmed += count.Count
		}
	}
	return groups
}

func (s *Service) ListMappings(
	ctx context.Context,
	req *services.ListAccountingMappingsRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error) {
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	return s.mappings.ListConnection(ctx, &repositories.ListAccountingMappingsConnectionRequest{
		Filter:       &pagination.QueryOptions{TenantInfo: req.TenantInfo},
		Cursor:       req.Cursor,
		ConnectionID: conn.ID,
		TargetTypes:  req.TargetTypes,
		States:       req.States,
		RequiredOnly: req.RequiredOnly,
		Search:       req.Search,
	})
}

func (s *Service) GetMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingMapping, error) {
	return s.mappings.GetByID(ctx, repositories.GetAccountingMappingRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (s *Service) GetMappingsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*accountingsync.AccountingMapping, error) {
	return s.mappings.GetByIDs(ctx, repositories.GetAccountingMappingsByIDsRequest{
		TenantInfo: tenantInfo,
		IDs:        ids,
	})
}

func (s *Service) FindMapping(
	ctx context.Context,
	req *services.SetAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	if err := req.ValidateTarget(); err != nil {
		return nil, err
	}
	if !req.MappingID.IsNil() {
		return s.GetMapping(ctx, req.TenantInfo, req.MappingID)
	}
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	return s.mappings.GetByTarget(ctx, &repositories.GetAccountingMappingByTargetRequest{
		TenantInfo:      req.TenantInfo,
		ConnectionID:    conn.ID,
		TargetType:      req.TargetType,
		TrenovaObjectID: req.TrenovaObjectID,
		TrenovaKey:      req.TrenovaKey,
	})
}

func (s *Service) SearchReference(
	ctx context.Context,
	req *services.SearchAccountingReferenceRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	if !req.Kind.IsValid() {
		return nil, errortypes.NewValidationError(
			"kind",
			errortypes.ErrInvalid,
			"Record kind is not recognized",
		)
	}
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	return s.references.Search(ctx, &repositories.SearchAccountingReferenceObjectsRequest{
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
	req *services.GetAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	return s.references.GetByExternalIDs(ctx, &repositories.GetAccountingReferenceObjectsRequest{
		TenantInfo:   req.TenantInfo,
		ConnectionID: req.ConnectionID,
		Kind:         req.Kind,
		ExternalIDs:  req.ExternalIDs,
	})
}

func (s *Service) usableReference(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	mapping *accountingsync.AccountingMapping,
	externalID string,
) (*accountingsync.AccountingReferenceObject, error) {
	refs, err := s.references.GetByExternalIDs(
		ctx,
		&repositories.GetAccountingReferenceObjectsRequest{
			TenantInfo:   tenantInfo,
			ConnectionID: mapping.ConnectionID,
			Kind:         mapping.ProviderKind,
			ExternalIDs:  []string{externalID},
		},
	)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, errortypes.NewValidationError(
			"externalId",
			errortypes.ErrInvalid,
			"That record is not in the accounting system's data Trenova has. Refresh it and try again.",
		)
	}
	ref := refs[0]
	if !ref.Usable() {
		return nil, errortypes.NewValidationError(
			"externalId",
			errortypes.ErrInvalid,
			"{0} is inactive or cannot be used on documents",
			ref.Label(),
		)
	}
	if mapping.TargetType == accountingsync.TargetAccountRole &&
		!eligibleForRole(mapping.TrenovaKey, ref) {
		return nil, errortypes.NewValidationError(
			"externalId",
			errortypes.ErrInvalid,
			"{0} is a {1} account, which cannot be used for this role",
			ref.Label(),
			ref.AccountType,
		)
	}
	return ref, nil
}

func (s *Service) Confirm(
	ctx context.Context,
	req *services.ConfirmAccountingMappingsRequest,
) ([]*accountingsync.AccountingMapping, error) {
	if len(req.Items) == 0 {
		return []*accountingsync.AccountingMapping{}, nil
	}
	if len(req.Items) > maxConfirmBatch {
		return nil, errortypes.NewValidationError(
			"ids",
			errortypes.ErrInvalid,
			"Confirm at most {0} mappings at a time",
			maxConfirmBatch,
		)
	}

	expected := make(map[pulid.ID]string, len(req.Items))
	for _, item := range req.Items {
		expected[item.ID] = item.ExternalID
	}
	ids := make([]pulid.ID, 0, len(expected))
	for id := range expected {
		ids = append(ids, id)
	}
	rows, err := s.mappings.GetByIDs(ctx, repositories.GetAccountingMappingsByIDsRequest{
		TenantInfo: req.TenantInfo,
		IDs:        ids,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) != len(ids) {
		return nil, errortypes.NewNotFoundError("One or more mappings were not found")
	}
	for _, row := range rows {
		if row.ExternalID != expected[row.ID] {
			return nil, errortypes.NewBusinessError(
				"The proposal for {0} changed since it was shown; review it again",
				row.TargetLabel,
			)
		}
	}

	now := timeutils.NowUnix()
	confirmed := make([]*accountingsync.AccountingMapping, 0, len(rows))
	previous := make(map[pulid.ID]map[string]any, len(rows))
	for _, row := range rows {
		if row.State == accountingsync.MappingStateConfirmed {
			confirmed = append(confirmed, row)
			continue
		}
		if row.State != accountingsync.MappingStateProposed {
			return nil, errortypes.NewBusinessError(
				"{0} has no proposed match to confirm; choose a record for it instead",
				row.TargetLabel,
			)
		}
		if _, err = s.usableReference(ctx, req.TenantInfo, row, row.ExternalID); err != nil {
			return nil, err
		}
		previous[row.ID] = jsonutils.MustToJSON(row)
		source := row.Source
		if req.Source == accountingsync.MappingSourceAgent {
			source = accountingsync.MappingSourceAgent
		}
		row.Confirm(&accountingsync.Choice{
			ExternalID:   row.ExternalID,
			ExternalName: row.ExternalName,
			Source:       source,
			ActorID:      req.UserID,
			At:           now,
		})
		confirmed = append(confirmed, row)
	}

	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		for _, row := range confirmed {
			if _, ok := previous[row.ID]; !ok {
				continue
			}
			if _, updateErr := s.mappings.Update(txCtx, row); updateErr != nil {
				return updateErr
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	connectionIDs := make([]pulid.ID, 0, 1)
	for _, row := range confirmed {
		if before, ok := previous[row.ID]; ok {
			s.logAudit(row, req.UserID, before, "Confirmed mapping for "+row.TargetLabel)
			connectionIDs = append(connectionIDs, row.ConnectionID)
		}
	}
	s.publishInvalidation(ctx, req.TenantInfo, req.UserID, confirmed)
	s.requeueMappingBlocked(ctx, req.TenantInfo, connectionIDs...)
	return confirmed, nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *services.AccountingMappingActionRequest,
) (*accountingsync.AccountingMapping, error) {
	row, err := s.GetMapping(ctx, req.TenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	before := jsonutils.MustToJSON(row)
	if err = row.Reject(); err != nil {
		return nil, errortypes.NewBusinessError("Only a proposed mapping can be rejected").
			WithInternal(err)
	}
	updated, err := s.mappings.Update(ctx, row)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, req.UserID, before, "Rejected the proposed match for "+updated.TargetLabel)
	s.publishInvalidation(
		ctx,
		req.TenantInfo,
		req.UserID,
		[]*accountingsync.AccountingMapping{updated},
	)
	return updated, nil
}

func (s *Service) Set(
	ctx context.Context,
	req *services.SetAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	row, err := s.FindMapping(ctx, req)
	if err != nil {
		return nil, err
	}
	ref, err := s.usableReference(ctx, req.TenantInfo, row, req.ExternalID)
	if err != nil {
		return nil, err
	}

	used := 0
	if row.ExternalID != ref.ExternalID {
		if used, err = s.guardHistory(
			ctx,
			req.TenantInfo,
			row,
			req.AcknowledgeHistory,
		); err != nil {
			return nil, err
		}
	}

	source := req.Source
	if source != accountingsync.MappingSourceAgent {
		source = accountingsync.MappingSourceManual
	}
	before := jsonutils.MustToJSON(row)
	row.Confirm(&accountingsync.Choice{
		ExternalID:   ref.ExternalID,
		ExternalName: ref.Label(),
		Source:       source,
		Reason:       req.Reason,
		ActorID:      req.UserID,
		At:           timeutils.NowUnix(),
	})

	updated, err := s.mappings.Update(ctx, row)
	if err != nil {
		return nil, err
	}
	s.logAudit(
		updated,
		req.UserID,
		before,
		"Mapped "+updated.TargetLabel+" to "+ref.Label()+historyNote(used),
	)
	s.publishInvalidation(
		ctx,
		req.TenantInfo,
		req.UserID,
		[]*accountingsync.AccountingMapping{updated},
	)
	s.requeueMappingBlocked(ctx, req.TenantInfo, updated.ConnectionID)
	return updated, nil
}

func (s *Service) Clear(
	ctx context.Context,
	req *services.AccountingMappingActionRequest,
) (*accountingsync.AccountingMapping, error) {
	row, err := s.GetMapping(ctx, req.TenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	used, err := s.guardHistory(ctx, req.TenantInfo, row, req.AcknowledgeHistory)
	if err != nil {
		return nil, err
	}
	before := jsonutils.MustToJSON(row)
	if err = row.Clear(); err != nil {
		return nil, errortypes.NewBusinessError("{0} is not mapped", row.TargetLabel).
			WithInternal(err)
	}
	updated, err := s.mappings.Update(ctx, row)
	if err != nil {
		return nil, err
	}
	s.logAudit(
		updated,
		req.UserID,
		before,
		"Cleared the mapping for "+updated.TargetLabel+historyNote(used),
	)
	s.publishInvalidation(
		ctx,
		req.TenantInfo,
		req.UserID,
		[]*accountingsync.AccountingMapping{updated},
	)
	return updated, nil
}

func (s *Service) RequestRefresh(
	ctx context.Context,
	req *services.AccountingSetupRequest,
) (*accountingsync.AccountingConnection, error) {
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if !conn.IsActive() {
		return nil, errortypes.NewBusinessError(
			"{0} is not connected. A person must connect it from the integrations page.",
			accountingsync.ProviderName(conn.IntegrationType),
		)
	}
	if s.refresher == nil {
		return nil, errortypes.NewBusinessError(
			"Reference data refresh is not available on this server",
		)
	}
	if err = s.refresher.RequestReferenceRefresh(ctx, req.TenantInfo, conn.ID); err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *Service) CompleteSetup(
	ctx context.Context,
	req *services.AccountingSetupRequest,
) (*accountingsync.AccountingConnection, error) {
	summary, err := s.Summary(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	conn := summary.Connection
	if conn == nil {
		return nil, errortypes.NewNotFoundError(
			"{0} has not been connected yet",
			summary.ProviderName,
		)
	}
	if conn.SetupStep != accountingsync.SetupStepMappings {
		return conn, nil
	}
	if !summary.CanCompleteSetup {
		return nil, errortypes.NewBusinessError(
			"Confirm the required mappings first: {0} of {1} are confirmed",
			summary.RequiredConfirmed,
			summary.RequiredTotal,
		)
	}

	before := jsonutils.MustToJSON(conn)
	conn.FinishMappings()
	updated, err := s.connections.Update(ctx, conn)
	if err != nil {
		return nil, err
	}

	if logErr := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAccountingIntegration,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         req.UserID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  before,
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Confirmed the "+summary.ProviderName+" mappings")); logErr != nil {
		s.l.Error("failed to log accounting setup audit", zap.Error(logErr))
	}
	s.publishConnectionInvalidation(ctx, updated, req.UserID)
	return updated, nil
}

func (s *Service) logAudit(
	row *accountingsync.AccountingMapping,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceAccountingIntegration,
		ResourceID:     row.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(row),
		PreviousState:  previous,
		OrganizationID: row.OrganizationID,
		BusinessUnitID: row.BusinessUnitID,
	}
	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log accounting mapping audit", zap.Error(err))
	}
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	rows []*accountingsync.AccountingMapping,
) {
	if s.realtime == nil || len(rows) == 0 {
		return
	}
	recordID := rows[0].ID
	if len(rows) > 1 {
		recordID = pulid.Nil
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    userID,
		Resource:       permission.ResourceAccountingIntegration.String(),
		Action:         "updated",
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish accounting mapping invalidation", zap.Error(err))
	}
}

func (s *Service) publishConnectionInvalidation(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
		ActorUserID:    userID,
		Resource:       permission.ResourceAccountingIntegration.String(),
		Action:         "updated",
		RecordID:       conn.ID,
	}); err != nil {
		s.l.Warn("failed to publish accounting connection invalidation", zap.Error(err))
	}
}
