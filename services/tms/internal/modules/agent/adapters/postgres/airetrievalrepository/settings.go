package airetrievalrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval/airetrievaldb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

var settingsErrors = dberror.Mapping{Entity: "AIRetrievalSettings"}

const (
	settingsEntity = "AIRetrievalSettings"

	defaultPurgeBatch = 1000
	maxPurgeBatch     = 10000

	maxIndexedTenantPage = 1000
)

func findSettings(
	ctx context.Context,
	db *keel.DB,
	forUpdate bool,
) (*airetrieval.Settings, bool, error) {
	q := airetrievaldb.Settingses.Query()
	if forUpdate {
		q.Lock(keel.ForUpdate)
	}

	entity, err := q.First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read retrieval settings: %w", err)
	}

	return entity, true, nil
}

func (r *repository) GetSettings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*airetrieval.Settings, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*airetrieval.Settings, error) {
		if err := validateTenant(tenantInfo); err != nil {
			return nil, err
		}

		entity, found, err := findSettings(ctx, r.db.Keel(ctx), false)
		switch {
		case err != nil:
			return nil, err
		case !found:
			return airetrieval.DefaultSettings(tenantInfo.OrgID, tenantInfo.BuID), nil
		default:
			return entity, nil
		}
	})
}

func (r *repository) UpdateSettings(
	ctx context.Context,
	entity *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*airetrieval.Settings, error) {
		if entity == nil {
			return nil, invalid("settings are required")
		}

		tenantInfo := pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		}
		if err := validateTenant(tenantInfo); err != nil {
			return nil, err
		}

		db := r.db.Keel(ctx)
		existing, found, err := findSettings(ctx, db, false)
		if err != nil {
			return nil, err
		}

		if !found {
			return r.insertSettings(ctx, db, entity)
		}

		return updateSettings(ctx, db, entity, existing)
	})
}

func (r *repository) insertSettings(
	ctx context.Context,
	db *keel.DB,
	entity *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	entity.ID = pulid.Nil
	entity.Version = 0

	if err := airetrievaldb.Settingses.Create(ctx, db, entity); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError(settingsEntity, "")
		}
		r.l.Error("failed to create retrieval settings", zap.Error(err))

		return nil, fmt.Errorf("create retrieval settings: %w", err)
	}

	return entity, nil
}

func updateSettings(
	ctx context.Context,
	db *keel.DB,
	entity *airetrieval.Settings,
	existing *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	s := airetrievaldb.Settingses
	entity.ID = existing.ID
	entity.CreatedAt = existing.CreatedAt

	if err := s.Save(ctx, db, entity,
		s.MemoryEnabled,
		s.DocumentsEnabled,
		s.InboundMessagesEnabled,
		s.MonthlyIndexingBudgetUSD,
		s.Paused,
		s.PausedReason,
		s.PausedAt,
		s.ActiveModelKey,
		s.Dimensions,
		s.PendingModelKey,
		s.PendingDimensions,
	); err != nil {
		return nil, settingsErrors.Map(err, existing.ID.String())
	}

	return entity, nil
}

func (r *repository) SetPaused(
	ctx context.Context,
	req *agentports.SetAIRetrievalPausedRequest,
) (*airetrieval.Settings, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*airetrieval.Settings, error) {
		if req == nil {
			return nil, invalid("pausing indexing needs a request")
		}
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
		err := r.db.Transact(
			ctx,
			kernelports.TxOptions{},
			func(ctx context.Context) error {
				db := r.db.Keel(ctx)
				current, found, err := findSettings(ctx, db, true)
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
					updated, err = r.insertSettings(ctx, db, current)
					return err
				}

				updated, err = writePause(ctx, db, current)
				return err
			},
		)
		if err != nil {
			return nil, err
		}

		return updated, nil
	})
}

func applyPause(
	settings *airetrieval.Settings,
	req *agentports.SetAIRetrievalPausedRequest,
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

func writePause(
	ctx context.Context,
	db *keel.DB,
	settings *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	s := airetrievaldb.Settingses
	if err := s.Save(ctx, db, settings, s.Paused, s.PausedReason, s.PausedAt); err != nil {
		return nil, fmt.Errorf(
			"pause retrieval indexing: %w",
			settingsErrors.Map(err, settings.ID.String()),
		)
	}

	return settings, nil
}

func (r *repository) SwapModel(
	ctx context.Context,
	req agentports.SwapAIRetrievalModelRequest,
) (*agentports.SwapAIRetrievalModelResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*agentports.SwapAIRetrievalModelResult, error) {
			if err := validateTenant(req.TenantInfo); err != nil {
				return nil, err
			}
			if err := validateModelKey(req.PendingModelKey); err != nil {
				return nil, err
			}

			var result *agentports.SwapAIRetrievalModelResult
			err := r.db.Transact(
				ctx,
				kernelports.TxOptions{},
				func(ctx context.Context) error {
					db := r.db.Keel(ctx)
					current, found, err := findSettings(ctx, db, true)
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

					s := airetrievaldb.Settingses
					if err = s.Save(ctx, db, current,
						s.ActiveModelKey,
						s.Dimensions,
						s.PendingModelKey,
						s.PendingDimensions,
					); err != nil {
						return fmt.Errorf(
							"swap retrieval embedding model: %w",
							settingsErrors.Map(err, current.ID.String()),
						)
					}

					result = &agentports.SwapAIRetrievalModelResult{
						Settings:        current,
						RetiredModelKey: retired,
					}

					return nil
				},
			)
			if err != nil {
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) PurgeModel(
	ctx context.Context,
	req agentports.PurgeAIRetrievalModelRequest,
) (agentports.PurgeAIRetrievalModelResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (agentports.PurgeAIRetrievalModelResult, error) {
			var result agentports.PurgeAIRetrievalModelResult

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

			err = r.db.Transact(
				ctx,
				kernelports.TxOptions{},
				func(ctx context.Context) error {
					db := r.db.Keel(ctx)
					current, found, txErr := findSettings(ctx, db, true)
					if txErr != nil {
						return txErr
					}
					if found &&
						(current.ActiveModelKey == req.ModelKey || current.PendingModelKey == req.ModelKey) {
						return fmt.Errorf("%w: %s", airetrieval.ErrModelInUse, req.ModelKey)
					}

					if result.IndexEntries, txErr = purgeIndexEntries(
						ctx,
						db,
						req,
						batch,
					); txErr != nil {
						return txErr
					}
					if !vectorReady {
						return nil
					}

					result.Embeddings, txErr = purgeEmbeddings(ctx, db, req, batch)
					return txErr
				},
			)
			if err != nil {
				return agentports.PurgeAIRetrievalModelResult{}, err
			}

			return result, nil
		},
	)
}

func purgeIndexEntries(
	ctx context.Context,
	db *keel.DB,
	req agentports.PurgeAIRetrievalModelRequest,
	batch int,
) (int, error) {
	e := airetrievaldb.IndexEntries
	victim := airetrievaldb.IndexEntries.As("victim_entry")
	victims := keel.CTE("victims", []string{colSourceType, colSourceID}, victim.
		Where(victim.ModelKey.Eq(req.ModelKey)).
		Pick(victim.SourceType, victim.SourceID).
		Limit(batch))

	removed, err := e.With(victims).
		Join(victims,
			victims.Col(colSourceType).EqCol(e.SourceType),
			victims.Col(colSourceID).EqCol(e.SourceID),
		).
		Where(e.ModelKey.Eq(req.ModelKey)).
		Delete(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("purge retired model index entries: %w", err)
	}

	return int(removed), nil
}

func purgeEmbeddings(
	ctx context.Context,
	db *keel.DB,
	req agentports.PurgeAIRetrievalModelRequest,
	batch int,
) (int, error) {
	e := airetrievaldb.Embeddings
	victim := airetrievaldb.Embeddings.As("victim_embedding")
	victims := keel.CTE("victims", []string{colSourceType, colSourceID, "chunk_index"}, victim.
		Where(victim.ModelKey.Eq(req.ModelKey)).
		Pick(victim.SourceType, victim.SourceID, victim.ChunkIndex).
		Limit(batch))

	removed, err := e.With(victims).
		Join(victims,
			victims.Col(colSourceType).EqCol(e.SourceType),
			victims.Col(colSourceID).EqCol(e.SourceID),
			victims.Col("chunk_index").EqCol(e.ChunkIndex),
		).
		Where(e.ModelKey.Eq(req.ModelKey)).
		Delete(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("purge retired model embeddings: %w", err)
	}

	return int(removed), nil
}

type indexedTenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r *repository) ListIndexedTenants(
	ctx context.Context,
	req agentports.ListIndexedRetrievalTenantsRequest,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that index sources for retrieval")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		s := airetrievaldb.Settingses
		limit := intutils.Clamp(req.Limit, 1, maxIndexedTenantPage)

		q := s.AllTenants().
			Where(s.ActiveModelKey.Ne("")).
			Pick(s.OrganizationID, s.BusinessUnitID).
			Order(s.OrganizationID.Asc(), s.BusinessUnitID.Asc()).
			Limit(limit)
		if req.After != nil {
			q.Where(keel.Raw(
				"(?, ?) > (?, ?)",
				s.OrganizationID,
				s.BusinessUnitID,
				req.After.OrgID,
				req.After.BuID,
			))
		}

		rows, err := keel.Into[indexedTenantRow](ctx, r.db.Keel(ctx), q)
		if err != nil {
			return nil, fmt.Errorf("list organizations that index sources for retrieval: %w", err)
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}

		return tenants, nil
	})
}
