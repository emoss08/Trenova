//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package ediinboundfilerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultRecentQuarantinedLimit = 10

var inboundFileErrors = dberror.Mapping{Entity: "EDIInboundFile"}

var purgeableStatuses = []ediinboundfile.InboundFileStatus{
	ediinboundfile.InboundFileStatusProcessed,
	ediinboundfile.InboundFileStatusDuplicate,
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) edimessageports.EDIInboundFileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-inbound-file-repository"),
	}
}

func (r *repository) ListInboundFiles(
	ctx context.Context,
	req *edimessageports.ListEDIInboundFilesRequest,
) (*pagination.ListResult[*ediinboundfile.EDIInboundFile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ediinboundfile.EDIInboundFile], error) {
			f := ediinboundfiledb.EDIInboundFiles
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				listedFiles(req),
				&listquery.Request[ediinboundfile.EDIInboundFile]{
					Filter: req.Filter,
					Shape:  withPartner,
					Then:   []keel.Order{f.ReceivedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) ListInboundFilesCursor(
	ctx context.Context,
	req *edimessageports.ListEDIInboundFilesRequest,
) (*pagination.CursorListResult[*ediinboundfile.EDIInboundFile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ediinboundfile.EDIInboundFile], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				listedFiles(req),
				&listquery.Request[ediinboundfile.EDIInboundFile]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withPartner,
				},
			)
		},
	)
}

func (r *repository) GetInboundFileByID(
	ctx context.Context,
	req edimessageports.GetEDIInboundFileByIDRequest,
) (*ediinboundfile.EDIInboundFile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ediinboundfile.EDIInboundFile, error) {
		f := ediinboundfiledb.EDIInboundFiles
		q := f.Where(f.ID.Eq(req.ID)).Load(f.Partner(), f.CommunicationProfile())
		if req.IncludeMessages {
			q.Load(f.Messages())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, inboundFileErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateInboundFile(
	ctx context.Context,
	entity *ediinboundfile.EDIInboundFile,
) (*ediinboundfile.EDIInboundFile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ediinboundfile.EDIInboundFile, error) {
		if err := ediinboundfiledb.EDIInboundFiles.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, inboundFileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateInboundFile(
	ctx context.Context,
	entity *ediinboundfile.EDIInboundFile,
) (*ediinboundfile.EDIInboundFile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ediinboundfile.EDIInboundFile, error) {
		if err := ediinboundfiledb.EDIInboundFiles.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, inboundFileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) PurgeRawContentBefore(
	ctx context.Context,
	req edimessageports.PurgeEDIRawPayloadsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		f := ediinboundfiledb.EDIInboundFiles
		due := ediinboundfiledb.EDIInboundFiles.As("due")
		batch := due.Pick(due.ID).
			Where(
				due.ReceivedAt.Lt(req.Before),
				due.Status.In(purgeableStatuses...),
				due.RawPurgedAt.IsNull(),
			).
			Limit(req.Limit)

		return f.Where(f.ID.InQuery(batch)).Update(
			ctx,
			r.db.Keel(ctx),
			f.RawContent.To(""),
			f.RawPurgedAt.To(req.PurgedAt),
			f.Version.ToExpr(f.Version),
			f.UpdatedAt.ToExpr(f.UpdatedAt),
		)
	})
}

func (r *repository) CountQuarantinedSince(ctx context.Context, since int64) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"count quarantined inbound EDI files instance-wide for the health check",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		f := ediinboundfiledb.EDIInboundFiles
		return f.AllTenants().
			Where(f.Status.Eq(ediinboundfile.InboundFileStatusQuarantined), f.UpdatedAt.Gte(since)).
			Count(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ExistsByChecksum(
	ctx context.Context,
	req edimessageports.ExistsEDIInboundFileByChecksumRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		f := ediinboundfiledb.EDIInboundFiles
		return f.Where(
			f.CommunicationProfileID.Eq(req.CommunicationProfileID),
			f.Checksum.Eq(req.Checksum),
		).Exists(ctx, r.db.Keel(ctx))
	})
}

type statusCount struct {
	Status ediinboundfile.InboundFileStatus
	Count  int64
}

func (r *repository) GetInboundFileStatusCounts(
	ctx context.Context,
	req edimessageports.GetEDIInboundFileStatusCountsRequest,
) (map[ediinboundfile.InboundFileStatus]int, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[ediinboundfile.InboundFileStatus]int, error) {
			f := ediinboundfiledb.EDIInboundFiles
			rows, err := keel.Into[statusCount](
				ctx,
				r.db.Keel(ctx),
				f.Where(keel.When(req.Since > 0, f.ReceivedAt.Gte(req.Since))).
					Group(f.Status).
					Pick(f.Status, keel.Count()),
			)
			if err != nil {
				return nil, err
			}

			counts := make(map[ediinboundfile.InboundFileStatus]int, len(rows))
			for _, row := range rows {
				counts[row.Status] = int(row.Count)
			}

			return counts, nil
		},
	)
}

func (r *repository) ListRecentQuarantined(
	ctx context.Context,
	req edimessageports.ListRecentQuarantinedEDIInboundFilesRequest,
) ([]*ediinboundfile.EDIInboundFile, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ediinboundfile.EDIInboundFile, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultRecentQuarantinedLimit
			}

			f := ediinboundfiledb.EDIInboundFiles
			return f.Omit(f.RawContent).
				Where(
					f.Status.Eq(ediinboundfile.InboundFileStatusQuarantined),
					keel.When(req.PartnerID.IsNotNil(), f.EDIPartnerID.Eq(req.PartnerID)),
					keel.When(req.Since > 0, f.ReceivedAt.Gte(req.Since)),
				).
				Load(f.Partner()).
				Order(f.ReceivedAt.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
		},
	)
}

func listedFiles(
	req *edimessageports.ListEDIInboundFilesRequest,
) *keel.Query[*ediinboundfile.EDIInboundFile] {
	f := ediinboundfiledb.EDIInboundFiles
	return f.Omit(f.RawContent).Where(
		keel.When(req.Status != "", f.Status.Eq(req.Status)),
		keel.When(req.PartnerID.IsNotNil(), f.EDIPartnerID.Eq(req.PartnerID)),
	)
}

func withPartner(
	q *keel.Query[*ediinboundfile.EDIInboundFile],
) *keel.Query[*ediinboundfile.EDIInboundFile] {
	return q.Load(ediinboundfiledb.EDIInboundFiles.Partner())
}
