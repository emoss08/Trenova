package tenantcopyrepository

import (
	"slices"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var resealColumns = []string{
	"source_prev_hash", "source_hash", "source_hash_key_id", "source_hash_version", "resealed_at",
}

func keysOf(row map[string]any) []string {
	out := make([]string, 0, len(row))
	for k := range row {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func modelColumns(names ...[]string) []string {
	out := make([]string, 0)
	for _, list := range names {
		out = append(out, list...)
	}
	slices.Sort(out)
	return out
}

func TestTheArchivesOwnRowsNameEveryColumnTheirModelsHave(t *testing.T) {
	t.Parallel()
	seals := make([]string, 0)
	for _, c := range auditdb.SealsSchema().Columns {
		seals = append(seals, c.Name)
	}
	heads := make([]string, 0)
	for _, c := range auditdb.ChainHeadsSchema().Columns {
		heads = append(heads, c.Name)
	}
	seal := &audit.Seal{OrganizationID: "org", BusinessUnitID: "bu", Seq: 3}
	assert.Equal(t, modelColumns(seals, resealColumns), keysOf(tailSealRow(seal)),
		"a column added to audit_seals needs a value in the archive's last seal")
	assert.Equal(t, modelColumns(heads), keysOf(newHeadRow(seal, 10)),
		"a column added to audit_chain_heads needs a value in the archive's new head")
}

func TestARowThatDoesNotMatchItsTableIsRefusedNotShifted(t *testing.T) {
	t.Parallel()
	table := &archiveplan.PlannedTable{Name: "audit_seals", Columns: []string{"a", "b", "c"}}

	_, err := tailRow(table, map[string]any{"a": 1, "b": 2})
	require.ErrorIs(t, err, ErrTailRowShape)
	assert.Contains(t, err.Error(), "[c]")

	_, err = tailRow(table, map[string]any{"a": 1, "b": 2, "c": 3, "d": 4})
	require.ErrorIs(t, err, ErrTailRowShape)
	assert.Contains(t, err.Error(), "[d]")

	body, err := tailRow(table, map[string]any{"a": 1, "b": nil, "c": "x"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1,"b":null,"c":"x"}`, body)
}

func TestHostileTextIsCarriedAsDataAndComesBackUnchanged(t *testing.T) {
	t.Parallel()
	hostile := []string{
		`'); DROP TABLE audit_seals; --`,
		`\'\\x00\n\t"quoted"`,
		`E'\\'`,
		"$$ $tag$ ::jsonb",
		"Übermäßig — 運送 🚚 \u2028\u2029",
	}
	columns := modelColumns(keysOf(tailSealRow(&audit.Seal{})))
	table := &archiveplan.PlannedTable{Name: archiveSealsTable, Columns: columns}
	for _, text := range hostile {
		seal := &audit.Seal{
			OrganizationID: "org",
			BusinessUnitID: "bu",
			Seq:            1<<53 + 1,
			PrevHash:       text,
			Hash:           text,
			HashKeyID:      text,
			ClassDigests:   map[string]audit.ClassDigest{text: {Count: 1, Digest: text}},
		}
		body, err := tailRow(table, tailSealRow(seal))
		require.NoError(t, err)
		var back struct {
			Seq          int64                        `json:"seq"`
			PrevHash     string                       `json:"prev_hash"`
			HashKeyID    string                       `json:"hash_key_id"`
			ClassDigests map[string]audit.ClassDigest `json:"class_digests"`
		}
		require.NoError(t, sonic.ConfigStd.UnmarshalFromString(body, &back))
		assert.Equal(t, text, back.PrevHash)
		assert.Equal(t, text, back.HashKeyID)
		assert.Equal(t, seal.ClassDigests, back.ClassDigests)
		assert.Equal(t, seal.Seq, back.Seq, "a large number keeps every digit")
	}
}

func TestAKeyIDThatNeverVerifiesIsAConflictAndOneThatSometimesDoesIsABreak(t *testing.T) {
	t.Parallel()
	conflict := new(linkCheck)
	conflict.signed("k", 1, false)
	conflict.signed("k", 2, false)
	conflict.settleKeys()
	assert.True(t, conflict.missing)
	assert.True(t, conflict.conflict)
	assert.Zero(t, conflict.brokenAt)
	assert.Equal(t, archiveports.SourceNotVerifiable, conflict.source(false))

	tampered := new(linkCheck)
	tampered.signed("k", 1, true)
	tampered.signed("k", 2, false)
	tampered.signed("k", 3, true)
	tampered.settleKeys()
	assert.False(t, tampered.conflict)
	assert.Equal(t, int64(2), tampered.brokenAt)
	assert.Equal(t, archiveports.SourceNotVerified, tampered.source(false))

	firstTampered := new(linkCheck)
	firstTampered.signed("k", 1, false)
	firstTampered.signed("k", 2, true)
	firstTampered.settleKeys()
	assert.Equal(t, int64(1), firstTampered.brokenAt,
		"a first link that fails under a key that verifies later is a break, not a conflict")

	rotated := new(linkCheck)
	rotated.signed("old", 1, true)
	rotated.signed("new", 2, false)
	rotated.breakAt(5)
	rotated.settleKeys()
	assert.True(t, rotated.conflict)
	assert.Equal(t, int64(5), rotated.brokenAt, "a broken link stays broken whatever the keys")
}

func TestAChainVerifiedHereStillAgreesWithWhatItsArchiveStates(t *testing.T) {
	t.Parallel()
	sealed := time.Unix(1000, 0)
	through := func(att *tenantarchive.TrailAttestation) archiveports.AttestationRefusal {
		if att.SealedThrough.Unix() != 1000 {
			return archiveports.AttestationSealedThrough
		}
		return ""
	}
	held := &linkCheck{links: 4}
	assert.Empty(t, held.heldRefusal(nil, through), "an archive that states nothing is not compared")
	assert.Empty(t, held.heldRefusal(
		&tenantarchive.TrailAttestation{Links: 4, SealedThrough: sealed}, through))
	assert.Empty(t, held.heldRefusal(
		&tenantarchive.TrailAttestation{Links: 4, SealedThrough: sealed, Verified: false}, through),
		"a source that could not verify is outweighed by this server verifying")
	truncated := held.heldRefusal(
		&tenantarchive.TrailAttestation{Links: 6, SealedThrough: sealed}, through)
	assert.Equal(t, archiveports.AttestationLinks, truncated)
	assert.Equal(t, archiveports.AttestationSealedThrough, held.heldRefusal(
		&tenantarchive.TrailAttestation{Links: 4, SealedThrough: sealed.Add(time.Hour)}, through))

	assert.Equal(t, archiveports.EstablishedHere, established(archiveports.SourceVerified, ""))
	assert.Equal(t, archiveports.EstablishedNotVerified,
		established(archiveports.SourceVerified, truncated))
	assert.Equal(t, archiveports.EstablishedAtSource,
		established(archiveports.SourceNotVerifiable, ""))
}
