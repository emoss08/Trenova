package tenantcopyrepository

import (
	"context"
	"crypto/hmac"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit/aiauditdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/pkg/dbscope"
)

const (
	trailsReason = "sign an imported tenant's audit seals and AI audit chain under this install's key inside the archive load, and record where its history came from"
	trailPage    = 1_000

	ActionImported = "organization.imported"
	ActionRestored = "organization.restored"
	provenanceName = "Trenova"

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

	auditHeadSQL = `UPDATE public.audit_chain_heads SET
    last_hash = $3, hash_key_id = NULLIF($4, ''), last_verified_seq = 0, last_verified_at = NULL,
    last_verification_status = NULL, last_verification_failed_seq = NULL,
    last_verification_detail = NULL, verify_cursor_seq = 0
WHERE organization_id = $1 AND business_unit_id = $2 AND last_seq = $5`

	aiHeadSQL = `UPDATE public.ai_audit_chain_heads SET
    last_hash = $3, hash_key_id = NULLIF($4, ''), last_verified_seq = 0, last_verified_at = NULL,
    last_verification_status = NULL, last_verification_failed_seq = NULL,
    last_verification_detail = NULL
WHERE organization_id = $1 AND business_unit_id = $2 AND last_seq = $5`
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

func (k *trailKeys) signSeal(seal *audit.Seal) {
	if k.active == nil {
		seal.Sign("", nil)
		return
	}
	seal.Sign(k.active.ID, k.active.Secret)
}

func (k *trailKeys) checkSeal(seal *audit.Seal) (ok, missing bool) {
	var secret []byte
	if seal.HashVersion != audit.SealHashVersionUnsigned {
		key, found := k.keys[seal.HashKeyID]
		if !found {
			return false, true
		}
		secret = key.Secret
	}
	return hmac.Equal([]byte(seal.ComputeHash(secret)), []byte(seal.Hash)), false
}

func (k *trailKeys) eventKey(event *aiaudit.AIAuditEvent) (key *aiaudit.ChainKey, missing bool) {
	if event.HashVersion != aiaudit.HashVersionSigned {
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
	result := new(archiveports.TrailResult)
	var err error
	if result.AuditSeals, err = run.auditSeals(); err != nil {
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

type sealCheck struct {
	links    int64
	missing  bool
	brokenAt int64
	last     *audit.Seal
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
		linked := (out.links == 0 && seal.Seq > 1) || seal.PrevHash == prev
		ok, missing := t.keys.checkSeal(seal)
		switch {
		case missing:
			out.missing = true
		case (!ok || !linked) && out.brokenAt == 0:
			out.brokenAt = seal.Seq
		}
		prev = seal.Hash
		out.links++
		out.last = seal
		return nil
	})
	return out, err
}

func (t *trailRun) auditSeals() (archiveports.ChainResult, error) {
	checked, err := t.checkSeals()
	if err != nil {
		return archiveports.ChainResult{}, err
	}
	result := archiveports.ChainResult{Links: checked.links}
	switch {
	case checked.links == 0:
		result.Outcome = archiveports.ChainEmpty
		return result, nil
	case t.req.Restoring && !checked.missing && checked.brokenAt == 0:
		result.Outcome = archiveports.ChainKept
		result.KeyID = checked.last.HashKeyID
		return result, nil
	case t.req.Restoring && checked.brokenAt != 0:
		result.Outcome = archiveports.ChainBroken
		result.BrokenAt = checked.brokenAt
		return result, nil
	}
	return t.resealSeals(result)
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
) (archiveports.ChainResult, error) {
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
		t.keys.signSeal(seal)
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
		return result, err
	}
	if _, err = t.l.tx.Exec(t.ctx, auditHeadSQL, t.tenant.OrganizationID.String(),
		t.tenant.BusinessUnitID.String(), last.Hash, last.HashKeyID, last.Seq); err != nil {
		return result, fmt.Errorf("move the audit chain head to the signed seals: %w", err)
	}
	result.Outcome = archiveports.ChainResealed
	result.KeyID = last.HashKeyID
	return result, nil
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
	links    int64
	missing  bool
	brokenAt int64
	last     *aiaudit.AIAuditEvent
}

func (t *trailRun) checkEvents() (*eventCheck, error) {
	out := new(eventCheck)
	prev := aiaudit.GenesisHash
	err := t.eachEvent(func(event *aiaudit.AIAuditEvent) error {
		linked := (out.links == 0 && event.Seq > 1) || event.PrevHash == prev
		key, missing := t.keys.eventKey(event)
		if missing {
			out.missing = true
		} else if ok, verr := aiaudit.VerifyHash(event, key); (verr != nil || !ok || !linked) &&
			out.brokenAt == 0 {
			out.brokenAt = event.Seq
		}
		prev = event.Hash
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
	result := archiveports.ChainResult{Links: checked.links}
	switch {
	case checked.links == 0:
		result.Outcome = archiveports.ChainEmpty
		return result, nil
	case t.req.Restoring && !checked.missing && checked.brokenAt == 0:
		result.Outcome = archiveports.ChainKept
		result.KeyID = checked.last.HashKeyID
		return result, nil
	case t.req.Restoring && checked.brokenAt != 0:
		result.Outcome = archiveports.ChainBroken
		result.BrokenAt = checked.brokenAt
		return result, nil
	}
	return t.resealEvents(result)
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
		if serr := aiaudit.Seal(event, link, active); serr != nil {
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
		Metadata: map[string]any{
			"archive": map[string]any{
				"manifestSha256":   t.req.ManifestSHA256,
				"sourceInstanceId": m.InstanceID,
				"purpose":          string(m.Purpose),
				"takenAt":          m.TakenAt.UTC().Format(time.RFC3339),
				"releaseVersion":   m.ReleaseVersion,
				"schemaVersion":    m.SchemaVersion,
				"loadedAt":         time.Unix(t.req.At, 0).UTC().Format(time.RFC3339),
				"rows":             rows,
				"auditSeals":       chainMetadata(&result.AuditSeals),
				"aiAuditChain":     chainMetadata(&result.AIAudit),
			},
		},
	}
	if err := auditdb.Entries.Create(t.ctx, t.db, entry); err != nil {
		return fmt.Errorf("record where the tenant's history came from: %w", err)
	}
	return nil
}

func chainMetadata(c *archiveports.ChainResult) map[string]any {
	out := map[string]any{"outcome": string(c.Outcome), "links": c.Links}
	if c.KeyID != "" {
		out["keyId"] = c.KeyID
	}
	if c.BrokenAt != 0 {
		out["brokenAt"] = c.BrokenAt
	}
	return out
}
