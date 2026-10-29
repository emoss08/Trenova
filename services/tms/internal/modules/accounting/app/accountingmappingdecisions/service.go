package accountingmappingdecisions

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingreads"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingscore"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingtargets"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const MaxConfirmBatch = 200

func (s *Service) usableReference(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	mapping *accountingmapping.AccountingMapping,
	externalID string,
) (*accountingmapping.AccountingReferenceObject, error) {
	refs, err := s.references.GetByExternalIDs(
		ctx,
		&accountingmappingports.GetAccountingReferenceObjectsRequest{
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
		!accountingmappingscore.EligibleForRole(mapping.TrenovaKey, ref) {
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
	req *accountingmappingports.ConfirmAccountingMappingsRequest,
) ([]*accountingmapping.AccountingMapping, error) {
	if len(req.Items) == 0 {
		return []*accountingmapping.AccountingMapping{}, nil
	}
	plan, err := s.PlanConfirm(ctx, req)
	if err != nil {
		return nil, err
	}

	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		for _, change := range plan.Changes {
			if _, updateErr := s.mappings.Update(txCtx, change.After); updateErr != nil {
				return updateErr
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	connectionIDs := make([]pulid.ID, 0, 1)
	for _, change := range plan.Changes {
		s.kit.LogAudit(
			change.After,
			req.UserID,
			jsonutils.MustToJSON(change.Before),
			"Confirmed mapping for "+change.After.TargetLabel,
		)
		connectionIDs = append(connectionIDs, change.After.ConnectionID)
	}
	s.kit.PublishInvalidation(ctx, req.TenantInfo, req.UserID, plan.Confirmed)
	s.kit.RequeueMappingBlocked(ctx, req.TenantInfo, connectionIDs...)
	return plan.Confirmed, nil
}

func (s *Service) PlanConfirm(
	ctx context.Context,
	req *accountingmappingports.ConfirmAccountingMappingsRequest,
) (*accountingmappingports.AccountingMappingConfirmPlan, error) {
	if len(req.Items) > MaxConfirmBatch {
		return nil, errortypes.NewValidationError(
			"ids",
			errortypes.ErrInvalid,
			"Confirm at most {0} mappings at a time",
			MaxConfirmBatch,
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
	rows, err := s.mappings.GetByIDs(ctx, accountingmappingports.GetAccountingMappingsByIDsRequest{
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
	plan := &accountingmappingports.AccountingMappingConfirmPlan{
		Changes:   make([]*accountingmappingports.AccountingMappingChange, 0, len(rows)),
		Confirmed: make([]*accountingmapping.AccountingMapping, 0, len(rows)),
	}
	for _, row := range rows {
		if row.State == accountingmapping.MappingStateConfirmed {
			plan.Confirmed = append(plan.Confirmed, row)
			continue
		}
		if row.State != accountingmapping.MappingStateProposed {
			return nil, errortypes.NewBusinessError(
				"{0} has no proposed match to confirm; choose a record for it instead",
				row.TargetLabel,
			)
		}
		if _, err = s.usableReference(ctx, req.TenantInfo, row, row.ExternalID); err != nil {
			return nil, err
		}
		before := row.Clone()
		source := row.Source
		if req.Source == accountingmapping.MappingSourceAgent {
			source = accountingmapping.MappingSourceAgent
		}
		row.Confirm(&accountingmapping.Choice{
			ExternalID:   row.ExternalID,
			ExternalName: row.ExternalName,
			Source:       source,
			ActorID:      req.UserID,
			At:           now,
		})
		plan.Changes = append(plan.Changes, &accountingmappingports.AccountingMappingChange{
			Before: before,
			After:  row,
		})
		plan.Confirmed = append(plan.Confirmed, row)
	}

	return plan, nil
}

func (s *Service) PlanReject(
	ctx context.Context,
	req *accountingmappingports.AccountingMappingActionRequest,
) (*accountingmappingports.AccountingMappingChange, error) {
	row, err := s.reads.GetMapping(ctx, req.TenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	before := row.Clone()
	if err = row.Reject(); err != nil {
		return nil, errortypes.NewBusinessError("Only a proposed mapping can be rejected").
			WithInternal(err)
	}

	return &accountingmappingports.AccountingMappingChange{Before: before, After: row}, nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *accountingmappingports.AccountingMappingActionRequest,
) (*accountingmapping.AccountingMapping, error) {
	change, err := s.PlanReject(ctx, req)
	if err != nil {
		return nil, err
	}
	updated, err := s.mappings.Update(ctx, change.After)
	if err != nil {
		return nil, err
	}
	s.kit.LogAudit(
		updated,
		req.UserID,
		jsonutils.MustToJSON(change.Before),
		"Rejected the proposed match for "+updated.TargetLabel,
	)
	s.kit.PublishInvalidation(
		ctx,
		req.TenantInfo,
		req.UserID,
		[]*accountingmapping.AccountingMapping{updated},
	)
	return updated, nil
}

func (s *Service) Set(
	ctx context.Context,
	req *accountingmappingports.SetAccountingMappingRequest,
) (*accountingmapping.AccountingMapping, error) {
	row, err := s.reads.FindMapping(ctx, req)
	if err != nil {
		return nil, err
	}
	ref, err := s.usableReference(ctx, req.TenantInfo, row, req.ExternalID)
	if err != nil {
		return nil, err
	}

	used := 0
	if row.ExternalID != ref.ExternalID {
		if used, err = s.kit.GuardHistory(
			ctx,
			req.TenantInfo,
			row,
			req.AcknowledgeHistory,
		); err != nil {
			return nil, err
		}
	}
	before := jsonutils.MustToJSON(row)
	row.Confirm(accountingmappingkit.MappingChoice(req, ref, timeutils.NowUnix()))

	updated, err := s.mappings.Update(ctx, row)
	if err != nil {
		return nil, err
	}
	s.kit.LogAudit(
		updated,
		req.UserID,
		before,
		"Mapped "+updated.TargetLabel+" to "+ref.Label()+accountingmappingkit.HistoryNote(used),
	)
	s.kit.PublishInvalidation(
		ctx,
		req.TenantInfo,
		req.UserID,
		[]*accountingmapping.AccountingMapping{updated},
	)
	s.kit.RequeueMappingBlocked(ctx, req.TenantInfo, updated.ConnectionID)
	return updated, nil
}

func (s *Service) Clear(
	ctx context.Context,
	req *accountingmappingports.AccountingMappingActionRequest,
) (*accountingmapping.AccountingMapping, error) {
	row, err := s.reads.GetMapping(ctx, req.TenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	used, err := s.kit.GuardHistory(ctx, req.TenantInfo, row, req.AcknowledgeHistory)
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
	s.kit.LogAudit(
		updated,
		req.UserID,
		before,
		"Cleared the mapping for "+updated.TargetLabel+accountingmappingkit.HistoryNote(used),
	)
	s.kit.PublishInvalidation(
		ctx,
		req.TenantInfo,
		req.UserID,
		[]*accountingmapping.AccountingMapping{updated},
	)
	return updated, nil
}

func (s *Service) RequestRefresh(
	ctx context.Context,
	req *accountingmappingports.AccountingSetupRequest,
) (*accountingconnection.AccountingConnection, error) {
	conn, err := s.reads.ConnectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if !conn.IsActive() {
		return nil, errortypes.NewBusinessError(
			"{0} is not connected. A person must connect it from the integrations page.",
			accountingprovider.ProviderName(conn.IntegrationType),
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
	req *accountingmappingports.AccountingSetupRequest,
) (*accountingconnection.AccountingConnection, error) {
	summary, err := s.reads.Summary(ctx, req.TenantInfo, req.IntegrationType)
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
	if conn.SetupStep != accountingconnection.SetupStepMappings {
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

	if logErr := s.audit.LogAction(&auditports.LogActionParams{
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
	s.kit.PublishConnectionInvalidation(ctx, updated, req.UserID)
	return updated, nil
}

func (s *Service) EnsureMapping(
	ctx context.Context,
	req *accountingmappingports.EnsureAccountingMappingRequest,
) (*accountingmapping.AccountingMapping, error) {
	lookup := &accountingmappingports.GetAccountingMappingByTargetRequest{
		TenantInfo:      req.TenantInfo,
		ConnectionID:    req.ConnectionID,
		TargetType:      req.TargetType,
		TrenovaObjectID: req.ObjectID,
		TrenovaKey:      req.Key,
	}
	row, err := s.mappings.GetByTarget(ctx, lookup)
	if err == nil || !errortypes.IsNotFoundError(err) {
		return row, err
	}

	conn, err := s.connections.GetByID(
		ctx,
		accountingconnectionports.GetAccountingConnectionByIDRequest{
			TenantInfo: req.TenantInfo,
			ID:         req.ConnectionID,
		},
	)
	if err != nil {
		return nil, err
	}
	profile := accountingprovider.MustProfile(conn.IntegrationType)
	if !profile.OffersKey(req.TargetType, req.Key) {
		return nil, errortypes.NewValidationError(
			"targetType",
			errortypes.ErrInvalid,
			"{0} does not use this mapping",
			profile.Name,
		)
	}
	t, err := s.catalog.TargetFor(ctx, req)
	if err != nil {
		return nil, err
	}
	t.Kind = profile.KindFor(t.TargetType)
	if _, err = s.mappings.CreateMissing(ctx, []*accountingmapping.AccountingMapping{{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		ConnectionID:    req.ConnectionID,
		TargetType:      t.TargetType,
		TrenovaObjectID: t.ObjectID,
		TrenovaKey:      t.Key,
		TargetLabel:     t.Label,
		ProviderKind:    t.Kind,
		State:           accountingmapping.MappingStateUnmatched,
		Signals:         accountingmapping.MappingSignals{},
	}}); err != nil {
		return nil, err
	}
	return s.mappings.GetByTarget(ctx, lookup)
}

type Deps struct {
	Logger      *zap.Logger
	DB          kernelports.Transactor
	Connections accountingconnectionports.AccountingConnectionRepository
	References  accountingmappingports.AccountingReferenceObjectRepository
	Mappings    accountingmappingports.AccountingMappingRepository
	Refresher   accountingconnectionports.AccountingReferenceRefresher
	Audit       auditports.AuditService
	Kit         *accountingmappingkit.Kit
	Catalog     *accountingmappingtargets.Catalog
	Reads       *accountingmappingreads.Service
}

type Service struct {
	l           *zap.Logger
	db          kernelports.Transactor
	connections accountingconnectionports.AccountingConnectionRepository
	references  accountingmappingports.AccountingReferenceObjectRepository
	mappings    accountingmappingports.AccountingMappingRepository
	refresher   accountingconnectionports.AccountingReferenceRefresher
	audit       auditports.AuditService
	kit         *accountingmappingkit.Kit
	catalog     *accountingmappingtargets.Catalog
	reads       *accountingmappingreads.Service
}

//nolint:gocritic // dependency injection
func New(d Deps) *Service {
	return &Service{
		l:           d.Logger,
		db:          d.DB,
		connections: d.Connections,
		references:  d.References,
		mappings:    d.Mappings,
		refresher:   d.Refresher,
		audit:       d.Audit,
		kit:         d.Kit,
		catalog:     d.Catalog,
		reads:       d.Reads,
	}
}
