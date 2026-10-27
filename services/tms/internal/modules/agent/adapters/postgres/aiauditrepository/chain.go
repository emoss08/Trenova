package aiauditrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit/aiauditdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	pruneSetting      = "trenova.ai_audit_prune"
	defaultPruneBatch = 5000
)

// GetChainHead reads a tenant's chain head, or an empty one for a tenant
// whose trail has not started.
func (r *repository) GetChainHead(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*aiaudit.AIAuditChainHead, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aiaudit.AIAuditChainHead, error) {
		head, err := aiauditdb.AIAuditChainHeads.Query().One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return &aiaudit.AIAuditChainHead{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
			}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read AI audit chain head: %w", err)
		}

		return head, nil
	})
}

func (r *repository) ListChainHeads(ctx context.Context) ([]*aiaudit.AIAuditChainHead, error) {
	ctx = dbscope.WithSystem(ctx, "list every organization's AI audit chain head")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiaudit.AIAuditChainHead, error) {
		h := aiauditdb.AIAuditChainHeads
		heads, err := h.AllTenants().
			Where(h.LastSeq.Gt(0)).
			Order(h.OrganizationID.Asc(), h.BusinessUnitID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list AI audit chain heads: %w", err)
		}

		return heads, nil
	})
}

// RecordVerification writes what a check of the chain found. It touches only
// the verification columns, so it never races the projector's advance of the
// head it shares a row with.
func (r *repository) RecordVerification(
	ctx context.Context,
	req *aiauditports.RecordAIAuditVerificationRequest,
) (*aiaudit.AIAuditChainHead, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aiaudit.AIAuditChainHead, error) {
		h := aiauditdb.AIAuditChainHeads
		sets := []keel.Assignment{
			h.LastVerificationStatus.To(req.Status),
			h.LastVerifiedAt.To(req.At),
			h.LastVerificationDetail.To(req.Detail),
			h.UpdatedAt.To(req.At),
			h.Version.ToExpr(h.Version),
		}
		if req.Status == aiaudit.VerificationVerified {
			sets = append(sets,
				h.LastVerifiedSeq.To(req.VerifiedSeq),
				h.LastVerificationFailedSeq.ToNull(),
			)
		} else if req.FailedSeq != nil {
			sets = append(sets, h.LastVerificationFailedSeq.To(*req.FailedSeq))
		} else {
			sets = append(sets, h.LastVerificationFailedSeq.ToNull())
		}

		heads, err := h.Query().UpdateReturning(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return nil, fmt.Errorf("record AI audit verification: %w", err)
		}
		if len(heads) == 0 {
			return r.GetChainHead(ctx, req.TenantInfo)
		}

		return heads[0], nil
	})
}

func (r *repository) GetSealEndingAt(
	ctx context.Context,
	_ pagination.TenantInfo,
	toSeq int64,
) (*aiaudit.AIAuditSeal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aiaudit.AIAuditSeal, error) {
		s := aiauditdb.AIAuditSeals
		seal, err := s.Where(s.ToSeq.Eq(toSeq)).One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, nil //nolint:nilnil // no seal ends there
		}
		if err != nil {
			return nil, fmt.Errorf("read AI audit seal: %w", err)
		}

		return seal, nil
	})
}

func (r *repository) ListSeals(
	ctx context.Context,
	req aiauditports.ListAIAuditSealsRequest,
) ([]*aiaudit.AIAuditSeal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiaudit.AIAuditSeal, error) {
		s := aiauditdb.AIAuditSeals
		seals, err := s.Where(s.ToSeq.Gte(req.FromSeq)).
			Order(s.ToSeq.Asc()).
			Limit(req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list AI audit seals: %w", err)
		}

		return seals, nil
	})
}

// LastSealBefore is the newest seal written before a time, which is where a
// retention sweep may cut the chain: pruning at a seal's end leaves the next
// row's prev_hash checkable against the seal.
func (r *repository) LastSealBefore(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sealedBefore int64,
) (*aiaudit.AIAuditSeal, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"read an organization's last AI audit seal during the retention sweep",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aiaudit.AIAuditSeal, error) {
		s := aiauditdb.AIAuditSeals
		seal, err := s.AllTenants().
			Where(
				s.OrganizationID.Eq(tenantInfo.OrgID),
				s.BusinessUnitID.Eq(tenantInfo.BuID),
				s.SealedAt.Lt(sealedBefore),
			).
			Order(s.ToSeq.Desc()).
			First(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, nil //nolint:nilnil // nothing is old enough
		}
		if err != nil {
			return nil, fmt.Errorf("read the last AI audit seal before a time: %w", err)
		}

		return seal, nil
	})
}

// Prune removes a tenant's chain up to a seq, oldest first, a batch per
// transaction, each under the setting the append-only trigger admits.
func (r *repository) Prune(
	ctx context.Context,
	req aiauditports.PruneAIAuditEventsRequest,
) (int, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"prune AI audit events past retention behind the projector horizon",
	)
	batch := req.BatchSize
	if batch <= 0 {
		batch = defaultPruneBatch
	}

	admitPrune := kernelports.TxOptions{
		Settings: []kernelports.TxSetting{{Name: pruneSetting, Value: "on"}},
	}
	total := 0
	for {
		var deleted int
		err := r.db.Transact(
			ctx,
			admitPrune,
			func(ctx context.Context) error {
				var batchErr error
				deleted, batchErr = r.pruneBatch(ctx, req.TenantInfo, req.ThroughSeq, batch)
				return batchErr
			},
		)
		if err != nil {
			r.l.Error("failed to prune AI audit events",
				zap.String("organizationId", req.TenantInfo.OrgID.String()),
				zap.Error(err),
			)

			return total, err
		}

		total += deleted
		if deleted < batch {
			return total, nil
		}
		if err = ctx.Err(); err != nil {
			return total, err
		}
	}
}

func (r *repository) pruneBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	throughSeq int64,
	batch int,
) (int, error) {
	db := r.db.Keel(ctx)
	e := aiauditdb.AIAuditEvents
	doomed := aiauditdb.AIAuditEvents.As("doomed")
	deleted, err := e.AllTenants().
		Where(
			e.OrganizationID.Eq(tenantInfo.OrgID),
			e.BusinessUnitID.Eq(tenantInfo.BuID),
			e.ID.InQuery(doomed.AllTenants().
				Where(
					doomed.OrganizationID.Eq(tenantInfo.OrgID),
					doomed.BusinessUnitID.Eq(tenantInfo.BuID),
					doomed.Seq.Lte(throughSeq),
				).
				Order(doomed.Seq.Asc()).
				Limit(batch).
				Pick(doomed.ID)),
		).
		Delete(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("prune AI audit events: %w", err)
	}

	return int(deleted), nil
}

func (r *repository) GetWatermarks(
	ctx context.Context,
) (map[aiaudit.Source]*aiaudit.AIAuditProjectorState, error) {
	ctx = dbscope.WithSystem(ctx, "read the AI audit projector's watermarks")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[aiaudit.Source]*aiaudit.AIAuditProjectorState, error) {
			states, err := aiauditdb.AIAuditProjectorStates.Query().All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("read AI audit projector watermarks: %w", err)
			}

			byName := make(map[aiaudit.Source]*aiaudit.AIAuditProjectorState, len(states))
			for _, state := range states {
				byName[state.Source] = state
			}

			return byName, nil
		},
	)
}

func (r *repository) SaveWatermark(
	ctx context.Context,
	state *aiaudit.AIAuditProjectorState,
) error {
	ctx = dbscope.WithSystem(ctx, "advance the AI audit projector's watermark")
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := aiauditdb.AIAuditProjectorStates
		if _, err := p.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*aiaudit.AIAuditProjectorState{state},
			keel.OnConflict(p.Source).Update(p.WatermarkTS, p.WatermarkID, p.UpdatedAt),
		); err != nil {
			return fmt.Errorf("save AI audit projector watermark: %w", err)
		}

		return nil
	})
}
