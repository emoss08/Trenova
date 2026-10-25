package triagetemplaterepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

var templateErrors = dberror.Mapping{Entity: "TriageReplyTemplate"}

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) inboundmessageports.TriageReplyTemplateRepository {
	return &repository{db: p.DB}
}

func (r *repository) List(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*triage.ReplyTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.ReplyTemplate, error) {
		t := triagedb.ReplyTemplates
		entities, err := t.Order(keel.Asc(t.Name.Lower()), t.ID.Asc()).
			Limit(triage.MaxTemplatesPerTenant).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if entities == nil {
			entities = []*triage.ReplyTemplate{}
		}

		return entities, nil
	})
}

func (r *repository) Count(ctx context.Context, _ pagination.TenantInfo) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := triagedb.ReplyTemplates.Query().Count(ctx, r.db.Keel(ctx))

		return int(count), err
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req inboundmessageports.GetTriageReplyTemplateRequest,
) (*triage.ReplyTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.ReplyTemplate, error) {
		entity, err := triagedb.ReplyTemplates.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, templateErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) NameTaken(
	ctx context.Context,
	_ pagination.TenantInfo,
	name string,
	exceptID pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		t := triagedb.ReplyTemplates
		return t.Where(
			t.Name.Lower().Eq(strings.ToLower(strings.TrimSpace(name))),
			keel.When(exceptID.IsNotNil(), t.ID.Ne(exceptID)),
		).Exists(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *triage.ReplyTemplate,
) (*triage.ReplyTemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.ReplyTemplate, error) {
		if err := triagedb.ReplyTemplates.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, templateErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *triage.ReplyTemplate,
) (*triage.ReplyTemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.ReplyTemplate, error) {
		if err := triagedb.ReplyTemplates.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, templateErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req inboundmessageports.GetTriageReplyTemplateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := triagedb.ReplyTemplates.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&triage.ReplyTemplate{ID: req.ID},
		)

		return templateErrors.Map(err, req.ID.String())
	})
}
