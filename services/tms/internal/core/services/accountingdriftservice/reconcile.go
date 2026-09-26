package accountingdriftservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type compareTally struct {
	compared int
	skipped  int
	opened   []*accountingsync.AccountingDriftFinding
	updated  int
	resolved int
}

func (s *Service) ReconcileBatch(
	ctx context.Context,
	req *services.ReconcileAccountingDriftRequest,
) (*services.AccountingDriftBatchResult, error) {
	result := &services.AccountingDriftBatchResult{LastID: req.AfterID}
	sess, held, err := s.startRead(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil || held {
		result.Held = held
		return result, err
	}

	datedFrom, err := s.source.ScopeStart(ctx, sess.tenant)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultBatch
	}
	limit = intutils.Clamp(limit, 1, maxBatch)
	records, err := s.source.ListRecords(ctx, &repositories.ListAccountingDriftRecordsRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		DatedFrom:    datedFrom,
		AfterID:      req.AfterID,
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}
	if len(records) > 0 {
		result.LastID = records[len(records)-1].ID
	}
	result.More = len(records) == limit

	tally, err := s.compareRecords(ctx, sess, records)
	if err != nil {
		return nil, err
	}
	s.finishTally(ctx, sess, tally, req.EventBudget, result)
	return result, nil
}

func (s *Service) RecheckDocuments(
	ctx context.Context,
	req *services.RecheckAccountingDriftRequest,
) (*services.AccountingDriftBatchResult, error) {
	result := new(services.AccountingDriftBatchResult)
	if len(req.Documents) == 0 {
		return result, nil
	}
	sess, held, err := s.startRead(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil || held {
		result.Held = held
		return result, err
	}

	records, err := s.changedRecords(ctx, sess, req.Documents)
	if err != nil {
		return nil, err
	}
	tally, err := s.compareRecords(ctx, sess, records)
	if err != nil {
		return nil, err
	}
	s.finishTally(ctx, sess, tally, req.EventBudget, result)
	return result, nil
}

func (s *Service) startRead(
	ctx context.Context,
	tenant pagination.TenantInfo,
	connectionID pulid.ID,
) (*readSession, bool, error) {
	conn, err := s.connectionByID(ctx, tenant, connectionID)
	if err != nil {
		return nil, false, err
	}
	if !conn.ChecksDrift() {
		return nil, true, nil
	}
	sess, err := s.openRead(ctx, conn)
	if err != nil {
		s.l.Warn("drift checks are holding: the connection cannot be used",
			zap.String("connectionId", conn.ID.String()), zap.Error(err))
		s.recordFailure(ctx, tenant, conn.ID, s.classify(nil, err))
		return nil, true, nil
	}
	return sess, false, nil
}

func (s *Service) finishTally(
	ctx context.Context,
	sess *readSession,
	tally *compareTally,
	eventBudget int,
	result *services.AccountingDriftBatchResult,
) {
	result.Compared = tally.compared
	result.Skipped = tally.skipped
	result.Opened = len(tally.opened)
	result.Updated = tally.updated
	result.Resolved = tally.resolved
	result.Events = s.announce(ctx, sess.tenant, tally.opened, eventBudget)
	if result.Opened+result.Resolved+result.Updated > 0 {
		s.publishInvalidation(ctx, sess.tenant, pulid.Nil, sess.conn.ID)
	}
}

func (s *Service) changedRecords(
	ctx context.Context,
	sess *readSession,
	documents []services.AccountingChangedDocument,
) ([]*accountingsync.AccountingSyncRecord, error) {
	ids := make([]string, 0, len(documents))
	types := make([]accountingsync.SyncObjectType, 0, len(accountingsync.DriftObjectTypes()))
	seenType := make(map[accountingsync.SyncObjectType]struct{}, cap(types))
	for idx := range documents {
		if documents[idx].ExternalID == "" {
			continue
		}
		ids = append(ids, documents[idx].ExternalID)
		for _, objectType := range documents[idx].ObjectTypes {
			if _, seen := seenType[objectType]; seen {
				continue
			}
			seenType[objectType] = struct{}{}
			types = append(types, objectType)
		}
	}
	if len(ids) == 0 || len(types) == 0 {
		return nil, nil
	}
	found, err := s.records.ListByExternalIDs(
		ctx,
		&repositories.ListAccountingSyncRecordsByExternalIDsRequest{
			TenantInfo:   sess.tenant,
			ConnectionID: sess.conn.ID,
			ObjectTypes:  types,
			ExternalIDs:  ids,
		},
	)
	if err != nil || len(found) == 0 {
		return nil, err
	}
	all, err := s.recordsOf(ctx, sess, found)
	if err != nil {
		return nil, err
	}
	return latestDocumentRecords(all), nil
}

func (s *Service) compareRecords(
	ctx context.Context,
	sess *readSession,
	records []*accountingsync.AccountingSyncRecord,
) (*compareTally, error) {
	tally := new(compareTally)
	if len(records) == 0 {
		return tally, nil
	}
	busy, err := s.busyObjects(ctx, sess, records)
	if err != nil {
		return nil, err
	}

	comparisons := make([]*comparison, 0, len(records))
	for objectType, group := range groupByType(records) {
		ready := make([]*accountingsync.AccountingSyncRecord, 0, len(group))
		for _, record := range group {
			if _, waiting := busy[record.ObjectID]; waiting {
				tally.skipped++
				continue
			}
			ready = append(ready, record)
		}
		compared, compareErr := s.compareType(ctx, sess, objectType, ready)
		if compareErr != nil {
			return nil, compareErr
		}
		tally.skipped += len(ready) - len(compared)
		comparisons = append(comparisons, compared...)
	}
	if len(comparisons) == 0 {
		return tally, nil
	}

	at := s.now().Unix()
	base := &accountingsync.DriftObservation{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		At:           at,
	}
	observed := make(map[pulid.ID]*accountingsync.DriftObservation, len(comparisons))
	objectIDs := make([]pulid.ID, 0, len(comparisons))
	for _, c := range comparisons {
		objectIDs = append(objectIDs, c.record.ObjectID)
		obs := c.observation(base)
		if obs != nil && sess.writer != nil {
			obs.ExternalURL = sess.writer.DocumentURL(obs.ObjectType, obs.ExternalID)
		}
		observed[c.record.ObjectID] = obs
	}
	tally.compared = len(comparisons)

	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		return s.applyObservations(txCtx, sess, objectIDs, observed, at, tally)
	})
	if err != nil {
		return nil, err
	}
	return tally, nil
}

func (s *Service) applyObservations(
	ctx context.Context,
	sess *readSession,
	objectIDs []pulid.ID,
	observed map[pulid.ID]*accountingsync.DriftObservation,
	at int64,
	tally *compareTally,
) error {
	open, err := s.findings.ListOpen(ctx, &repositories.ListOpenAccountingDriftFindingsRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		ObjectIDs:    objectIDs,
	})
	if err != nil {
		return err
	}
	matched := make(map[pulid.ID]bool, len(open))
	for _, finding := range open {
		obs := observed[finding.ObjectID]
		if obs != nil && obs.Kind == finding.Kind {
			matched[finding.ObjectID] = true
			finding.Observe(obs)
			if _, err = s.findings.Update(ctx, finding); err != nil {
				return err
			}
			tally.updated++
			continue
		}
		if !finding.Clear(at) {
			continue
		}
		if _, err = s.findings.Update(ctx, finding); err != nil {
			return err
		}
		tally.resolved++
	}
	for _, objectID := range objectIDs {
		obs := observed[objectID]
		if obs == nil || matched[objectID] {
			continue
		}
		created, createErr := s.findings.Create(ctx, accountingsync.NewAccountingDriftFinding(obs))
		if createErr != nil {
			return createErr
		}
		tally.opened = append(tally.opened, created)
	}
	return nil
}

func (s *Service) recordsOf(
	ctx context.Context,
	sess *readSession,
	records []*accountingsync.AccountingSyncRecord,
) ([]*accountingsync.AccountingSyncRecord, error) {
	types := make([]accountingsync.SyncObjectType, 0, len(accountingsync.DriftObjectTypes()))
	seen := make(map[accountingsync.SyncObjectType]struct{}, cap(types))
	ids := make([]pulid.ID, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ObjectID)
		if _, ok := seen[record.ObjectType]; !ok {
			seen[record.ObjectType] = struct{}{}
			types = append(types, record.ObjectType)
		}
	}
	return s.records.ListByObjects(ctx, &repositories.ListAccountingSyncRecordsByObjectsRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		ObjectTypes:  types,
		ObjectIDs:    ids,
	})
}

func (s *Service) busyObjects(
	ctx context.Context,
	sess *readSession,
	records []*accountingsync.AccountingSyncRecord,
) (map[pulid.ID]struct{}, error) {
	all, err := s.recordsOf(ctx, sess, records)
	if err != nil {
		return nil, err
	}
	busy := make(map[pulid.ID]struct{}, len(all))
	for _, record := range all {
		if !record.Status.IsFinal() {
			busy[record.ObjectID] = struct{}{}
		}
	}
	return busy, nil
}

func (s *Service) compareType(
	ctx context.Context,
	sess *readSession,
	objectType accountingsync.SyncObjectType,
	records []*accountingsync.AccountingSyncRecord,
) ([]*comparison, error) {
	if len(records) == 0 {
		return nil, nil
	}
	ids := make([]pulid.ID, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ObjectID)
	}
	states, err := s.source.ListStates(ctx, &repositories.ListAccountingDriftStatesRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		ObjectType:   objectType,
		ObjectIDs:    ids,
	})
	if err != nil {
		return nil, err
	}
	byObject := make(map[pulid.ID]*repositories.AccountingDriftState, len(states))
	for _, state := range states {
		byObject[state.ObjectID] = state
	}

	provider, err := s.readProvider(ctx, sess, objectType, records)
	if err != nil {
		return nil, err
	}

	out := make([]*comparison, 0, len(records))
	for _, record := range records {
		trenova := byObject[record.ObjectID]
		state, read := provider[record.ExternalID]
		if trenova == nil || !read {
			continue
		}
		out = append(out, &comparison{record: record, trenova: trenova, provider: state})
	}
	return out, nil
}

func (s *Service) readProvider(
	ctx context.Context,
	sess *readSession,
	objectType accountingsync.SyncObjectType,
	records []*accountingsync.AccountingSyncRecord,
) (map[string]*services.AccountingDocumentState, error) {
	out := make(map[string]*services.AccountingDocumentState, len(records))
	for start := 0; start < len(records); start += sess.limit {
		chunk := records[start:min(start+sess.limit, len(records))]
		targets := make([]services.AccountingDocumentTarget, 0, len(chunk))
		for _, record := range chunk {
			targets = append(targets, services.AccountingDocumentTarget{
				ExternalID: record.ExternalID,
				Refs:       record.ExternalRefs,
			})
		}
		states, err := sess.reader.ReadDocuments(ctx, &services.ReadAccountingDocumentsRequest{
			Auth:    sess.auth,
			Kind:    objectType,
			Targets: targets,
		})
		if err != nil {
			s.recordFailure(ctx, sess.tenant, sess.conn.ID, s.classify(sess, err))
			return nil, err
		}
		for _, state := range states {
			if state != nil {
				out[state.ExternalID] = state
			}
		}
	}
	return out, nil
}

func groupByType(
	records []*accountingsync.AccountingSyncRecord,
) map[accountingsync.SyncObjectType][]*accountingsync.AccountingSyncRecord {
	out := make(map[accountingsync.SyncObjectType][]*accountingsync.AccountingSyncRecord, 4)
	for _, record := range records {
		out[record.ObjectType] = append(out[record.ObjectType], record)
	}
	return out
}

func latestDocumentRecords(
	records []*accountingsync.AccountingSyncRecord,
) []*accountingsync.AccountingSyncRecord {
	latest := make(map[pulid.ID]*accountingsync.AccountingSyncRecord, len(records))
	order := make([]pulid.ID, 0, len(records))
	for _, record := range records {
		if record.Status != accountingsync.SyncStatusSynced || record.ExternalID == "" ||
			!slices.Contains(accountingsync.DriftCreateOperations(), record.Operation) {
			continue
		}
		current, seen := latest[record.ObjectID]
		if !seen {
			order = append(order, record.ObjectID)
		}
		if current == nil || record.Revision > current.Revision {
			latest[record.ObjectID] = record
		}
	}
	out := make([]*accountingsync.AccountingSyncRecord, 0, len(order))
	for _, objectID := range order {
		out = append(out, latest[objectID])
	}
	return out
}
