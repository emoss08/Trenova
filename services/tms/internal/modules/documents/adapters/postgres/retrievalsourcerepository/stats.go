package retrievalsourcerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent/documentcontentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultCharsSample = 500
	maxCharsSample     = 5000
	charsSampleCTE     = "chars_sample"
)

type sourceTable struct {
	ids    func(after pulid.ID, limit int) keel.Projection
	count  func(ctx context.Context, db *keel.DB) (int64, error)
	sample func(limit int) *keel.CTETable
}

var sourceTables = map[airetrieval.SourceType]sourceTable{
	airetrieval.SourceTypeMemory: {
		ids: func(after pulid.ID, limit int) keel.Projection {
			m := agentdb.Memories
			return m.Pick(m.ID).
				Where(keel.When(after.IsNotNil(), m.ID.Gt(after))).
				Order(m.ID.Asc()).
				Limit(limit)
		},
		count: func(ctx context.Context, db *keel.DB) (int64, error) {
			return agentdb.Memories.Query().Count(ctx, db)
		},
		sample: func(limit int) *keel.CTETable {
			m := agentdb.Memories
			return charsSample(m.Pick(charCount(m.Content)).
				Order(m.ID.Desc()).
				Limit(limit))
		},
	},
	airetrieval.SourceTypeDocument: {
		ids: func(after pulid.ID, limit int) keel.Projection {
			d := documentdb.Documents
			return d.Pick(d.ID).
				Where(keel.When(after.IsNotNil(), d.ID.Gt(after))).
				Order(d.ID.Asc()).
				Limit(limit)
		},
		count: func(ctx context.Context, db *keel.DB) (int64, error) {
			return documentdb.Documents.Query().Count(ctx, db)
		},
		sample: func(limit int) *keel.CTETable {
			d := documentdb.Documents
			dc := documentcontentdb.Contents
			return charsSample(d.LeftJoin(dc, dc.DocumentID.EqCol(d.ID)).
				Pick(charCount(dc.ContentText)).
				Order(d.ID.Desc()).
				Limit(limit))
		},
	},
	airetrieval.SourceTypeInboundMessage: {
		ids: func(after pulid.ID, limit int) keel.Projection {
			m := inboundmessagedb.InboundMessages
			return m.Pick(m.ID).
				Where(keel.When(after.IsNotNil(), m.ID.Gt(after))).
				Order(m.ID.Asc()).
				Limit(limit)
		},
		count: func(ctx context.Context, db *keel.DB) (int64, error) {
			return inboundmessagedb.InboundMessages.Query().Count(ctx, db)
		},
		sample: func(limit int) *keel.CTETable {
			m := inboundmessagedb.InboundMessages
			return charsSample(m.Pick(charCount(m.Subject).Add(charCount(m.TextBody))).
				Order(m.ID.Desc()).
				Limit(limit))
		},
	},
}

func charCount(e keel.Expr) keel.DynExpr {
	return keel.DynFunc("length", keel.DynOf(e).Coalesce(""))
}

func charsSample[T any](q *keel.Query[T]) *keel.CTETable {
	return keel.CTE(charsSampleCTE, []string{"chars"}, q)
}

func sourceTableFor(sourceType airetrieval.SourceType) (sourceTable, error) {
	table, ok := sourceTables[sourceType]
	if !ok {
		return sourceTable{}, fmt.Errorf(
			"%w: source type %q is not one retrieval indexes",
			airetrieval.ErrInvalidStorageRequest,
			sourceType,
		)
	}

	return table, nil
}

func (r *repository) CountSources(
	ctx context.Context,
	req agentports.CountRetrievalSourcesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		if err := validateTenant(req.TenantInfo); err != nil {
			return 0, err
		}

		table, err := sourceTableFor(req.SourceType)
		if err != nil {
			return 0, err
		}

		count, err := table.count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count %s sources: %w", req.SourceType, err)
		}

		return int(count), nil
	})
}

func (r *repository) AverageSourceChars(
	ctx context.Context,
	req agentports.AverageRetrievalSourceCharsRequest,
) (agentports.RetrievalSourceChars, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (agentports.RetrievalSourceChars, error) {
		var out agentports.RetrievalSourceChars

		if err := validateTenant(req.TenantInfo); err != nil {
			return out, err
		}

		table, err := sourceTableFor(req.SourceType)
		if err != nil {
			return out, err
		}

		sample := req.Sample
		if sample <= 0 {
			sample = defaultCharsSample
		}
		sample = intutils.Clamp(sample, 1, maxCharsSample)

		newest := table.sample(sample)
		out, err = keel.IntoOne[agentports.RetrievalSourceChars](
			ctx,
			r.db.Keel(ctx),
			newest.Query().
				With(newest).
				Pick(keel.Count(), keel.DynOf(newest.Col("chars")).Avg().Coalesce(0).Cast(keel.TypeFloat64)),
		)
		if err != nil {
			return agentports.RetrievalSourceChars{}, fmt.Errorf(
				"measure %s text: %w",
				req.SourceType,
				err,
			)
		}

		return out, nil
	})
}
