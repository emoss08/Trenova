package accountingmappingservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	maxModelTargets        = 100
	maxRefreshErrorLength  = 1000
	maxReferencePagesGuard = 500
)

var errReferenceUnsupported = errors.New("this accounting system cannot share its reference data")

func (s *Service) PullReference(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	kind accountingsync.ReferenceKind,
) (*services.AccountingReferencePull, error) {
	if !kind.IsValid() {
		return nil, fmt.Errorf("unknown reference kind %q", kind)
	}
	session, err := s.connectionService.Session(ctx, tenantInfo, connectionID)
	if err != nil {
		return nil, err
	}
	reader, ok := session.Connector.(services.AccountingReferenceReader)
	if !ok {
		return nil, errReferenceUnsupported
	}

	seenAt := timeutils.NowUnix()
	pull := &services.AccountingReferencePull{Kind: kind}
	start := 1
	for pages := 0; start > 0; pages++ {
		if pages >= maxReferencePagesGuard {
			return nil, fmt.Errorf("stopped after %d pages of %s records", pages, kind)
		}
		page, listErr := reader.ListReference(ctx, &services.AccountingReferencePageRequest{
			RealmID:       session.Connection.ExternalRealmID,
			AccessToken:   session.AccessToken,
			Kind:          kind,
			StartPosition: start,
			PageSize:      reader.MaxReferencePageSize(),
		})
		if listErr != nil {
			s.connectionService.ReportCallFailure(ctx, tenantInfo, connectionID, listErr)
			return nil, listErr
		}
		if err = s.references.Upsert(ctx, &repositories.UpsertAccountingReferenceObjectsRequest{
			TenantInfo:   tenantInfo,
			ConnectionID: connectionID,
			Objects:      page.Objects,
			SeenAt:       seenAt,
		}); err != nil {
			return nil, err
		}
		pull.Fetched += len(page.Objects)
		start = page.NextStart
	}

	removed, err := s.references.MarkRemovedUnseen(
		ctx,
		&repositories.MarkAccountingReferenceRemovedRequest{
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
) (*services.AccountingRescoreResult, error) {
	targets, err := s.listTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	result := &services.AccountingRescoreResult{Targets: len(targets)}

	fresh := make([]*accountingsync.AccountingMapping, 0, len(targets))
	for _, t := range targets {
		fresh = append(fresh, &accountingsync.AccountingMapping{
			OrganizationID:  tenantInfo.OrgID,
			BusinessUnitID:  tenantInfo.BuID,
			ConnectionID:    connectionID,
			TargetType:      t.TargetType,
			TrenovaObjectID: t.ObjectID,
			TrenovaKey:      t.Key,
			TargetLabel:     t.Label,
			ProviderKind:    t.TargetType.ProviderKind(),
			State:           accountingsync.MappingStateUnmatched,
			Signals:         accountingsync.MappingSignals{},
		})
	}
	if result.Created, err = s.mappings.CreateMissing(ctx, fresh); err != nil {
		return nil, err
	}

	rows, err := s.mappings.ListByConnection(ctx, &repositories.ListAccountingMappingsRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: connectionID,
	})
	if err != nil {
		return nil, err
	}

	refs, err := s.loadReferences(ctx, tenantInfo, connectionID)
	if err != nil {
		return nil, err
	}

	byIdentity := make(map[string]*target, len(targets))
	for _, t := range targets {
		byIdentity[t.identity()] = t
	}

	changed := make([]*accountingsync.AccountingMapping, 0, len(rows))
	for _, row := range rows {
		t, ok := byIdentity[mappingIdentity(row)]
		if !ok {
			continue
		}
		if refs.rescore(row, t) {
			changed = append(changed, row)
		}
		tallyRow(result, row)
	}

	if result.Updated, err = s.mappings.ApplyScoring(ctx, changed); err != nil {
		return nil, err
	}
	return result, nil
}

func tallyRow(result *services.AccountingRescoreResult, row *accountingsync.AccountingMapping) {
	if row.State == accountingsync.MappingStateProposed {
		result.Proposed++
	}
	if row.Rescorable() && row.State == accountingsync.MappingStateUnmatched &&
		len(row.Signals.Candidates) > 0 && len(result.NeedsModel) < maxModelTargets {
		result.NeedsModel = append(result.NeedsModel, row.ID)
	}
}

type referenceSet struct {
	byKind  map[accountingsync.ReferenceKind][]*accountingsync.AccountingReferenceObject
	indexes map[accountingsync.ReferenceKind]*tokenIndex
}

func (s *Service) loadReferences(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*referenceSet, error) {
	set := &referenceSet{
		byKind:  make(map[accountingsync.ReferenceKind][]*accountingsync.AccountingReferenceObject),
		indexes: make(map[accountingsync.ReferenceKind]*tokenIndex),
	}
	for _, kind := range accountingsync.AllReferenceKinds() {
		refs, err := s.references.ListByKind(
			ctx,
			&repositories.ListAccountingReferenceObjectsRequest{
				TenantInfo:   tenantInfo,
				ConnectionID: connectionID,
				Kind:         kind,
			},
		)
		if err != nil {
			return nil, err
		}
		set.byKind[kind] = refs
		if kind == accountingsync.ReferenceKindCustomer ||
			kind == accountingsync.ReferenceKindVendor {
			set.indexes[kind] = newTokenIndex(refs)
		}
	}
	return set, nil
}

func (r *referenceSet) rescore(row *accountingsync.AccountingMapping, t *target) bool {
	labelChanged := row.TargetLabel != t.Label
	row.TargetLabel = t.Label
	if !row.Rescorable() {
		return labelChanged
	}
	kind := row.TargetType.ProviderKind()
	return row.ApplyProposal(score(t, r.byKind[kind], r.indexes[kind])) || labelChanged
}

func (s *Service) MarkRefreshStarted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	now := timeutils.NowUnix()
	return s.connections.MarkReferenceRefresh(
		ctx,
		repositories.MarkAccountingReferenceRefreshRequest{
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
	req := repositories.MarkAccountingReferenceRefreshRequest{
		TenantInfo: tenantInfo,
		ID:         connectionID,
		Error:      stringutils.TruncateRunes(failure, maxRefreshErrorLength),
	}
	if failure == "" {
		now := timeutils.NowUnix()
		req.RefreshedAt = &now
	}
	if err := s.connections.MarkReferenceRefresh(ctx, req); err != nil {
		return err
	}

	conn, err := s.connections.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         connectionID,
	})
	if err != nil {
		s.l.Warn("could not reload the connection after a refresh", zap.Error(err))
		return nil
	}
	s.publishConnectionInvalidation(ctx, conn, pulid.Nil)
	return nil
}
