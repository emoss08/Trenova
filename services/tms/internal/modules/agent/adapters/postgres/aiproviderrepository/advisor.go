package aiproviderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"go.uber.org/fx"
)

type AdvisorParams struct {
	fx.In

	DB *postgres.Connection
}

type advisorRecorder struct {
	db *postgres.Connection
}

func NewAdvisorRecorder(p AdvisorParams) aiproviderports.AIProviderAdvisorRecorder {
	return &advisorRecorder{db: p.DB}
}

// RecordAdvisorRefusal keeps the refusal only while the provider still
// declares the advisor the refused call carried, for the model it carried.
// It leaves the version alone: the refusal is the API's word, not an edit,
// and an administrator's open editor must still save.
func (r *advisorRecorder) RecordAdvisorRefusal(
	ctx context.Context,
	req *aiproviderports.RecordAdvisorRefusalRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := aiproviderdb.Providers
		refusal := req.Refusal

		if _, err := p.Where(
			p.ID.Eq(req.ID),
			p.AdvisorEnabled.IsTrue(),
			p.Model.Eq(refusal.Model),
		).Update(ctx, r.db.Keel(ctx), append(untouched(), p.AdvisorRefusal.To(&refusal))...); err != nil {
			return fmt.Errorf("record ai provider advisor refusal: %w", err)
		}

		return nil
	})
}
