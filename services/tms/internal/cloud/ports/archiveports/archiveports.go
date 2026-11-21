package archiveports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/pkg/dbscope"
)

var ErrMissing = errors.New("the archive location holds no object at that key")

type Bucket interface {
	Put(ctx context.Context, key string, body io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	List(ctx context.Context, prefix string, fn func(key string, size int64) error) error
	Children(ctx context.Context, prefix string) ([]string, error)
	DeletePrefix(ctx context.Context, prefix string) (int64, error)
}

type Buckets interface {
	Open(location *tenantarchive.Location, credentials *tenantarchive.Credentials) (Bucket, error)
	CheckEndpoint(endpoint string) error
}

type Dangling struct {
	Constraint string `json:"constraint" db:"constraint_name"`
	Table      string `json:"table"      db:"table_name"`
	Columns    string `json:"columns"    db:"column_names"`
	Rows       int64  `json:"rows"       db:"row_count"`
}

type TableCount struct {
	Table string `json:"table" db:"table_name"`
	Rows  int64  `json:"rows"  db:"row_count"`
}

type Reference struct {
	Table string
	Key   []string
	Rows  map[string][]string
}

type ObjectColumn struct {
	Table  string
	Column string
}

type Snapshot interface {
	Plan() *archiveplan.Plan
	Tenant() dbscope.Tenant
	OrganizationName() string
	CopyOut(ctx context.Context, table *archiveplan.PlannedTable, dst io.Writer) error
	References(ctx context.Context) ([]Reference, error)
	Outside(ctx context.Context) ([]Dangling, error)
	Trails() *TrailFacts
	Close(ctx context.Context)
}

type SnapshotRequest struct {
	Tenant   dbscope.Tenant
	Purpose  tenantarchive.Purpose
	Envelope map[string][]string
	At       int64
}

type ChainFacts struct {
	Links         int64
	FirstSeq      int64
	LastSeq       int64
	LastHash      string
	SealedHere    int64
	SealedThrough int64
}

func (c *ChainFacts) Held() int64 {
	return c.Links - c.SealedHere
}

type TrailFacts struct {
	Audit   *ChainFacts
	AIAudit *ChainFacts
}

type TrailRequest struct {
	Restoring      bool
	ManifestSHA256 string
	Manifest       *tenantarchive.Manifest
	At             int64
}

type ChainOutcome string

const (
	ChainEmpty    = ChainOutcome("empty")
	ChainKept     = ChainOutcome("kept")
	ChainResealed = ChainOutcome("resealed")
	ChainBroken   = ChainOutcome("broken")
)

type SourceCheck string

const (
	SourceVerified      = SourceCheck("verified")
	SourceNotVerified   = SourceCheck("not_verified")
	SourceNotVerifiable = SourceCheck("not_verifiable")
)

type Established string

const (
	EstablishedHere        = Established("verified_here")
	EstablishedAtSource    = Established("verified_at_source")
	EstablishedNotVerified = Established("not_verified")
)

type AttestationRefusal string

const (
	AttestationAbsent        = AttestationRefusal("absent")
	AttestationNotVerified   = AttestationRefusal("not_verified_at_source")
	AttestationLinks         = AttestationRefusal("links")
	AttestationBroken        = AttestationRefusal("broken")
	AttestationWindows       = AttestationRefusal("windows")
	AttestationSealedThrough = AttestationRefusal("sealed_through")
	AttestationDatedAfter    = AttestationRefusal("dated_after")
	AttestationUnsealedTail  = AttestationRefusal("unsealed_tail")
	AttestationCheckpoints   = AttestationRefusal("checkpoints")
)

type Attestation struct {
	SealedThrough int64 `json:"sealedThrough"`
	Verified      bool  `json:"verified"`
	Links         int64 `json:"links"`
}

type ChainResult struct {
	Outcome     ChainOutcome       `json:"outcome"`
	Links       int64              `json:"links"`
	KeyID       string             `json:"keyId,omitempty"`
	BrokenAt    int64              `json:"brokenAt,omitempty"`
	Source      SourceCheck        `json:"source,omitempty"`
	Established Established        `json:"established,omitempty"`
	KeyMissing  bool               `json:"keyMissing,omitempty"`
	Unsigned    bool               `json:"unsigned,omitempty"`
	Marked      int64              `json:"marked,omitempty"`
	Attested    int64              `json:"attested,omitempty"`
	Attestation *Attestation       `json:"attestation,omitempty"`
	Refused     AttestationRefusal `json:"attestationRefused,omitempty"`
}

func (c *ChainResult) Unverified() bool {
	return c.Marked > 0 || c.Outcome == ChainBroken
}

type TailSeal struct {
	Seq  int64 `json:"seq"`
	From int64 `json:"from"`
	To   int64 `json:"to"`
	Rows int64 `json:"rows"`
}

type TrailResult struct {
	AuditSeals ChainResult `json:"auditSeals"`
	AIAudit    ChainResult `json:"aiAudit"`
	Tail       *TailSeal   `json:"tail,omitempty"`
}

const maxRestoreNote = 500

func (r *TrailResult) RestoreNote() string {
	if r == nil {
		return ""
	}
	notes := make([]string, 0, 3)
	for _, chain := range []struct {
		name   string
		result *ChainResult
	}{{"audit log", &r.AuditSeals}, {"AI audit trail", &r.AIAudit}} {
		switch {
		case chain.result.Outcome == ChainBroken:
			notes = append(notes, fmt.Sprintf(
				"the %s does not verify from link %d and was restored as it was, so its verification reports the break",
				chain.name,
				chain.result.BrokenAt,
			))
		case chain.result.Marked > 0:
			notes = append(notes, fmt.Sprintf(
				"the %s names a key this server does not hold, so its %d links were signed again here and are marked as not verified at their source",
				chain.name,
				chain.result.Marked,
			))
		case chain.result.Attested > 0:
			notes = append(notes, fmt.Sprintf(
				"the %s names a key this server does not hold, so its %d links were signed again here and are marked as verified at their source, as the archive states",
				chain.name,
				chain.result.Attested,
			))
		}
	}
	if r.Tail != nil {
		notes = append(notes, fmt.Sprintf(
			"%d audit log entries written after the archive's last seal are marked as not verified at their source",
			r.Tail.Rows,
		))
	}
	if len(notes) == 0 {
		return ""
	}
	note := "Restored, but " + strings.Join(notes, "; ") + "."
	if len(note) > maxRestoreNote {
		note = note[:maxRestoreNote]
	}
	return note
}

type LoadRequest struct {
	Tenant   dbscope.Tenant
	Manifest *tenantarchive.Manifest
	Envelope map[string][]string
	Beat     func()
}

type Load interface {
	Plan() *archiveplan.Plan
	CopyIn(ctx context.Context, table *archiveplan.PlannedTable, src io.Reader) (int64, error)
	Finish(ctx context.Context) error
	Counts(ctx context.Context) ([]TableCount, error)
	Dangling(ctx context.Context) ([]Dangling, error)
	MarkForeignRecoveryCodes(ctx context.Context) (int64, error)
	Trails(ctx context.Context, req *TrailRequest) (*TrailResult, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context)
}

type Rows interface {
	SchemaVersion() string
	OpenSnapshot(ctx context.Context, req *SnapshotRequest) (Snapshot, error)
	PlanFor(
		ctx context.Context,
		purpose tenantarchive.Purpose,
		envelope map[string][]string,
	) (*archiveplan.Plan, error)
	TenantExists(ctx context.Context, tenant dbscope.Tenant) (bool, error)
	Holding(ctx context.Context, plan *archiveplan.Plan, tenant dbscope.Tenant) ([]string, error)
	ExistingKeys(
		ctx context.Context,
		table *archiveplan.PlannedTable,
		keys [][]string,
	) (int64, error)
	Resolve(
		ctx context.Context,
		table string,
		key []string,
		values [][]string,
	) (map[string]string, error)
	TakenIdentities(ctx context.Context, emails, usernames []string) (bool, error)
	TakenSlugs(ctx context.Context, slugs []string) (bool, error)
	EnsureAuditPartitions(ctx context.Context, from int64) error
	OpenLoad(ctx context.Context, req *LoadRequest) (Load, error)
}
