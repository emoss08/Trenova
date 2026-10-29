package accountingmappingscoring

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingscore"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingtargets"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func (s *Service) ModelPass(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	mappingIDs []pulid.ID,
) (int, error) {
	if s.completion == nil || len(mappingIDs) == 0 {
		return 0, nil
	}
	if len(mappingIDs) > maxModelTargets {
		mappingIDs = mappingIDs[:maxModelTargets]
	}

	rows, err := s.mappings.GetByIDs(ctx, accountingmappingports.GetAccountingMappingsByIDsRequest{
		TenantInfo: tenantInfo,
		IDs:        mappingIDs,
	})
	if err != nil {
		return 0, err
	}

	pending := make([]*accountingmapping.AccountingMapping, 0, len(rows))
	for _, row := range rows {
		if row.ConnectionID == connectionID && row.NeedsModelReview() {
			pending = append(pending, row)
		}
	}

	applied := 0
	for start := 0; start < len(pending); start += modelBatchSize {
		batch := newModelBatch(pending[start:min(start+modelBatchSize, len(pending))])
		changed, reviewed, batchErr := s.runModelBatch(ctx, tenantInfo, connectionID, batch)
		if batchErr != nil {
			if errors.Is(batchErr, llmports.ErrNoProviderConfigured) {
				return applied, nil
			}
			return applied, batchErr
		}
		count, applyErr := s.mappings.ApplyScoring(ctx, changed)
		if applyErr != nil {
			return applied, applyErr
		}
		applied += int(count)
		if !reviewed {
			continue
		}
		if _, applyErr = s.mappings.ApplyScoring(ctx, batch.declined(changed)); applyErr != nil {
			return applied, applyErr
		}
	}

	return applied, nil
}

func (s *Service) runModelBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	batch *modelBatch,
) ([]*accountingmapping.AccountingMapping, bool, error) {
	result, err := s.completion.CompleteStructured(ctx, &llmports.StructuredCompletionRequest{
		TenantInfo:   tenantInfo,
		Task:         aitask.TaskAccountingMapping,
		System:       modelSystemPrompt,
		Context:      batch.context(),
		OutputSchema: modelSchema(),
		SchemaName:   modelSchemaName,
		MaxTokens:    modelMaxTokens,
		Attribution: aiusageports.AIUsageAttribution{
			Feature: aiusage.FeatureAccountingMapping,
			Subject: aiusage.Subject{
				Type: aiusage.SubjectTypeAccountingConnection,
				ID:   connectionID.String(),
			},
		},
	})
	if err != nil {
		return nil, false, err
	}

	var reply modelReply
	if err = sonic.UnmarshalString(result.Text, &reply); err != nil {
		s.l.Warn("the mapping model's reply could not be read", zap.Error(err))
		return nil, false, nil
	}

	for _, row := range batch.rows {
		row.MarkModelReviewed()
	}
	return batch.apply(reply.Answers), true, nil
}

func (s *Service) PullReference(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	kind accountingprovider.ReferenceKind,
) (*accountingmappingports.AccountingReferencePull, error) {
	if !kind.IsValid() {
		return nil, fmt.Errorf("unknown reference kind %q", kind)
	}
	session, err := s.sessions.Session(ctx, tenantInfo, connectionID)
	if err != nil {
		return nil, err
	}
	reader, ok := session.Connector.(accountingconnectorports.AccountingReferenceReader)
	if !ok {
		return nil, errReferenceUnsupported
	}
	profile := accountingprovider.MustProfile(session.Connection.IntegrationType)
	if !profile.Has(kind) {
		return &accountingmappingports.AccountingReferencePull{Kind: kind}, nil
	}

	seenAt := timeutils.NowUnix()
	pull := &accountingmappingports.AccountingReferencePull{Kind: kind}
	start := 1
	for pages := 0; start > 0; pages++ {
		if pages >= maxReferencePagesGuard {
			return nil, fmt.Errorf("stopped after %d pages of %s records", pages, kind)
		}
		page, listErr := reader.ListReference(
			ctx,
			&accountingconnectorports.AccountingReferencePageRequest{
				RealmID:       session.Connection.ExternalRealmID,
				AccessToken:   session.AccessToken,
				Kind:          kind,
				StartPosition: start,
				PageSize:      reader.MaxReferencePageSize(),
			},
		)
		if listErr != nil {
			s.sessions.ReportCallFailure(ctx, tenantInfo, connectionID, listErr)
			return nil, listErr
		}
		if err = s.references.Upsert(
			ctx,
			&accountingmappingports.UpsertAccountingReferenceObjectsRequest{
				TenantInfo:   tenantInfo,
				ConnectionID: connectionID,
				Objects:      page.Objects,
				SeenAt:       seenAt,
			},
		); err != nil {
			return nil, err
		}
		pull.Fetched += len(page.Objects)
		start = page.NextStart
	}

	removed, err := s.references.MarkRemovedUnseen(
		ctx,
		&accountingmappingports.MarkAccountingReferenceRemovedRequest{
			TenantInfo:   tenantInfo,
			ConnectionID: connectionID,
			Kind:         kind,
			SeenBefore:   seenAt,
			At:           timeutils.NowUnix(),
		},
	)
	if err != nil {
		return nil, err
	}
	pull.Removed = removed
	return pull, nil
}

func (s *Service) Rescore(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*accountingmappingports.AccountingRescoreResult, error) {
	conn, err := s.connections.GetByID(
		ctx,
		accountingconnectionports.GetAccountingConnectionByIDRequest{
			TenantInfo: tenantInfo,
			ID:         connectionID,
		},
	)
	if err != nil {
		return nil, err
	}
	targets, err := s.catalog.ListTargets(ctx, tenantInfo, conn)
	if err != nil {
		return nil, err
	}
	result := &accountingmappingports.AccountingRescoreResult{Targets: len(targets)}

	fresh := make([]*accountingmapping.AccountingMapping, 0, len(targets))
	for _, t := range targets {
		fresh = append(fresh, &accountingmapping.AccountingMapping{
			OrganizationID:  tenantInfo.OrgID,
			BusinessUnitID:  tenantInfo.BuID,
			ConnectionID:    connectionID,
			TargetType:      t.TargetType,
			TrenovaObjectID: t.ObjectID,
			TrenovaKey:      t.Key,
			TargetLabel:     t.Label,
			ProviderKind:    t.Kind,
			State:           accountingmapping.MappingStateUnmatched,
			Signals:         accountingmapping.MappingSignals{},
		})
	}
	if result.Created, err = s.mappings.CreateMissing(ctx, fresh); err != nil {
		return nil, err
	}

	rows, err := s.mappings.ListByConnection(
		ctx,
		&accountingmappingports.ListAccountingMappingsQuery{
			TenantInfo:   tenantInfo,
			ConnectionID: connectionID,
		},
	)
	if err != nil {
		return nil, err
	}

	refs, err := s.loadReferences(ctx, tenantInfo, connectionID)
	if err != nil {
		return nil, err
	}

	byIdentity := make(map[string]*accountingmappingscore.Target, len(targets))
	for _, t := range targets {
		byIdentity[t.Identity()] = t
	}

	changed := make([]*accountingmapping.AccountingMapping, 0, len(rows))
	relabeled := make([]*accountingmapping.AccountingMapping, 0)
	for _, row := range rows {
		t, ok := byIdentity[accountingmappingtargets.MappingIdentity(row)]
		if !ok {
			continue
		}
		switch refs.rescore(row, t) {
		case rescoreChanged:
			changed = append(changed, row)
		case rescoreRelabeled:
			relabeled = append(relabeled, row)
		case rescoreUnchanged:
		}
		tallyRow(result, row)
	}

	if result.Updated, err = s.mappings.ApplyScoring(ctx, changed); err != nil {
		return nil, err
	}
	if err = s.mappings.UpdateLabels(ctx, relabeled); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) loadReferences(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*referenceSet, error) {
	set := &referenceSet{
		byKind: make(
			map[accountingprovider.ReferenceKind][]*accountingmapping.AccountingReferenceObject,
		),
		indexes: make(map[accountingprovider.ReferenceKind]*accountingmappingscore.TokenIndex),
	}
	for _, kind := range accountingprovider.AllReferenceKinds() {
		refs, err := s.references.ListByKind(
			ctx,
			&accountingmappingports.ListAccountingReferenceObjectsRequest{
				TenantInfo:   tenantInfo,
				ConnectionID: connectionID,
				Kind:         kind,
			},
		)
		if err != nil {
			return nil, err
		}
		set.byKind[kind] = refs
		if kind == accountingprovider.ReferenceKindCustomer ||
			kind == accountingprovider.ReferenceKindVendor {
			set.indexes[kind] = accountingmappingscore.NewTokenIndex(refs)
		}
	}
	return set, nil
}

func (s *Service) MarkRefreshStarted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	now := timeutils.NowUnix()
	return s.connections.MarkReferenceRefresh(
		ctx,
		accountingconnectionports.MarkAccountingReferenceRefreshRequest{
			TenantInfo: tenantInfo,
			ID:         connectionID,
			StartedAt:  &now,
		},
	)
}

func (s *Service) MarkRefreshFinished(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	failure string,
) error {
	req := accountingconnectionports.MarkAccountingReferenceRefreshRequest{
		TenantInfo: tenantInfo,
		ID:         connectionID,
		Error:      stringutils.TruncateRunes(failure, MaxRefreshErrorLength),
	}
	if failure == "" {
		now := timeutils.NowUnix()
		req.RefreshedAt = &now
	}
	if err := s.connections.MarkReferenceRefresh(ctx, req); err != nil {
		return err
	}

	conn, err := s.connections.GetByID(
		ctx,
		accountingconnectionports.GetAccountingConnectionByIDRequest{
			TenantInfo: tenantInfo,
			ID:         connectionID,
		},
	)
	if err != nil {
		s.l.Warn("could not reload the connection after a refresh", zap.Error(err))
		return nil
	}
	s.kit.PublishConnectionInvalidation(ctx, conn, pulid.Nil)
	return nil
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	Connections accountingconnectionports.AccountingConnectionRepository
	References  accountingmappingports.AccountingReferenceObjectRepository
	Mappings    accountingmappingports.AccountingMappingRepository
	Sessions    accountingconnectionports.AccountingSessionProvider
	Completion  llmports.CompletionService `optional:"true"`
	Kit         *accountingmappingkit.Kit
	Catalog     *accountingmappingtargets.Catalog
}

type Service struct {
	l           *zap.Logger
	connections accountingconnectionports.AccountingConnectionRepository
	references  accountingmappingports.AccountingReferenceObjectRepository
	mappings    accountingmappingports.AccountingMappingRepository
	sessions    accountingconnectionports.AccountingSessionProvider
	completion  llmports.CompletionService
	kit         *accountingmappingkit.Kit
	catalog     *accountingmappingtargets.Catalog
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.accounting-mapping"),
		connections: p.Connections,
		references:  p.References,
		mappings:    p.Mappings,
		sessions:    p.Sessions,
		completion:  p.Completion,
		kit:         p.Kit,
		catalog:     p.Catalog,
	}
}

var _ accountingmappingports.AccountingMappingScorer = (*Service)(nil)
