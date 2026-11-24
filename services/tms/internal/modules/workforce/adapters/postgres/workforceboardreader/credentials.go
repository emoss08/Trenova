package workforceboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential/workercredentialdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workforceboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardwording"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const defaultRenewalDays = 30

var (
	workerKind     = recordkind.Kind(permission.ResourceWorker)
	credentialKind = recordkind.Kind(permission.ResourceWorkerCredential)
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Workers struct{ db *postgres.Connection }

type Credentials struct{ db *postgres.Connection }

type Expiry struct{ db *postgres.Connection }

func NewWorkers(p Params) *Workers { return &Workers{db: p.DB} }

func NewCredentials(p Params) *Credentials { return &Credentials{db: p.DB} }

func NewExpiry(p Params) *Expiry { return &Expiry{db: p.DB} }

func (*Workers) Kind() recordkind.Kind     { return workerKind }
func (*Credentials) Kind() recordkind.Kind { return credentialKind }
func (*Expiry) Kind() recordkind.Kind      { return credentialKind }
func (*Expiry) Measure() string            { return workforceboard.MeasureExpiry }

type workerRow struct {
	ID       pulid.ID
	Label    string
	Subtitle string
	Status   string
}

type credentialRow struct {
	ID        pulid.ID
	WorkerID  pulid.ID
	Worker    string
	Type      string
	Status    string
	ExpiresAt *int64
	Renewal   int32
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func (r *Workers) Summaries(
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
			t := workerdb.Workers
			rows, err := keel.Into[workerRow](ctx, r.db.Keel(ctx), t.Pick(
				t.ID,
				text(keel.ConcatWS(" ", t.FirstName, t.LastName)),
				text(t.Type),
				text(t.Status),
			).Where(t.ID.In(req.IDs...)))
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for _, row := range rows {
				status, tone := boardwording.ActiveStatus(ctx, row.Status)
				out[row.ID] = boardreading.Summary{
					Label:      strings.TrimSpace(row.Label),
					Subtitle:   strings.TrimSpace(row.Subtitle),
					Status:     status,
					StatusTone: tone,
				}
			}

			return out, nil
		},
	)
}

func readCredentials(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
) ([]credentialRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	wc := workercredentialdb.WorkerCredentials
	w := workerdb.Workers
	ct := workercredentialdb.WorkerCredentialTypes

	return keel.Into[credentialRow](ctx, db.Keel(ctx),
		wc.Join(w, w.ID.EqCol(wc.WorkerID)).
			LeftJoin(ct, ct.ID.EqCol(wc.CredentialTypeID)).
			Pick(
				wc.ID, wc.WorkerID, text(keel.ConcatWS(" ", w.FirstName, w.LastName)),
				text(ct.Name), text(wc.Status), wc.ExpiresAt,
				ct.RenewalWindowDays.Coalesce(0),
			).
			Where(wc.ID.In(ids...)))
}

func credentialStatus(
	ctx context.Context,
	row *credentialRow,
	now int64,
) (string, boardcatalog.Tone) {
	if row.Status != string(workercredential.CredentialStatusActive) {
		return i18n.T(ctx, "Archived"), ""
	}
	switch workercredential.EvaluateCredentialHealth(row.ExpiresAt, row.Renewal, now) {
	case workercredential.CredentialHealthExpired:
		return i18n.T(ctx, "Expired"), boardcatalog.ToneBad
	case workercredential.CredentialHealthExpiringSoon:
		return i18n.T(ctx, "Expiring soon"), boardcatalog.ToneWarn
	case workercredential.CredentialHealthMissing:
		return i18n.T(ctx, "Missing"), boardcatalog.ToneBad
	case workercredential.CredentialHealthValid:
	}

	return i18n.T(ctx, "Valid"), boardcatalog.ToneOK
}

func (r *Credentials) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			rows, err := readCredentials(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for idx := range rows {
				row := &rows[idx]
				status, tone := credentialStatus(ctx, row, req.Now)
				summary := boardreading.Summary{
					Label:      strings.TrimSpace(row.Worker),
					Subtitle:   strings.TrimSpace(row.Type),
					Status:     status,
					StatusTone: tone,
				}
				if row.WorkerID.IsNotNil() {
					summary.Opens = &boardcatalog.Ref{Kind: workerKind, ID: row.WorkerID}
				}
				out[row.ID] = summary
			}

			return out, nil
		},
	)
}

func (r *Expiry) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := readCredentials(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Reading, len(rows))
			for idx := range rows {
				row := &rows[idx]
				if row.ExpiresAt == nil || *row.ExpiresAt <= 0 {
					continue
				}
				out[row.ID] = expiry(ctx, row, req)
			}

			return out, nil
		},
	)
}

func expiry(
	ctx context.Context,
	row *credentialRow,
	req *boardreaderports.RecordRequest,
) boardreading.Reading {
	renewal := int64(row.Renewal)
	if renewal <= 0 {
		renewal = defaultRenewalDays
	}
	countdown := boardwording.Countdown{
		Ends:     *row.ExpiresAt,
		Span:     renewal * timeutils.SecondsPerDay,
		Date:     true,
		Location: req.Location,
		Ending:   i18n.T(ctx, "Expires {0}"),
		Ended:    i18n.T(ctx, "Expired {0}"),
	}

	return countdown.Read(ctx, req.Now)
}
