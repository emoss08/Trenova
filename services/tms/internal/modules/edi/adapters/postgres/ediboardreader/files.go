package ediboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediboard"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
)

type InboundFiles struct{ db *postgres.Connection }

type FileWaiting struct{ db *postgres.Connection }

func NewInboundFiles(p Params) *InboundFiles { return &InboundFiles{db: p.DB} }

func NewFileWaiting(p Params) *FileWaiting { return &FileWaiting{db: p.DB} }

func (*InboundFiles) Kind() recordkind.Kind { return recordkind.EDIInboundFile }
func (*FileWaiting) Kind() recordkind.Kind  { return recordkind.EDIInboundFile }
func (*FileWaiting) Measure() string        { return ediboard.MeasureWaiting }

type fileRow struct {
	ID            pulid.ID
	FileName      string
	Partner       string
	Status        string
	FailureReason string
	ReceivedAt    int64
	ProcessedAt   *int64
}

func queryFiles(ctx context.Context, db *postgres.Connection, ids []pulid.ID) ([]fileRow, error) {
	f := ediinboundfiledb.EDIInboundFiles
	p := edidb.EDIPartners

	return keel.Into[fileRow](ctx, db.Keel(ctx),
		f.LeftJoin(p, p.ID.EqCol(f.EDIPartnerID)).
			Pick(
				f.ID, text(f.FileName), text(p.Name), text(f.Status),
				text(f.FailureReason), f.ReceivedAt, f.ProcessedAt,
			).
			Where(f.ID.In(ids...)))
}

func (r *InboundFiles) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Summary{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			rows, err := queryFiles(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for idx := range rows {
				row := &rows[idx]
				status, tone := fileStatus(ctx, row.Status)
				out[row.ID] = boardreading.Summary{
					Label:      strings.TrimSpace(row.FileName),
					Subtitle:   strings.TrimSpace(row.Partner),
					Status:     status,
					StatusTone: tone,
				}
			}

			return out, nil
		},
	)
}

func (r *FileWaiting) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := queryFiles(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Reading, len(rows))
			for idx := range rows {
				out[rows[idx].ID] = fileWaiting(ctx, &rows[idx], req.Now)
			}

			return out, nil
		},
	)
}

func fileWaiting(ctx context.Context, row *fileRow, now int64) boardreading.Reading {
	reading := boardreading.Reading{Value: boardcatalog.Duration(0)}
	if reason := firstLine(row.FailureReason); reason != "" {
		reading.Fact = &boardreading.Words{
			Text:  "{0}",
			Value: &boardcatalog.Value{Kind: boardcatalog.ValueText, Text: reason},
		}
	}
	waiting := fileNeedsHand(row.Status) || fileUnderWay(row.Status)
	if !waiting {
		at := row.ReceivedAt
		if row.ProcessedAt != nil && *row.ProcessedAt > 0 {
			at = *row.ProcessedAt
		}
		reading.Standing = boardreading.Words{Text: doneWords(ctx, row.Status), At: at}

		return reading
	}
	waited := boardcatalog.Duration(max(now-row.ReceivedAt, 0))
	reading.Value = waited
	reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} waiting"), Value: &waited}
	if fileNeedsHand(row.Status) {
		reading.Tone = boardcatalog.ToneWarn
	}

	return reading
}

func doneWords(ctx context.Context, status string) string {
	if ediinboundfile.InboundFileStatus(status) == ediinboundfile.InboundFileStatusDuplicate {
		return i18n.T(ctx, "Set aside {0}")
	}

	return i18n.T(ctx, "Processed {0}")
}
