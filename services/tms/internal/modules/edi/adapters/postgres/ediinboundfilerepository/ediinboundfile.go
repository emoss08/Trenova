//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package ediinboundfilerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultRecentQuarantinedLimit = 10

var inboundFileErrors = dberror.Mapping{Entity: "EDIInboundFile"}

var purgeableStatuses = []edi.InboundFileStatus{
	edi.InboundFileStatusProcessed,
	edi.InboundFileStatusDuplicate,
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

func New(p Params) ediports.EDIInboundFileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-inbound-file-repository"),
	}
}

func (r *repository) ListInboundFiles(
	ctx context.Context,
	req *ediports.ListEDIInboundFilesRequest,
) (*pagination.ListResult[*edi.EDIInboundFile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIInboundFile], error) {
			f := edidb.EDIInboundFiles
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				listedFiles(req),
				&listquery.Request[edi.EDIInboundFile]{
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
	req *ediports.ListEDIInboundFilesRequest,
) (*pagination.CursorListResult[*edi.EDIInboundFile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDIInboundFile], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				listedFiles(req),
				&listquery.Request[edi.EDIInboundFile]{
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
	req ediports.GetEDIInboundFileByIDRequest,
) (*edi.EDIInboundFile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIInboundFile, error) {
		f := edidb.EDIInboundFiles
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
	entity *edi.EDIInboundFile,
) (*edi.EDIInboundFile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIInboundFile, error) {
		if err := edidb.EDIInboundFiles.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, inboundFileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateInboundFile(
	ctx context.Context,
	entity *edi.EDIInboundFile,
) (*edi.EDIInboundFile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIInboundFile, error) {
		if err := edidb.EDIInboundFiles.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, inboundFileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) PurgeRawContentBefore(
	ctx context.Context,
	req ediports.PurgeEDIRawPayloadsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		f := edidb.EDIInboundFiles
		due := edidb.EDIInboundFiles.As("due")
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
		f := edidb.EDIInboundFiles
		return f.AllTenants().
			Where(f.Status.Eq(edi.InboundFileStatusQuarantined), f.UpdatedAt.Gte(since)).
			Count(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ExistsByChecksum(
	ctx context.Context,
	req ediports.ExistsEDIInboundFileByChecksumRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		f := edidb.EDIInboundFiles
		return f.Where(
			f.CommunicationProfileID.Eq(req.CommunicationProfileID),
			f.Checksum.Eq(req.Checksum),
		).Exists(ctx, r.db.Keel(ctx))
	})
}

type statusCount struct {
	Status edi.InboundFileStatus
	Count  int64
}

func (r *repository) GetInboundFileStatusCounts(
	ctx context.Context,
	req ediports.GetEDIInboundFileStatusCountsRequest,
) (map[edi.InboundFileStatus]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[edi.InboundFileStatus]int, error) {
		f := edidb.EDIInboundFiles
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

		counts := make(map[edi.InboundFileStatus]int, len(rows))
		for _, row := range rows {
			counts[row.Status] = int(row.Count)
		}

		return counts, nil
	})
}

func (r *repository) ListRecentQuarantined(
	ctx context.Context,
	req ediports.ListRecentQuarantinedEDIInboundFilesRequest,
) ([]*edi.EDIInboundFile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIInboundFile, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultRecentQuarantinedLimit
		}

		f := edidb.EDIInboundFiles
		return f.Omit(f.RawContent).
			Where(
				f.Status.Eq(edi.InboundFileStatusQuarantined),
				keel.When(req.PartnerID.IsNotNil(), f.EDIPartnerID.Eq(req.PartnerID)),
				keel.When(req.Since > 0, f.ReceivedAt.Gte(req.Since)),
			).
			Load(f.Partner()).
			Order(f.ReceivedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
	})
}

func listedFiles(req *ediports.ListEDIInboundFilesRequest) *keel.Query[edi.EDIInboundFile] {
	f := edidb.EDIInboundFiles
	return f.Omit(f.RawContent).Where(
		keel.When(req.Status != "", f.Status.Eq(req.Status)),
		keel.When(req.PartnerID.IsNotNil(), f.EDIPartnerID.Eq(req.PartnerID)),
	)
}

func withPartner(q *keel.Query[edi.EDIInboundFile]) *keel.Query[edi.EDIInboundFile] {
	return q.Load(edidb.EDIInboundFiles.Partner())
}
