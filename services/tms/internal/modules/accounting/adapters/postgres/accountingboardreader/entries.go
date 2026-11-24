package accountingboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingboard"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const slot = "{0}"

var entryKind = recordkind.Kind(permission.ResourceJournalEntry)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Entries struct{ db *postgres.Connection }

type Amount struct{ db *postgres.Connection }

type PendingFor struct{ db *postgres.Connection }

func NewEntries(p Params) *Entries { return &Entries{db: p.DB} }

func NewAmount(p Params) *Amount { return &Amount{db: p.DB} }

func NewPendingFor(p Params) *PendingFor { return &PendingFor{db: p.DB} }

func (*Entries) Kind() recordkind.Kind    { return entryKind }
func (*Amount) Kind() recordkind.Kind     { return entryKind }
func (*Amount) Measure() string           { return accountingboard.MeasureAmount }
func (*PendingFor) Kind() recordkind.Kind { return entryKind }
func (*PendingFor) Measure() string       { return accountingboard.MeasurePendingFor }

type entryRow struct {
	ID        pulid.ID
	Label     string
	Subtitle  string
	Status    string
	Debit     int64
	CreatedAt int64
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func readEntries(ctx context.Context, db *postgres.Connection, ids []pulid.ID) ([]entryRow, error) {
	t := journalentrydb.JournalEntries

	return keel.Into[entryRow](ctx, db.Keel(ctx), t.Pick(
		t.ID, text(t.EntryNumber), text(t.Description), text(t.Status), t.TotalDebit, t.CreatedAt,
	).Where(t.ID.In(ids...)))
}

func read[T any](
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
	each func(row *entryRow) (T, bool),
) (map[pulid.ID]T, error) {
	if len(ids) == 0 {
		return map[pulid.ID]T{}, nil
	}

	rows, err := readEntries(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]T, len(rows))
	for idx := range rows {
		if value, ok := each(&rows[idx]); ok {
			out[rows[idx].ID] = value
		}
	}

	return out, nil
}

func (r *Entries) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			return read(ctx, r.db, req.IDs, func(row *entryRow) (boardreading.Summary, bool) {
				status, tone := entryStatus(ctx, row.Status)
				return boardreading.Summary{
					Label:      strings.TrimSpace(row.Label),
					Subtitle:   strings.TrimSpace(row.Subtitle),
					Status:     status,
					StatusTone: tone,
				}, true
			})
		},
	)
}

func (r *Amount) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			return read(ctx, r.db, req.IDs, func(row *entryRow) (boardreading.Reading, bool) {
				value := boardcatalog.Money(row.Debit, "")

				return boardreading.Reading{
					Value:    value,
					Standing: boardreading.Words{Text: slot, Value: &value},
				}, true
			})
		},
	)
}

func waiting(status string) bool {
	return status == string(journalentry.StatusDraft) ||
		status == string(journalentry.StatusPending) ||
		status == string(journalentry.StatusApproved)
}

func (r *PendingFor) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			return read(ctx, r.db, req.IDs, func(row *entryRow) (boardreading.Reading, bool) {
				since := &boardreading.Words{Text: i18n.T(ctx, "Since {0}"), At: row.CreatedAt}
				if !waiting(row.Status) {
					return boardreading.Reading{
						Value:    boardcatalog.Duration(0),
						Standing: boardreading.Words{Text: i18n.T(ctx, "Settled")},
						Fact:     since,
						Tone:     boardcatalog.ToneOK,
						Track:    &boardreading.TrackFacts{},
					}, true
				}
				pending := max(req.Now-row.CreatedAt, 0)
				value := boardcatalog.Duration(pending)

				return boardreading.Reading{
					Value:    value,
					Standing: boardreading.Words{Text: i18n.T(ctx, "Waiting {0}"), Value: &value},
					Fact:     since,
					Tone:     boardcatalog.ToneWarn,
					Track:    &boardreading.TrackFacts{Behind: pending},
				}, true
			})
		},
	)
}
