package detention

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*DetentionEvidence)(nil)

// GenesisHash anchors the first link of every occurrence's evidence chain.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// DetentionEvidence is one append-only fact about an occurrence, hash-chained
// to its predecessor. The carrier's structural credibility problem in a dispute
// is "you edited the arrival time after the fact" — chaining makes any edit
// provable rather than deniable.
type DetentionEvidence struct {
	bun.BaseModel `bun:"table:detention_evidence,alias:dte" json:"-"`

	ID                    pulid.ID       `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID       `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID        pulid.ID       `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	DetentionOccurrenceID pulid.ID       `json:"detentionOccurrenceId" bun:"detention_occurrence_id,type:VARCHAR(100),notnull"`
	Sequence              int32          `json:"sequence"              bun:"sequence,type:INTEGER,notnull,default:0"`
	Kind                  EvidenceKind   `json:"kind"                  bun:"kind,type:detention_evidence_kind_enum,notnull"`
	Source                EvidenceSource `json:"source"                bun:"source,type:detention_evidence_source_enum,notnull"`
	Summary               string         `json:"summary"               bun:"summary,type:TEXT,notnull"`
	ObservedAt            int64          `json:"observedAt"            bun:"observed_at,type:BIGINT,notnull"`
	RecordedAt            int64          `json:"recordedAt"            bun:"recorded_at,type:BIGINT,notnull"`
	RecordedByID          *pulid.ID      `json:"recordedById"          bun:"recorded_by_id,type:VARCHAR(100),nullzero"`
	DocumentID            *pulid.ID      `json:"documentId"            bun:"document_id,type:VARCHAR(100),nullzero"`
	Payload               map[string]any `json:"payload"               bun:"payload,type:JSONB,nullzero"`
	PrevHash              string         `json:"prevHash"              bun:"prev_hash,type:VARCHAR(64),notnull"`
	Hash                  string         `json:"hash"                  bun:"hash,type:VARCHAR(64),notnull"`
	CreatedAt             int64          `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// ComputeHash derives the chain link from the immutable fields of this record.
// Field order is fixed and values are length-delimited so no combination of
// inputs can collide by concatenation.
func (e *DetentionEvidence) ComputeHash() string {
	var b strings.Builder

	write := func(v string) {
		b.WriteString(strconv.Itoa(len(v)))
		b.WriteString(":")
		b.WriteString(v)
		b.WriteString("|")
	}

	write(e.PrevHash)
	write(e.DetentionOccurrenceID.String())
	write(strconv.Itoa(int(e.Sequence)))
	write(string(e.Kind))
	write(string(e.Source))
	write(e.Summary)
	write(strconv.FormatInt(e.ObservedAt, 10))
	write(strconv.FormatInt(e.RecordedAt, 10))

	if e.RecordedByID != nil {
		write(e.RecordedByID.String())
	} else {
		write("")
	}

	if e.DocumentID != nil {
		write(e.DocumentID.String())
	} else {
		write("")
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// Seal fixes the hash for this link. Callers must set PrevHash to the current
// chain head first.
func (e *DetentionEvidence) Seal() {
	e.Hash = e.ComputeHash()
}

// Verify reports whether the stored hash still matches the record's contents.
func (e *DetentionEvidence) Verify() bool {
	return e.Hash == e.ComputeHash()
}

// VerifyChain walks an ordered evidence list and returns the index of the first
// broken link, or -1 when the chain is intact.
func VerifyChain(entries []*DetentionEvidence) int {
	prev := GenesisHash

	for i, entry := range entries {
		if entry == nil {
			return i
		}
		if entry.PrevHash != prev {
			return i
		}
		if !entry.Verify() {
			return i
		}
		prev = entry.Hash
	}

	return -1
}

func (e *DetentionEvidence) Validate(multiErr *errortypes.MultiError) {
	if e.DetentionOccurrenceID.IsNil() {
		multiErr.Add("detentionOccurrenceId", errortypes.ErrRequired,
			"Detention occurrence is required")
	}
	if e.Summary == "" {
		multiErr.Add("summary", errortypes.ErrRequired, "Summary is required")
	}
	if _, err := EvidenceKindFromString(string(e.Kind)); err != nil {
		multiErr.Add("kind", errortypes.ErrInvalid, "Evidence kind is invalid")
	}
	if _, err := EvidenceSourceFromString(string(e.Source)); err != nil {
		multiErr.Add("source", errortypes.ErrInvalid, "Evidence source is invalid")
	}
}

func (e *DetentionEvidence) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("dte_")
		}
		if e.RecordedAt == 0 {
			e.RecordedAt = now
		}
		if e.PrevHash == "" {
			e.PrevHash = GenesisHash
		}
		if e.Hash == "" {
			e.Seal()
		}
		e.CreatedAt = now
	}

	return nil
}
