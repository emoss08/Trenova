package airetrievalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	settingsEntity = "AIRetrievalSettings"

	defaultPurgeBatch = 1000
	maxPurgeBatch     = 10000
)

func (r *repository) findSettings(
	ctx context.Context,
	dba bun.IDB,
	tenantInfo pagination.TenantInfo,
	forUpdate bool,
) (*airetrieval.Settings, bool, error) {
	entity := new(airetrieval.Settings)
	q := dba.NewSelect().
		Model(entity).
		Apply(buncolgen.SettingsApplyTenant(tenantInfo)).
		Limit(1)
	if forUpdate {
		q = q.For("UPDATE")
	}

	if err := q.Scan(ctx); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read retrieval settings: %w", err)
	}

	return entity, true, nil
}

func (r *repository) GetSettings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*airetrieval.Settings, error) {
	if err := validateTenant(tenantInfo); err != nil {
		return nil, err
	}

	entity, found, err := r.findSettings(ctx, r.db.DBForContext(ctx), tenantInfo, false)
	switch {
	case err != nil:
		return nil, err
	case !found:
		return airetrieval.DefaultSettings(tenantInfo.OrgID, tenantInfo.BuID), nil
	default:
		return entity, nil
	}
}

func (r *repository) UpdateSettings(
	ctx context.Context,
	entity *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	if entity == nil {
		return nil, invalid("settings are required")
	}

	tenantInfo := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	if err := validateTenant(tenantInfo); err != nil {
		return nil, err
	}

	dba := r.db.DBForContext(ctx)
	existing, found, err := r.findSettings(ctx, dba, tenantInfo, false)
	if err != nil {
		return nil, err
	}

	if !found {
		return r.insertSettings(ctx, dba, entity)
	}

	return r.updateSettings(ctx, dba, entity, existing)
}

func (r *repository) insertSettings(
	ctx context.Context,
	dba bun.IDB,
	entity *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	entity.ID = pulid.Nil
	entity.Version = 0

	if _, err := dba.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError(settingsEntity, "")
		}
		r.l.Error("failed to create retrieval settings", zap.Error(err))

		return nil, fmt.Errorf("create retrieval settings: %w", err)
	}

	return entity, nil
}

func (r *repository) updateSettings(
	ctx context.Context,
	dba bun.IDB,
	entity *airetrieval.Settings,
	existing *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	cols := buncolgen.SettingsColumns
	ov := entity.Version
	entity.ID = existing.ID
	entity.CreatedAt = existing.CreatedAt
	entity.Version = ov + 1
	entity.UpdatedAt = timeutils.NowUnix()

	res, err := dba.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.SettingsScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), existing.ID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.MemoryEnabled.Set(), entity.MemoryEnabled).
		Set(cols.DocumentsEnabled.Set(), entity.DocumentsEnabled).
		Set(cols.InboundMessagesEnabled.Set(), entity.InboundMessagesEnabled).
		Set(cols.MonthlyIndexingBudgetUSD.Set(), entity.MonthlyIndexingBudgetUSD).
		Set(cols.Paused.Set(), entity.Paused).
		Set(cols.PausedReason.Set(), bun.NullZero(entity.PausedReason)).
		Set(cols.PausedAt.Set(), entity.PausedAt).
		Set(cols.ActiveModelKey.Set(), bun.NullZero(entity.ActiveModelKey)).
		Set(cols.Dimensions.Set(), bun.NullZero(entity.Dimensions)).
		Set(cols.PendingModelKey.Set(), bun.NullZero(entity.PendingModelKey)).
		Set(cols.PendingDimensions.Set(), bun.NullZero(entity.PendingDimensions)).
		Set(cols.UpdatedAt.Set(), entity.UpdatedAt).
		Set(cols.Version.Set(), entity.Version).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update retrieval settings: %w", err)
	}

	rows, err := rowsAffected(res)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, dberror.CreateVersionMismatchError(settingsEntity, existing.ID.String())
	}

	return entity, nil
}

func (r *repository) SetPaused(
	ctx context.Context,
	req repositories.SetAIRetrievalPausedRequest,
) (*airetrieval.Settings, error) {
	if err := validateTenant(req.TenantInfo); err != nil {
		return nil, err
	}
	if req.Paused && !req.Reason.IsValid() {
		return nil, invalid("pausing indexing needs a reason")
	}

	now := req.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	var updated *airetrieval.Settings
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		current, found, err := r.findSettings(ctx, tx, req.TenantInfo, true)
		if err != nil {
			return err
		}
		if !found {
			current = airetrieval.DefaultSettings(req.TenantInfo.OrgID, req.TenantInfo.BuID)
		}

		if current.ID.IsNil() && !req.Paused {
			updated = current
			return nil
		}

		applyPause(current, req, now)

		if current.ID.IsNil() {
			updated, err = r.insertSettings(ctx, tx, current)
			return err
		}

		updated, err = r.writePause(ctx, tx, current)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func applyPause(
	settings *airetrieval.Settings,
	req repositories.SetAIRetrievalPausedRequest,
	now int64,
) {
	if !req.Paused {
		settings.Paused = false
		settings.PausedReason = ""
		settings.PausedAt = nil
		return
	}

	if !settings.Paused || settings.PausedReason != req.Reason {
		pausedAt := now
		settings.PausedAt = &pausedAt
	}
	settings.Paused = true
	settings.PausedReason = req.Reason
}

func (r *repository) writePause(
	ctx context.Context,
	tx bun.Tx,
	settings *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	cols := buncolgen.SettingsColumns
	settings.Version++
	settings.UpdatedAt = timeutils.NowUnix()

	if _, err := tx.NewUpdate().
		Model(settings).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.SettingsScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: settings.OrganizationID,
				BuID:  settings.BusinessUnitID,
			}).Where(cols.ID.Eq(), settings.ID)
		}).
		Set(cols.Paused.Set(), settings.Paused).
		Set(cols.PausedReason.Set(), bun.NullZero(settings.PausedReason)).
		Set(cols.PausedAt.Set(), settings.PausedAt).
		Set(cols.UpdatedAt.Set(), settings.UpdatedAt).
		Set(cols.Version.Set(), settings.Version).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("pause retrieval indexing: %w", err)
	}

	return settings, nil
}

func (r *repository) SwapModel(
	ctx context.Context,
	req repositories.SwapAIRetrievalModelRequest,
) (*repositories.SwapAIRetrievalModelResult, error) {
	if err := validateTenant(req.TenantInfo); err != nil {
		return nil, err
	}
	if err := validateModelKey(req.PendingModelKey); err != nil {
		return nil, err
	}

	var result *repositories.SwapAIRetrievalModelResult
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		current, found, err := r.findSettings(ctx, tx, req.TenantInfo, true)
		switch {
		case err != nil:
			return err
		case !found || !current.HasPendingModel():
			return airetrieval.ErrNoPendingModel
		case current.PendingModelKey != req.PendingModelKey:
			return fmt.Errorf(
				"%w: %s is pending, not %s",
				airetrieval.ErrPendingModelMoved,
				current.PendingModelKey,
				req.PendingModelKey,
			)
		}

		retired := current.ActiveModelKey
		current.ActiveModelKey = current.PendingModelKey
		current.Dimensions = current.PendingDimensions
		current.PendingModelKey = ""
		current.PendingDimensions = 0
		current.Version++
		current.UpdatedAt = timeutils.NowUnix()

		cols := buncolgen.SettingsColumns
		if _, err = tx.NewUpdate().
			Model(current).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.SettingsScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), current.ID)
			}).
			Set(cols.ActiveModelKey.Set(), current.ActiveModelKey).
			Set(cols.Dimensions.Set(), current.Dimensions).
			Set(cols.PendingModelKey.SetNull()).
			Set(cols.PendingDimensions.SetNull()).
			Set(cols.UpdatedAt.Set(), current.UpdatedAt).
			Set(cols.Version.Set(), current.Version).
			Exec(ctx); err != nil {
			return fmt.Errorf("swap retrieval embedding model: %w", err)
		}

		result = &repositories.SwapAIRetrievalModelResult{
			Settings:        current,
			RetiredModelKey: retired,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repository) PurgeModel(
	ctx context.Context,
	req repositories.PurgeAIRetrievalModelRequest,
) (repositories.PurgeAIRetrievalModelResult, error) {
	var result repositories.PurgeAIRetrievalModelResult

	if err := validateTenant(req.TenantInfo); err != nil {
		return result, err
	}
	if err := validateModelKey(req.ModelKey); err != nil {
		return result, err
	}

	batch := req.BatchSize
	if batch <= 0 {
		batch = defaultPurgeBatch
	}
	batch = intutils.Clamp(batch, 1, maxPurgeBatch)

	vectorReady, err := r.vectorReady(ctx)
	if err != nil {
		return result, err
	}

	err = r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		current, found, txErr := r.findSettings(ctx, tx, req.TenantInfo, true)
		if txErr != nil {
			return txErr
		}
		if found &&
			(current.ActiveModelKey == req.ModelKey || current.PendingModelKey == req.ModelKey) {
			return fmt.Errorf("%w: %s", airetrieval.ErrModelInUse, req.ModelKey)
		}

		if result.IndexEntries, txErr = purgeIndexEntries(ctx, tx, req, batch); txErr != nil {
			return txErr
		}
		if !vectorReady {
			return nil
		}

		result.Embeddings, txErr = purgeEmbeddings(ctx, tx, req, batch)
		return txErr
	})
	if err != nil {
		return repositories.PurgeAIRetrievalModelResult{}, err
	}

	return result, nil
}

func purgeIndexEntries(
	ctx context.Context,
	tx bun.Tx,
	req repositories.PurgeAIRetrievalModelRequest,
	batch int,
) (int, error) {
	cols := buncolgen.IndexEntryColumns
	victims := tx.NewSelect().
		Model((*airetrieval.IndexEntry)(nil)).
		Column(buncolgen.IndexEntryTable.PrimaryKey...).
		Apply(buncolgen.IndexEntryApplyTenant(req.TenantInfo)).
		Where(cols.ModelKey.Eq(), req.ModelKey).
		Limit(batch)

	res, err := tx.NewDelete().
		Model((*airetrieval.IndexEntry)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.IndexEntryScopeTenantDelete(dq, req.TenantInfo).
				Where(indexEntryKeyTuple()+" IN (?)", victims)
		}).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("purge retired model index entries: %w", err)
	}

	return rowsAffected(res)
}

func purgeEmbeddings(
	ctx context.Context,
	tx bun.Tx,
	req repositories.PurgeAIRetrievalModelRequest,
	batch int,
) (int, error) {
	cols := buncolgen.EmbeddingColumns
	victims := tx.NewSelect().
		Model((*airetrieval.Embedding)(nil)).
		Column(buncolgen.EmbeddingTable.PrimaryKey...).
		Apply(buncolgen.EmbeddingApplyTenant(req.TenantInfo)).
		Where(cols.ModelKey.Eq(), req.ModelKey).
		Limit(batch)

	res, err := tx.NewDelete().
		Model((*airetrieval.Embedding)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.EmbeddingScopeTenantDelete(dq, req.TenantInfo).
				Where(embeddingKeyTuple()+" IN (?)", victims)
		}).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("purge retired model embeddings: %w", err)
	}

	return rowsAffected(res)
}

func embeddingKeyTuple() string {
	cols := buncolgen.EmbeddingColumns

	return buncolgen.Expr("({0}, {1}, {2}, {3}, {4}, {5})",
		cols.OrganizationID,
		cols.BusinessUnitID,
		cols.SourceType,
		cols.SourceID,
		cols.ChunkIndex,
		cols.ModelKey,
	)
}
