package tenantcopyrepository

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

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

	tailSealSetting = "trenova.archive_tail_seal"
	tailHeadSetting = "trenova.archive_tail_head"

	setTailSQL = "SELECT set_config($1, $2, true)"

	movedHeadSQL = `SELECT (to_jsonb(h) || $3::jsonb)::text FROM public.audit_chain_heads h
WHERE h.organization_id = $1 AND h.business_unit_id = $2`

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
)

type chainHead struct {
	lastSeq       int64
	lastHash      string
	sealedThrough int64
}

var ErrTailRowShape = errors.New(
	"the archive's last audit seal does not match its table's columns",
)

type snapshotTrails struct {
	ctx        context.Context
	tx         pgx.Tx
	db         *keel.DB
	keys       *trailKeys
	tenant     dbscope.Tenant
	at         int64
	sealsTable *archiveplan.PlannedTable
	headsTable *archiveplan.PlannedTable
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
	seals, sealsCarried := s.plan.Table(archiveSealsTable)
	heads, headsCarried := s.plan.Table(archiveHeadsTable)
	if !sealsCarried || !headsCarried {
		return fmt.Errorf("%w: the plan carries no %s or %s", ErrTailRowShape,
			archiveSealsTable, archiveHeadsTable)
	}
	run := &snapshotTrails{
		ctx: joined, tx: s.tx, db: db, keys: s.repo.keys, tenant: s.tenant, at: at,
		sealsTable: seals, headsTable: heads,
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
	sealRow, err := tailRow(t.sealsTable, tailSealRow(next))
	if err != nil {
		return false, err
	}
	headRow, err := t.movedHead(next, head)
	if err != nil {
		return false, err
	}
	for _, setting := range [][2]string{{tailSealSetting, sealRow}, {tailHeadSetting, headRow}} {
		if _, err = t.tx.Exec(t.ctx, setTailSQL, setting[0], setting[1]); err != nil {
			return false, fmt.Errorf("hold the archive's last audit seal for its copy: %w", err)
		}
	}
	return true, nil
}

func tailSealRow(seal *audit.Seal) map[string]any {
	return map[string]any{
		"organization_id":     seal.OrganizationID.String(),
		"business_unit_id":    seal.BusinessUnitID.String(),
		"seq":                 seal.Seq,
		"from_ts":             seal.FromTS,
		"to_ts":               seal.ToTS,
		"row_count":           seal.RowCount,
		"class_digests":       seal.ClassDigests,
		"prev_hash":           seal.PrevHash,
		"hash":                seal.Hash,
		"hash_key_id":         nullIfEmpty(seal.HashKeyID),
		"hash_version":        seal.HashVersion,
		"sealed_at":           seal.SealedAt,
		"source_prev_hash":    nil,
		"source_hash":         nil,
		"source_hash_key_id":  nil,
		"source_hash_version": nil,
		"resealed_at":         nil,
	}
}

func headMove(seal *audit.Seal, at int64) map[string]any {
	return map[string]any{
		"last_seq":       seal.Seq,
		"last_hash":      seal.Hash,
		"sealed_through": seal.ToTS,
		"hash_key_id":    nullIfEmpty(seal.HashKeyID),
		"updated_at":     at,
	}
}

func newHeadRow(seal *audit.Seal, at int64) map[string]any {
	row := map[string]any{
		"organization_id":              seal.OrganizationID.String(),
		"business_unit_id":             seal.BusinessUnitID.String(),
		"last_verified_seq":            int64(0),
		"last_verified_at":             nil,
		"last_verification_status":     nil,
		"last_verification_failed_seq": nil,
		"last_verification_detail":     nil,
		"verify_cursor_seq":            int64(0),
		"version":                      int64(0),
		"created_at":                   at,
	}
	maps.Copy(row, headMove(seal, at))
	return row
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (t *snapshotTrails) movedHead(seal *audit.Seal, head *chainHead) (string, error) {
	if head == nil {
		return tailRow(t.headsTable, newHeadRow(seal, t.at))
	}
	move, err := sonic.ConfigStd.Marshal(headMove(seal, t.at))
	if err != nil {
		return "", fmt.Errorf("encode the archive's audit chain head: %w", err)
	}
	var moved string
	if err = t.tx.QueryRow(t.ctx, movedHeadSQL, t.tenant.OrganizationID.String(),
		t.tenant.BusinessUnitID.String(), string(move)).Scan(&moved); err != nil {
		return "", fmt.Errorf("read the archive's audit chain head: %w", err)
	}
	var row map[string]any
	if err = sonic.ConfigStd.UnmarshalFromString(moved, &row); err != nil {
		return "", fmt.Errorf("read the archive's audit chain head: %w", err)
	}
	if err = coversColumns(t.headsTable, row); err != nil {
		return "", err
	}
	return moved, nil
}

func tailRow(table *archiveplan.PlannedTable, row map[string]any) (string, error) {
	if err := coversColumns(table, row); err != nil {
		return "", err
	}
	body, err := sonic.ConfigStd.Marshal(row)
	if err != nil {
		return "", fmt.Errorf("encode the archive's %s row: %w", table.Name, err)
	}
	return string(body), nil
}

func coversColumns(table *archiveplan.PlannedTable, row map[string]any) error {
	missing := make([]string, 0)
	for _, column := range table.Columns {
		if _, ok := row[column]; !ok {
			missing = append(missing, column)
		}
	}
	extra := make([]string, 0)
	for column := range row {
		if !slices.Contains(table.Columns, column) {
			extra = append(extra, column)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return nil
	}
	slices.Sort(missing)
	slices.Sort(extra)
	return fmt.Errorf(
		"%w: %s: the archive's own row has no value for %v and values for columns the table lacks %v",
		ErrTailRowShape,
		table.Name,
		missing,
		extra,
	)
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
			" FROM " + heldRow(table.Name, tailSealSetting) + ") TO STDOUT (FORMAT binary)"
	case archiveHeadsTable:
		return "COPY (SELECT " + columns + " FROM " + heldRow(table.Name, tailHeadSetting) +
			") TO STDOUT (FORMAT binary)"
	default:
		return table.CopyOut(s.tenant)
	}
}

func heldRow(table, setting string) string {
	return "jsonb_populate_record(NULL::" + archiveplan.Qualified(table) + ", current_setting(" +
		archiveplan.QuoteLiteral(setting) + ")::jsonb) a"
}
