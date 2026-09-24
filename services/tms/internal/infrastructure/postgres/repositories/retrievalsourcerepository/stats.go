package retrievalsourcerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/uptrace/bun"
)

const (
	defaultCharsSample = 500
	maxCharsSample     = 5000
	charsSampleAlias   = "_sample"
	charsColumn        = "chars"
)

var sourceChars = map[airetrieval.SourceType]func(*bun.SelectQuery) *bun.SelectQuery{
	airetrieval.SourceTypeMemory: func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.ColumnExpr(
			buncolgen.MemoryColumns.Content.Expr("LENGTH(COALESCE({}, '')) AS " + charsColumn),
		)
	},
	airetrieval.SourceTypeDocument: func(q *bun.SelectQuery) *bun.SelectQuery {
		content := buncolgen.ContentColumns
		doc := buncolgen.DocumentColumns

		return q.
			ColumnExpr(content.ContentText.Expr("LENGTH(COALESCE({}, '')) AS " + charsColumn)).
			Join("LEFT JOIN " + buncolgen.ContentTable.As(buncolgen.ContentTable.Alias)).
			JoinOn(content.DocumentID.EqColumn(doc.ID)).
			JoinOn(content.OrganizationID.EqColumn(doc.OrganizationID)).
			JoinOn(content.BusinessUnitID.EqColumn(doc.BusinessUnitID))
	},
	airetrieval.SourceTypeInboundMessage: func(q *bun.SelectQuery) *bun.SelectQuery {
		msg := buncolgen.InboundMessageColumns

		return q.ColumnExpr(buncolgen.Expr(
			"LENGTH(COALESCE({0}, '')) + LENGTH(COALESCE({1}, '')) AS "+charsColumn,
			msg.Subject,
			msg.TextBody,
		))
	},
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
	req repositories.CountRetrievalSourcesRequest,
) (int, error) {
	if err := validateTenant(req.TenantInfo); err != nil {
		return 0, err
	}

	table, err := sourceTableFor(req.SourceType)
	if err != nil {
		return 0, err
	}

	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(table.model).
		Apply(table.applyTenant(req.TenantInfo)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count %s sources: %w", req.SourceType, err)
	}

	return count, nil
}

func (r *repository) AverageSourceChars(
	ctx context.Context,
	req repositories.AverageRetrievalSourceCharsRequest,
) (repositories.RetrievalSourceChars, error) {
	var out repositories.RetrievalSourceChars

	if err := validateTenant(req.TenantInfo); err != nil {
		return out, err
	}

	table, err := sourceTableFor(req.SourceType)
	if err != nil {
		return out, err
	}
	chars, ok := sourceChars[req.SourceType]
	if !ok {
		return out, fmt.Errorf(
			"%w: source type %q has no text to measure",
			airetrieval.ErrInvalidStorageRequest,
			req.SourceType,
		)
	}

	sample := req.Sample
	if sample <= 0 {
		sample = defaultCharsSample
	}
	sample = intutils.Clamp(sample, 1, maxCharsSample)

	dba := r.db.DBForContext(ctx)
	newest := dba.NewSelect().
		Model(table.model).
		Apply(chars).
		Apply(table.applyTenant(req.TenantInfo)).
		OrderExpr(table.id.OrderDesc()).
		Limit(sample)

	if err = dba.NewSelect().
		TableExpr("(?) AS "+charsSampleAlias, newest).
		ColumnExpr(buncolgen.Count("sampled")).
		ColumnExpr("COALESCE(AVG("+charsSampleAlias+"."+charsColumn+"), 0) AS average_chars").
		Scan(ctx, &out); err != nil {
		return repositories.RetrievalSourceChars{}, fmt.Errorf(
			"measure %s text: %w",
			req.SourceType,
			err,
		)
	}

	return out, nil
}
