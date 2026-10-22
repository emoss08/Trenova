package aiproviderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/buncolgen/aiprovidercol"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type AdvisorParams struct {
	fx.In

	DB *postgres.Connection
}

type advisorRecorder struct {
	db *postgres.Connection
}

func NewAdvisorRecorder(p AdvisorParams) agentports.AIProviderAdvisorRecorder {
	return &advisorRecorder{db: p.DB}
}

// RecordAdvisorRefusal keeps the refusal only while the provider still
// declares the advisor the refused call carried, for the model it carried.
// It leaves the version alone: the refusal is the API's word, not an edit,
// and an administrator's open editor must still save.
func (r *advisorRecorder) RecordAdvisorRefusal(
	ctx context.Context,
	req *agentports.RecordAdvisorRefusalRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := aiprovidercol.ProviderColumns
		refusal := req.Refusal

		if _, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*aiprovider.Provider)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return aiprovidercol.ProviderScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.AdvisorEnabled.Eq(), true).
					Where(cols.Model.Eq(), refusal.Model)
			}).
			Set(cols.AdvisorRefusal.Set(), &refusal).
			Exec(ctx); err != nil {
			return fmt.Errorf("record ai provider advisor refusal: %w", err)
		}

		return nil
	})
}
