package tenantcopyrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/auditdigest"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
)

const (
	snapshotTrailsReason = "read one tenant's audit trails inside its archive snapshot, seal the entries its last seal does not cover and state what the archive carries"

	archiveSealsTable = "audit_seals"
	archiveHeadsTable = "audit_chain_heads"

	createArchiveSeals = `CREATE TEMP TABLE IF NOT EXISTS archive_audit_seals
    (LIKE public.audit_seals INCLUDING DEFAULTS)`
	createArchiveHeads = `CREATE TEMP TABLE IF NOT EXISTS archive_audit_chain_heads
    (LIKE public.audit_chain_heads INCLUDING DEFAULTS)`

	sealStatsSQL = `SELECT count(*)::bigint, COALESCE(min(seq), 0)::bigint FROM public.audit_seals
WHERE organization_id = $1 AND business_unit_id = $2`

	headSQL = `SELECT last_seq, last_hash, sealed_through FROM public.audit_chain_heads
WHERE organization_id = $1 AND business_unit_id = $2`

	entryBoundsSQL = `SELECT COALESCE(min("timestamp"), 0)::bigint, COALESCE(max("timestamp"), 0)::bigint
FROM public.audit_entries WHERE organization_id = $1 AND business_unit_id = $2`

	eventStatsSQL = `SELECT count(*)::bigint, COALESCE(min(seq), 0)::bigint, COALESCE(max(seq), 0)::bigint,
    COALESCE(max(recorded_at), 0)::bigint
FROM public.ai_audit_events WHERE organization_id = $1 AND business_unit_id = $2`

	eventHashSQL = `SELECT hash FROM public.ai_audit_events
WHERE organization_id = $1 AND business_unit_id = $2 AND seq = $3`

	insertArchiveSealSQL = `INSERT INTO pg_temp.archive_audit_seals
    (organization_id, business_unit_id, seq, from_ts, to_ts, row_count, class_digests, prev_hash,
     hash, hash_key_id, hash_version, sealed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, NULLIF($10, ''), $11, $12)`

	copyArchiveHeadSQL = `INSERT INTO pg_temp.archive_audit_chain_heads
SELECT * FROM public.audit_chain_heads WHERE organization_id = $1 AND business_unit_id = $2`

	newArchiveHeadSQL = `INSERT INTO pg_temp.archive_audit_chain_heads
    (organization_id, business_unit_id, created_at, updated_at)
VALUES ($1, $2, $3, $3)`

	moveArchiveHeadSQL = `UPDATE pg_temp.archive_audit_chain_heads SET
    last_seq = $1, last_hash = $2, sealed_through = $3, hash_key_id = NULLIF($4, ''), updated_at = $5`
)

type chainHead struct {
	lastSeq       int64
	lastHash      string
	sealedThrough int64
}

type snapshotTrails struct {
	ctx    context.Context
	tx     pgx.Tx
	db     *keel.DB
	keys   *trailKeys
	tenant dbscope.Tenant
	at     int64
}

func prepareTrailTables(ctx context.Context, conn *pgx.Conn) error {
	for _, statement := range []string{createArchiveSeals, createArchiveHeads} {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("prepare the archive's audit trail statement: %w", err)
		}
	}
	return nil
}

type trailBounds struct {
	firstEntry   int64
	lastEntry    int64
	events       int64
	firstEvent   int64
	lastEvent    int64
	lastRecorded int64
}

func (b *trailBounds) through(at int64) int64 {
	return max(at, b.lastEntry, b.lastRecorded) + 1
}

func (s *snapshot) readTrails(ctx context.Context, at int64) error {
	ctx = dbscope.WithSystem(ctx, snapshotTrailsReason)
	db := s.repo.db.Keel(ctx)
	joined, done := keel.JoinTx(ctx, db, s.conn)
	defer done(false)
	run := &snapshotTrails{
		ctx: joined, tx: s.tx, db: db, keys: s.repo.keys, tenant: s.tenant, at: at,
	}
	bounds := new(trailBounds)
	if err := run.row(entryBoundsSQL, &bounds.firstEntry, &bounds.lastEntry); err != nil {
		return fmt.Errorf("read the archived audit log's dates: %w", err)
	}
	if err := run.row(eventStatsSQL, &bounds.events, &bounds.firstEvent, &bounds.lastEvent,
		&bounds.lastRecorded); err != nil {
		return fmt.Errorf("read the archived AI audit chain: %w", err)
	}
	through := bounds.through(at)
	facts := new(archiveports.TrailFacts)
	var err error
	if facts.Audit, err = run.auditTrail(bounds, through); err != nil {
		return err
	}
	if facts.AIAudit, err = run.aiTrail(bounds, through); err != nil {
		return err
	}
	s.trails = facts
	s.sealedTail = facts.Audit != nil && facts.Audit.SealedHere > 0
	return nil
}

func (t *snapshotTrails) row(sql string, dst ...any) error {
	return t.tx.QueryRow(t.ctx, sql, t.tenant.OrganizationID.String(),
		t.tenant.BusinessUnitID.String()).Scan(dst...)
}

func (t *snapshotTrails) head() (*chainHead, error) {
	out := new(chainHead)
	err := t.row(headSQL, &out.lastSeq, &out.lastHash, &out.sealedThrough)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // a tenant whose trail was never sealed has no chain head
	}
	if err != nil {
		return nil, fmt.Errorf("read the archived audit chain head: %w", err)
	}
	return out, nil
}

func (t *snapshotTrails) lastSeal() (*audit.Seal, error) {
	s := auditdb.Seals
	seals, err := s.AllTenants().Where(
		s.OrganizationID.Eq(t.tenant.OrganizationID),
		s.BusinessUnitID.Eq(t.tenant.BusinessUnitID),
	).Order(s.Seq.Desc()).Limit(1).List(t.ctx, t.db)
	if err != nil {
		return nil, fmt.Errorf("read the archived audit seals: %w", err)
	}
	if len(seals) == 0 {
		return nil, nil //nolint:nilnil // a tenant whose trail was never sealed has no seal
	}
	return seals[0], nil
}

func (t *snapshotTrails) auditTrail(
	bounds *trailBounds,
	through int64,
) (*archiveports.ChainFacts, error) {
	var links, first int64
	if err := t.row(sealStatsSQL, &links, &first); err != nil {
		return nil, fmt.Errorf("count the archived audit seals: %w", err)
	}
	last, err := t.lastSeal()
	if err != nil {
		return nil, err
	}
	head, err := t.head()
	if err != nil {
		return nil, err
	}
	facts := &archiveports.ChainFacts{Links: links, FirstSeq: first}
	from, next := bounds.firstEntry, &audit.Seal{
		OrganizationID: t.tenant.OrganizationID,
		BusinessUnitID: t.tenant.BusinessUnitID,
		Seq:            1,
	}
	if head != nil {
		next.Seq, next.PrevHash = head.lastSeq+1, head.lastHash
		if head.sealedThrough > 0 {
			from = head.sealedThrough
		}
	}
	if last != nil {
		facts.LastSeq, facts.LastHash, facts.SealedThrough = last.Seq, last.Hash, last.ToTS
		from = last.ToTS
		next.Seq, next.PrevHash = max(next.Seq, last.Seq+1), last.Hash
	}
	if from > 0 && from < through {
		sealed, serr := t.sealTail(next, from, through, head)
		if serr != nil {
			return nil, serr
		}
		if sealed {
			facts.Links++
			facts.SealedHere = 1
			facts.SealedThrough = through
			if facts.FirstSeq == 0 {
				facts.FirstSeq = next.Seq
			}
		}
	}
	if facts.Links == 0 {
		return nil, nil //nolint:nilnil // a tenant without a sealed audit trail states none
	}
	return facts, nil
}

func (t *snapshotTrails) sealTail(
	next *audit.Seal,
	from, through int64,
	head *chainHead,
) (bool, error) {
	digest, err := auditdigest.Compute(t.ctx, t.db, auditdigest.Window{
		OrganizationID: t.tenant.OrganizationID,
		BusinessUnitID: t.tenant.BusinessUnitID,
		From:           from,
		To:             through,
	})
	if err != nil {
		return false, err
	}
	if digest.RowCount == 0 {
		return false, nil
	}
	next.FromTS = from
	next.ToTS = through
	next.RowCount = digest.RowCount
	next.ClassDigests = digest.Classes
	next.SealedAt = t.at
	t.keys.signSeal(next, audit.OriginHere)
	classes, err := sonic.ConfigStd.Marshal(next.ClassDigests)
	if err != nil {
		return false, fmt.Errorf("encode the archive's last audit seal: %w", err)
	}
	if _, err = t.tx.Exec(t.ctx, insertArchiveSealSQL, next.OrganizationID.String(),
		next.BusinessUnitID.String(), next.Seq, next.FromTS, next.ToTS, next.RowCount,
		string(classes), next.PrevHash, next.Hash, next.HashKeyID, next.HashVersion,
		next.SealedAt); err != nil {
		return false, fmt.Errorf("seal the archive's audit entries after its last seal: %w", err)
	}
	org, bu := t.tenant.OrganizationID.String(), t.tenant.BusinessUnitID.String()
	statement, args := copyArchiveHeadSQL, []any{org, bu}
	if head == nil {
		statement, args = newArchiveHeadSQL, []any{org, bu, t.at}
	}
	if _, err = t.tx.Exec(t.ctx, statement, args...); err != nil {
		return false, fmt.Errorf("carry the archive's audit chain head: %w", err)
	}
	if _, err = t.tx.Exec(t.ctx, moveArchiveHeadSQL, next.Seq, next.Hash, next.ToTS,
		next.HashKeyID, t.at); err != nil {
		return false, fmt.Errorf("move the archive's audit chain head to its last seal: %w", err)
	}
	return true, nil
}

func (t *snapshotTrails) aiTrail(
	bounds *trailBounds,
	through int64,
) (*archiveports.ChainFacts, error) {
	if bounds.events == 0 {
		return nil, nil //nolint:nilnil // a tenant without AI audit rows states no AI trail
	}
	facts := &archiveports.ChainFacts{
		Links:         bounds.events,
		FirstSeq:      bounds.firstEvent,
		LastSeq:       bounds.lastEvent,
		SealedThrough: through,
	}
	if err := t.tx.QueryRow(t.ctx, eventHashSQL, t.tenant.OrganizationID.String(),
		t.tenant.BusinessUnitID.String(), bounds.lastEvent).Scan(&facts.LastHash); err != nil {
		return nil, fmt.Errorf("read the archived AI audit chain's last row: %w", err)
	}
	return facts, nil
}

func (s *snapshot) copyOutQuery(table *archiveplan.PlannedTable) string {
	if !s.sealedTail {
		return table.CopyOut(s.tenant)
	}
	columns := archiveplan.JoinIdents(table.Columns, "a")
	switch table.Name {
	case archiveSealsTable:
		return "COPY (SELECT " + columns + " FROM " + archiveplan.Qualified(table.Name) +
			" a WHERE " + table.Where("a", s.tenant) + " UNION ALL SELECT " + columns +
			" FROM pg_temp.archive_audit_seals a) TO STDOUT (FORMAT binary)"
	case archiveHeadsTable:
		return "COPY (SELECT " + columns +
			" FROM pg_temp.archive_audit_chain_heads a) TO STDOUT (FORMAT binary)"
	default:
		return table.CopyOut(s.tenant)
	}
}
