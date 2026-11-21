package tenantcopyrepository

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/auditdigest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit/aiauditdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
)

const (
	trailsReason = "sign an imported tenant's audit seals and AI audit chain under this install's key inside the archive load, and record where its history came from"
	trailPage    = 1_000

	ActionImported = "organization.imported"
	ActionRestored = "organization.restored"
	provenanceName = "Trenova"

	reasonFutureEntries = "the archive's audit log holds entries dated at or after the moment it is loaded, which no archive taken earlier can hold"

	resealSealsSQL = `UPDATE public.audit_seals s SET
    source_prev_hash = COALESCE(s.source_prev_hash, s.prev_hash),
    source_hash = COALESCE(s.source_hash, s.hash),
    source_hash_key_id = CASE WHEN s.source_hash IS NULL THEN s.hash_key_id ELSE s.source_hash_key_id END,
    source_hash_version = COALESCE(s.source_hash_version, s.hash_version),
    resealed_at = $3,
    prev_hash = u.prev_hash,
    hash = u.hash,
    hash_key_id = NULLIF(u.hash_key_id, ''),
    hash_version = u.hash_version
FROM unnest($4::bigint[], $5::text[], $6::text[], $7::text[], $8::smallint[])
    AS u(seq, prev_hash, hash, hash_key_id, hash_version)
WHERE s.organization_id = $1 AND s.business_unit_id = $2 AND s.seq = u.seq`

	resealEventsSQL = `UPDATE public.ai_audit_events e SET
    source_prev_hash = COALESCE(e.source_prev_hash, e.prev_hash),
    source_hash = COALESCE(e.source_hash, e.hash),
    source_hash_key_id = CASE WHEN e.source_hash IS NULL THEN e.hash_key_id ELSE e.source_hash_key_id END,
    source_hash_version = COALESCE(e.source_hash_version, e.hash_version),
    resealed_at = $3,
    prev_hash = u.prev_hash,
    hash = u.hash,
    hash_key_id = NULLIF(u.hash_key_id, ''),
    hash_version = u.hash_version
FROM unnest($4::text[], $5::text[], $6::text[], $7::text[], $8::smallint[])
    AS u(id, prev_hash, hash, hash_key_id, hash_version)
WHERE e.organization_id = $1 AND e.business_unit_id = $2 AND e.id = u.id`

	resealAISealsSQL = `UPDATE public.ai_audit_seals s SET
    source_head_hash = COALESCE(s.source_head_hash, s.head_hash),
    source_hash_key_id = CASE WHEN s.source_head_hash IS NULL THEN s.hash_key_id ELSE s.source_hash_key_id END,
    head_hash = e.hash,
    hash_key_id = e.hash_key_id
FROM public.ai_audit_events e
WHERE s.organization_id = $1 AND s.business_unit_id = $2
    AND e.organization_id = s.organization_id AND e.business_unit_id = s.business_unit_id
    AND e.seq = s.to_seq`

	auditHeadSQL = `INSERT INTO public.audit_chain_heads
    (organization_id, business_unit_id, last_seq, last_hash, sealed_through, hash_key_id)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
ON CONFLICT (organization_id, business_unit_id) DO UPDATE SET
    last_seq = EXCLUDED.last_seq, last_hash = EXCLUDED.last_hash,
    sealed_through = EXCLUDED.sealed_through, hash_key_id = EXCLUDED.hash_key_id,
    last_verified_seq = 0, last_verified_at = NULL, last_verification_status = NULL,
    last_verification_failed_seq = NULL, last_verification_detail = NULL, verify_cursor_seq = 0`

	aiHeadSQL = `UPDATE public.ai_audit_chain_heads SET
    last_hash = $3, hash_key_id = NULLIF($4, ''), last_verified_seq = 0, last_verified_at = NULL,
    last_verification_status = NULL, last_verification_failed_seq = NULL,
    last_verification_detail = NULL
WHERE organization_id = $1 AND business_unit_id = $2 AND last_seq = $5`

	futureEntriesSQL = `SELECT count(*) FROM public.audit_entries
WHERE organization_id = $1 AND business_unit_id = $2 AND "timestamp" >= $3`

	firstEntrySQL = `SELECT COALESCE(min("timestamp"), 0) FROM public.audit_entries
WHERE organization_id = $1 AND business_unit_id = $2`

	archivedAuditHeadSQL = `SELECT last_seq, last_hash, sealed_through FROM public.audit_chain_heads
WHERE organization_id = $1 AND business_unit_id = $2`

	archivedAIHeadSQL = `SELECT last_seq, last_hash FROM public.ai_audit_chain_heads
WHERE organization_id = $1 AND business_unit_id = $2`

	checkpointMismatchSQL = `SELECT count(*) FROM public.ai_audit_seals s
LEFT JOIN public.ai_audit_events e ON e.organization_id = s.organization_id
    AND e.business_unit_id = s.business_unit_id AND e.seq = s.to_seq
WHERE s.organization_id = $1 AND s.business_unit_id = $2 AND s.to_seq >= $3
    AND e.hash IS DISTINCT FROM s.head_hash`
)

var provenanceTables = []string{
	"audit_entries", "audit_seals", "audit_chain_heads", "auth_events",
	"ai_audit_events", "ai_audit_seals", "ai_audit_chain_heads",
}

type trailKeys struct {
	active *aiaudit.ChainKey
	keys   map[string]*aiaudit.ChainKey
}

func newTrailKeys(cfg *aiauditconfig.Config) *trailKeys {
	out := &trailKeys{keys: make(map[string]*aiaudit.ChainKey)}
	if cfg == nil {
		return out
	}
	for _, key := range cfg.Chain.Keys {
		out.keys[key.ID] = &aiaudit.ChainKey{ID: key.ID, Secret: []byte(key.Secret)}
	}
	if active, ok := cfg.Chain.ActiveKey(); ok {
		out.active = out.keys[active.ID]
	}
	return out
}

func (k *trailKeys) signSeal(seal *audit.Seal, origin audit.Origin) {
	keyID, secret := "", []byte(nil)
	if k.active != nil {
		keyID, secret = k.active.ID, k.active.Secret
	}
	seal.SignFrom(origin, keyID, secret)
}

func (k *trailKeys) checkSeal(seal *audit.Seal) (ok, missing bool) {
	var secret []byte
	if !seal.Unsigned() {
		key, found := k.keys[seal.HashKeyID]
		if !found {
			return false, true
		}
		secret = key.Secret
	}
	return hmac.Equal([]byte(seal.ComputeHash(secret)), []byte(seal.Hash)), false
}

func (k *trailKeys) eventKey(event *aiaudit.AIAuditEvent) (key *aiaudit.ChainKey, missing bool) {
	if !event.Keyed() {
		return nil, false
	}
	key, found := k.keys[event.HashKeyID]
	return key, !found
}

type trailRun struct {
	l      *load
	ctx    context.Context
	db     *keel.DB
	req    *archiveports.TrailRequest
	keys   *trailKeys
	tenant dbscope.Tenant
}

func (l *load) Trails(
	ctx context.Context,
	req *archiveports.TrailRequest,
) (*archiveports.TrailResult, error) {
	ctx = dbscope.WithSystem(ctx, trailsReason)
	db := l.repo.db.Keel(ctx)
	joined, done := keel.JoinTx(ctx, db, l.conn)
	defer done(false)
	run := &trailRun{
		l: l, ctx: joined, db: db, req: req, keys: l.repo.keys, tenant: l.tenant,
	}
	if err := run.refuseFutureEntries(); err != nil {
		return nil, err
	}
	result := new(archiveports.TrailResult)
	var (
		last *audit.Seal
		err  error
	)
	if result.AuditSeals, last, err = run.auditSeals(); err != nil {
		return nil, err
	}
	if result.Tail, err = run.sealTail(last); err != nil {
		return nil, err
	}
	if result.AIAudit, err = run.aiChain(); err != nil {
		return nil, err
	}
	if err = run.provenance(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (t *trailRun) refuseFutureEntries() error {
	var later int64
	if err := t.l.tx.QueryRow(t.ctx, futureEntriesSQL, t.tenant.OrganizationID.String(),
		t.tenant.BusinessUnitID.String(), t.req.At).Scan(&later); err != nil {
		return fmt.Errorf("check the imported audit log's dates: %w", err)
	}
	if later > 0 {
		return tenantarchive.Refuse(tenantarchive.RefusalManifest, reasonFutureEntries)
	}
	return nil
}

type linkCheck struct {
	links    int64
	missing  bool
	unsigned bool
	brokenAt int64
	gapped   bool
	latest   int64
}

func (c *linkCheck) source(restoring bool) archiveports.SourceCheck {
	switch {
	case c.brokenAt != 0:
		return archiveports.SourceNotVerified
	case c.missing, c.unsigned && !restoring:
		return archiveports.SourceNotVerifiable
	default:
		return archiveports.SourceVerified
	}
}

func (c *linkCheck) attestationRefusal(
	att *tenantarchive.TrailAttestation,
) archiveports.AttestationRefusal {
	switch {
	case att == nil:
		return archiveports.AttestationAbsent
	case !att.Verified:
		return archiveports.AttestationNotVerified
	case c.brokenAt != 0:
		return archiveports.AttestationBroken
	case c.links != att.Links:
		return archiveports.AttestationLinks
	case c.gapped:
		return archiveports.AttestationWindows
	default:
		return ""
	}
}

func attestationOf(att *tenantarchive.TrailAttestation) *archiveports.Attestation {
	if att == nil {
		return nil
	}
	return &archiveports.Attestation{
		SealedThrough: att.SealedThrough.Unix(),
		Verified:      att.Verified,
		Links:         att.Links,
	}
}

func established(
	source archiveports.SourceCheck,
	refused archiveports.AttestationRefusal,
) archiveports.Established {
	switch {
	case source == archiveports.SourceVerified:
		return archiveports.EstablishedHere
	case source == archiveports.SourceNotVerifiable && refused == "":
		return archiveports.EstablishedAtSource
	default:
		return archiveports.EstablishedNotVerified
	}
}

func (c *linkCheck) result() archiveports.ChainResult {
	return archiveports.ChainResult{
		Links:      c.links,
		BrokenAt:   c.brokenAt,
		KeyMissing: c.missing,
		Unsigned:   c.unsigned,
	}
}

type sealCheck struct {
	linkCheck
	last *audit.Seal
}

func (t *trailRun) eachSeal(fn func(*audit.Seal) error) error {
	s := auditdb.Seals
	after := int64(0)
	for {
		page, err := s.AllTenants().Where(
			s.OrganizationID.Eq(t.tenant.OrganizationID),
			s.BusinessUnitID.Eq(t.tenant.BusinessUnitID),
			s.Seq.Gt(after),
		).Order(s.Seq.Asc()).Limit(trailPage).List(t.ctx, t.db)
		if err != nil {
			return fmt.Errorf("read the imported audit seals: %w", err)
		}
		for _, seal := range page {
			if err = fn(seal); err != nil {
				return err
			}
			after = seal.Seq
		}
		if len(page) < trailPage {
			return nil
		}
	}
}

func (t *trailRun) checkSeals() (*sealCheck, error) {
	out := new(sealCheck)
	prev := ""
	err := t.eachSeal(func(seal *audit.Seal) error {
		if seal.FromTS >= seal.ToTS ||
			(out.last != nil && (seal.Seq != out.last.Seq+1 || seal.FromTS != out.last.ToTS)) {
			out.gapped = true
		}
		linked := (out.links == 0 && seal.Seq > 1) || seal.PrevHash == prev
		ok, missing := t.keys.checkSeal(seal)
		if seal.Unsigned() {
			out.unsigned = true
		}
		if missing {
			out.missing = true
		}
		if (!linked || (!missing && !ok)) && out.brokenAt == 0 {
			out.brokenAt = seal.Seq
		}
		prev = seal.Hash
		out.links++
		out.last = seal
		return nil
	})
	return out, err
}

func (t *trailRun) auditSeals() (archiveports.ChainResult, *audit.Seal, error) {
	checked, err := t.checkSeals()
	if err != nil {
		return archiveports.ChainResult{}, nil, err
	}
	result := checked.result()
	if checked.links == 0 {
		result.Outcome = archiveports.ChainEmpty
		return result, nil, nil
	}
	result.Source = checked.source(t.req.Restoring)
	att := t.req.Manifest.AuditAttestation()
	result.Attestation = attestationOf(att)
	if result.Source != archiveports.SourceVerified {
		if result.Refused, err = t.auditAttestation(checked, att); err != nil {
			return result, nil, err
		}
	}
	result.Established = established(result.Source, result.Refused)
	switch {
	case t.req.Restoring && result.Source == archiveports.SourceVerified:
		result.Outcome = archiveports.ChainKept
		result.KeyID = checked.last.HashKeyID
		return result, checked.last, nil
	case t.req.Restoring && result.Source == archiveports.SourceNotVerified:
		result.Outcome = archiveports.ChainBroken
		return result, checked.last, nil
	}
	origin := audit.OriginUnverified
	switch result.Established {
	case archiveports.EstablishedHere:
		origin = audit.OriginHere
	case archiveports.EstablishedAtSource:
		origin = audit.OriginSourceVerified
	case archiveports.EstablishedNotVerified:
	}
	return t.resealSeals(result, origin)
}

func (t *trailRun) auditAttestation(
	checked *sealCheck,
	att *tenantarchive.TrailAttestation,
) (archiveports.AttestationRefusal, error) {
	if refused := checked.attestationRefusal(att); refused != "" {
		return refused, nil
	}
	sealedThrough := att.SealedThrough.Unix()
	if checked.last.ToTS != sealedThrough {
		return archiveports.AttestationSealedThrough, nil
	}
	org, bu := t.tenant.OrganizationID.String(), t.tenant.BusinessUnitID.String()
	var later int64
	if err := t.l.tx.QueryRow(t.ctx, futureEntriesSQL, org, bu, sealedThrough).
		Scan(&later); err != nil {
		return "", fmt.Errorf("check the imported audit log against its attestation: %w", err)
	}
	if later > 0 {
		return archiveports.AttestationDatedAfter, nil
	}
	var (
		headSeq, headThrough int64
		headHash             string
	)
	err := t.l.tx.QueryRow(t.ctx, archivedAuditHeadSQL, org, bu).
		Scan(&headSeq, &headHash, &headThrough)
	if errors.Is(err, pgx.ErrNoRows) {
		return archiveports.AttestationUnsealedTail, nil
	}
	if err != nil {
		return "", fmt.Errorf("read the imported audit chain head: %w", err)
	}
	if headSeq != checked.last.Seq || headHash != checked.last.Hash ||
		headThrough != checked.last.ToTS {
		return archiveports.AttestationUnsealedTail, nil
	}
	return "", nil
}

type sealBatch struct {
	seqs     []int64
	prev     []string
	hashes   []string
	keyIDs   []string
	versions []int16
}

func (b *sealBatch) add(seal *audit.Seal) {
	b.seqs = append(b.seqs, seal.Seq)
	b.prev = append(b.prev, seal.PrevHash)
	b.hashes = append(b.hashes, seal.Hash)
	b.keyIDs = append(b.keyIDs, seal.HashKeyID)
	b.versions = append(b.versions, seal.HashVersion)
}

func (t *trailRun) resealSeals(
	result archiveports.ChainResult,
	origin audit.Origin,
) (archiveports.ChainResult, *audit.Seal, error) {
	prev := ""
	var last *audit.Seal
	batch := new(sealBatch)
	flush := func() error {
		if len(batch.seqs) == 0 {
			return nil
		}
		if _, err := t.l.tx.Exec(t.ctx, resealSealsSQL, t.tenant.OrganizationID.String(),
			t.tenant.BusinessUnitID.String(), t.req.At, batch.seqs, batch.prev, batch.hashes,
			batch.keyIDs, batch.versions); err != nil {
			return fmt.Errorf("sign the imported audit seals again: %w", err)
		}
		batch = new(sealBatch)
		t.l.heartbeat()
		return nil
	}
	first := true
	err := t.eachSeal(func(seal *audit.Seal) error {
		if !first || seal.Seq == 1 {
			seal.PrevHash = prev
		}
		first = false
		linkOrigin := origin.Weaker(seal.Origin())
		t.keys.signSeal(seal, linkOrigin)
		switch linkOrigin {
		case audit.OriginUnverified:
			result.Marked++
		case audit.OriginSourceVerified:
			result.Attested++
		case audit.OriginHere:
		}
		prev = seal.Hash
		last = seal
		batch.add(seal)
		if len(batch.seqs) >= trailPage {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		return result, nil, err
	}
	if err = t.moveAuditHead(last); err != nil {
		return result, nil, err
	}
	result.Outcome = archiveports.ChainResealed
	result.KeyID = last.HashKeyID
	return result, last, nil
}

func (t *trailRun) moveAuditHead(last *audit.Seal) error {
	if _, err := t.l.tx.Exec(t.ctx, auditHeadSQL, t.tenant.OrganizationID.String(),
		t.tenant.BusinessUnitID.String(), last.Seq, last.Hash, last.ToTS,
		last.HashKeyID); err != nil {
		return fmt.Errorf("move the audit chain head to the signed seals: %w", err)
	}
	return nil
}

func (t *trailRun) sealTail(last *audit.Seal) (*archiveports.TailSeal, error) {
	from := int64(0)
	next := &audit.Seal{
		OrganizationID: t.tenant.OrganizationID,
		BusinessUnitID: t.tenant.BusinessUnitID,
		Seq:            1,
		ToTS:           t.req.At,
		SealedAt:       t.req.At,
	}
	if last != nil {
		from = last.ToTS
		next.Seq = last.Seq + 1
		next.PrevHash = last.Hash
	} else if err := t.l.tx.QueryRow(t.ctx, firstEntrySQL, t.tenant.OrganizationID.String(),
		t.tenant.BusinessUnitID.String()).Scan(&from); err != nil {
		return nil, fmt.Errorf("find the imported audit log's first entry: %w", err)
	}
	if from <= 0 || from >= t.req.At {
		return nil, nil //nolint:nilnil // no tail: the source sealed everything it archived
	}
	digest, err := auditdigest.Compute(t.ctx, t.db, auditdigest.Window{
		OrganizationID: t.tenant.OrganizationID,
		BusinessUnitID: t.tenant.BusinessUnitID,
		From:           from,
		To:             t.req.At,
	})
	if err != nil {
		return nil, err
	}
	if digest.RowCount == 0 {
		return nil, nil //nolint:nilnil // no tail: no entry lies after the last imported seal
	}
	next.FromTS = from
	next.RowCount = digest.RowCount
	next.ClassDigests = digest.Classes
	t.keys.signSeal(next, audit.OriginUnverified)
	if err = auditdb.Seals.Create(t.ctx, t.db, next); err != nil {
		return nil, fmt.Errorf("seal the imported audit log's unsealed entries: %w", err)
	}
	if err = t.moveAuditHead(next); err != nil {
		return nil, err
	}
	return &archiveports.TailSeal{
		Seq:  next.Seq,
		From: from,
		To:   t.req.At,
		Rows: digest.RowCount,
	}, nil
}

func (t *trailRun) eachEvent(fn func(*aiaudit.AIAuditEvent) error) error {
	e := aiauditdb.AIAuditEvents
	after := int64(0)
	for {
		page, err := e.AllTenants().Where(
			e.OrganizationID.Eq(t.tenant.OrganizationID),
			e.BusinessUnitID.Eq(t.tenant.BusinessUnitID),
			e.Seq.Gt(after),
		).Order(e.Seq.Asc()).Limit(trailPage).List(t.ctx, t.db)
		if err != nil {
			return fmt.Errorf("read the imported AI audit chain: %w", err)
		}
		for _, event := range page {
			if err = fn(event); err != nil {
				return err
			}
			after = event.Seq
		}
		if len(page) < trailPage {
			return nil
		}
	}
}

type eventCheck struct {
	linkCheck
	first int64
	last  *aiaudit.AIAuditEvent
}

func (t *trailRun) checkEvents() (*eventCheck, error) {
	out := new(eventCheck)
	prev := aiaudit.GenesisHash
	err := t.eachEvent(func(event *aiaudit.AIAuditEvent) error {
		if out.last != nil && event.Seq != out.last.Seq+1 {
			out.gapped = true
		}
		out.latest = max(out.latest, event.RecordedAt)
		linked := (out.links == 0 && event.Seq > 1) || event.PrevHash == prev
		if !event.Keyed() {
			out.unsigned = true
		}
		key, missing := t.keys.eventKey(event)
		intact := linked
		if missing {
			out.missing = true
		} else if ok, verr := aiaudit.VerifyHash(event, key); verr != nil || !ok {
			intact = false
		}
		if !intact && out.brokenAt == 0 {
			out.brokenAt = event.Seq
		}
		prev = event.Hash
		if out.links == 0 {
			out.first = event.Seq
		}
		out.links++
		out.last = event
		return nil
	})
	return out, err
}

func (t *trailRun) aiChain() (archiveports.ChainResult, error) {
	checked, err := t.checkEvents()
	if err != nil {
		return archiveports.ChainResult{}, err
	}
	result := checked.result()
	if checked.links == 0 {
		result.Outcome = archiveports.ChainEmpty
		return result, nil
	}
	result.Source = checked.source(t.req.Restoring)
	att := t.req.Manifest.AIAuditAttestation()
	result.Attestation = attestationOf(att)
	if result.Source != archiveports.SourceVerified {
		if result.Refused, err = t.aiAttestation(checked, att); err != nil {
			return result, err
		}
	}
	result.Established = established(result.Source, result.Refused)
	switch {
	case t.req.Restoring && result.Source == archiveports.SourceVerified:
		result.Outcome = archiveports.ChainKept
		result.KeyID = checked.last.HashKeyID
		return result, nil
	case t.req.Restoring && result.Source == archiveports.SourceNotVerified:
		result.Outcome = archiveports.ChainBroken
		return result, nil
	}
	origin := aiaudit.OriginUnverified
	switch result.Established {
	case archiveports.EstablishedHere:
		origin = aiaudit.OriginHere
	case archiveports.EstablishedAtSource:
		origin = aiaudit.OriginSourceVerified
	case archiveports.EstablishedNotVerified:
	}
	return t.resealEvents(result, origin)
}

func (t *trailRun) aiAttestation(
	checked *eventCheck,
	att *tenantarchive.TrailAttestation,
) (archiveports.AttestationRefusal, error) {
	if refused := checked.attestationRefusal(att); refused != "" {
		return refused, nil
	}
	if checked.latest >= att.SealedThrough.Unix() {
		return archiveports.AttestationDatedAfter, nil
	}
	org, bu := t.tenant.OrganizationID.String(), t.tenant.BusinessUnitID.String()
	var mismatched int64
	if err := t.l.tx.QueryRow(t.ctx, checkpointMismatchSQL, org, bu, checked.first).
		Scan(&mismatched); err != nil {
		return "", fmt.Errorf("check the imported AI audit checkpoints: %w", err)
	}
	if mismatched > 0 {
		return archiveports.AttestationCheckpoints, nil
	}
	var (
		headSeq  int64
		headHash string
	)
	err := t.l.tx.QueryRow(t.ctx, archivedAIHeadSQL, org, bu).Scan(&headSeq, &headHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return archiveports.AttestationUnsealedTail, nil
	}
	if err != nil {
		return "", fmt.Errorf("read the imported AI audit chain head: %w", err)
	}
	if headSeq != checked.last.Seq || headHash != checked.last.Hash {
		return archiveports.AttestationUnsealedTail, nil
	}
	return "", nil
}

type eventBatch struct {
	ids      []string
	prev     []string
	hashes   []string
	keyIDs   []string
	versions []int16
}

func (b *eventBatch) add(event *aiaudit.AIAuditEvent) {
	b.ids = append(b.ids, event.ID.String())
	b.prev = append(b.prev, event.PrevHash)
	b.hashes = append(b.hashes, event.Hash)
	b.keyIDs = append(b.keyIDs, event.HashKeyID)
	b.versions = append(b.versions, event.HashVersion)
}

func (t *trailRun) resealEvents(
	result archiveports.ChainResult,
	origin aiaudit.Origin,
) (archiveports.ChainResult, error) {
	active := t.keys.active
	prev := aiaudit.GenesisHash
	var last *aiaudit.AIAuditEvent
	batch := new(eventBatch)
	flush := func() error {
		if len(batch.ids) == 0 {
			return nil
		}
		if _, err := t.l.tx.Exec(t.ctx, resealEventsSQL, t.tenant.OrganizationID.String(),
			t.tenant.BusinessUnitID.String(), t.req.At, batch.ids, batch.prev, batch.hashes,
			batch.keyIDs, batch.versions); err != nil {
			return fmt.Errorf("sign the imported AI audit chain again: %w", err)
		}
		batch = new(eventBatch)
		t.l.heartbeat()
		return nil
	}
	first := true
	err := t.eachEvent(func(event *aiaudit.AIAuditEvent) error {
		link := prev
		if first && event.Seq > 1 {
			link = event.PrevHash
		}
		first = false
		linkOrigin := origin.Weaker(event.Origin())
		switch linkOrigin {
		case aiaudit.OriginUnverified:
			result.Marked++
		case aiaudit.OriginSourceVerified:
			result.Attested++
		case aiaudit.OriginHere:
		}
		if serr := aiaudit.SealFrom(event, link, active, linkOrigin); serr != nil {
			return fmt.Errorf("sign AI audit row %d again: %w", event.Seq, serr)
		}
		prev = event.Hash
		last = event
		batch.add(event)
		if len(batch.ids) >= trailPage {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		return result, err
	}
	org, bu := t.tenant.OrganizationID.String(), t.tenant.BusinessUnitID.String()
	if _, err = t.l.tx.Exec(t.ctx, resealAISealsSQL, org, bu); err != nil {
		return result, fmt.Errorf("move the AI audit checkpoints to the signed chain: %w", err)
	}
	if _, err = t.l.tx.Exec(t.ctx, aiHeadSQL, org, bu, last.Hash, last.HashKeyID,
		last.Seq); err != nil {
		return result, fmt.Errorf("move the AI audit chain head to the signed chain: %w", err)
	}
	result.Outcome = archiveports.ChainResealed
	result.KeyID = last.HashKeyID
	return result, nil
}

func (t *trailRun) provenance(result *archiveports.TrailResult) error {
	m := t.req.Manifest
	action, comment := ActionImported, "The organization was made from a tenant archive."
	if t.req.Restoring {
		action, comment = ActionRestored, "The organization was restored from a tenant archive."
	}
	rows := make(map[string]any, len(provenanceTables))
	for _, name := range provenanceTables {
		if table, ok := m.Table(tenantarchive.QualifiedTable(name)); ok {
			rows[name] = table.Rows
		}
	}
	archive := map[string]any{
		"manifestSha256":   t.req.ManifestSHA256,
		"sourceInstanceId": m.InstanceID,
		"purpose":          string(m.Purpose),
		"takenAt":          m.TakenAt.UTC().Format(time.RFC3339),
		"releaseVersion":   m.ReleaseVersion,
		"schemaVersion":    m.SchemaVersion,
		"loadedAt":         time.Unix(t.req.At, 0).UTC().Format(time.RFC3339),
		"signedHereFrom":   t.req.At,
		"rows":             rows,
		"auditSeals":       chainMetadata(&result.AuditSeals, t.req.ManifestSHA256),
		"aiAuditChain":     chainMetadata(&result.AIAudit, t.req.ManifestSHA256),
	}
	if result.Tail != nil {
		archive["auditTail"] = map[string]any{
			"seq":    result.Tail.Seq,
			"from":   result.Tail.From,
			"to":     result.Tail.To,
			"rows":   result.Tail.Rows,
			"source": string(archiveports.SourceNotVerifiable),
		}
	}
	entry := &audit.Entry{
		OrganizationID: t.tenant.OrganizationID,
		BusinessUnitID: t.tenant.BusinessUnitID,
		Timestamp:      t.req.At,
		Resource:       permission.ResourceOrganization,
		ResourceID:     t.tenant.OrganizationID.String(),
		Operation:      permission.OpUpdate,
		Action:         action,
		Category:       audit.CategorySecurity,
		PrincipalType:  string(permissionports.PrincipalTypeSystem),
		PrincipalID:    permissionports.SystemPrincipalID,
		ActorName:      provenanceName,
		Comment:        comment,
		Critical:       true,
		Metadata:       map[string]any{"archive": archive},
	}
	if err := auditdb.Entries.Create(t.ctx, t.db, entry); err != nil {
		return fmt.Errorf("record where the tenant's history came from: %w", err)
	}
	return nil
}

func chainMetadata(c *archiveports.ChainResult, manifestSHA256 string) map[string]any {
	out := map[string]any{"outcome": string(c.Outcome), "links": c.Links}
	if c.Source != "" {
		out["source"] = string(c.Source)
		out["sourceVerified"] = c.Source == archiveports.SourceVerified
	}
	if c.Established != "" {
		out["established"] = string(c.Established)
	}
	if c.Attestation != nil {
		out["attestation"] = map[string]any{
			"sealedThrough":  time.Unix(c.Attestation.SealedThrough, 0).UTC().Format(time.RFC3339),
			"verified":       c.Attestation.Verified,
			"links":          c.Attestation.Links,
			"manifestSha256": manifestSHA256,
		}
	}
	if c.Refused != "" {
		out["attestationRefused"] = string(c.Refused)
	}
	if c.Attested > 0 {
		out["markedVerifiedAtSource"] = c.Attested
	}
	if c.KeyMissing {
		out["keyMissing"] = true
	}
	if c.Unsigned {
		out["unsigned"] = true
	}
	if c.Marked > 0 {
		out["markedNotVerified"] = c.Marked
	}
	if c.KeyID != "" {
		out["keyId"] = c.KeyID
	}
	if c.BrokenAt != 0 {
		out["brokenAt"] = c.BrokenAt
	}
	return out
}
