package realtimebroker

import (
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
)

const keyPrefix = "trenova:realtime:"

const (
	fieldTenant     = "t"
	fieldAudience   = "a"
	fieldEvent      = "e"
	fieldScope      = "s"
	fieldPayload    = "p"
	fieldPortal     = "r"
	fieldEphemeral  = "x"
	fieldConnection = "c"
	fieldAction     = "k"
)

const (
	connFieldUser   = "u"
	connFieldOrg    = "o"
	connFieldBU     = "b"
	connFieldName   = "n"
	connFieldPortal = "p"
)

func tenantKey(orgID, buID pulid.ID) string {
	return orgID.String() + ":" + buID.String()
}

func shardFor(tenant string, shards int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(tenant))
	return int(h.Sum32() % uint32(shards)) //nolint:gosec // shards is bounded by config validation
}

func streamKey(shard int) string {
	return keyPrefix + "stream:" + strconv.Itoa(shard)
}

func connectionKey(connectionID string) string {
	return keyPrefix + "conn:" + connectionID
}

func connectionScopesKey(connectionID string) string {
	return keyPrefix + "conn:" + connectionID + ":scopes"
}

func presenceExpiryKey(tenant, scope string) string {
	return keyPrefix + "presence:" + tenant + ":" + scope + ":exp"
}

func presenceDataKey(tenant, scope string) string {
	return keyPrefix + "presence:" + tenant + ":" + scope + ":data"
}

func presenceIndexMember(tenant, scope string) string {
	return tenant + "|" + scope
}

func splitPresenceIndexMember(member string) (tenant, scope string, ok bool) {
	return strings.Cut(member, "|")
}

const (
	presenceIndexKey = keyPrefix + "presence:index"
	sweepLockKey     = keyPrefix + "presence:sweep-lock"
)

func throttleKey(key string) string {
	return keyPrefix + "throttle:" + key
}

// cursor is a position a reader can resume from: the shard its tenant writes
// to and the entry id inside that shard's stream. Written to the wire as
// "shard.ms-seq".
type cursor struct {
	shard int
	id    streamID
}

func formatCursor(shard int, id string) string {
	return strconv.Itoa(shard) + "." + id
}

func parseCursor(raw string) (cursor, bool) {
	shardText, idText, found := strings.Cut(raw, ".")
	if !found {
		return cursor{}, false
	}

	shard, err := strconv.Atoi(shardText)
	if err != nil || shard < 0 {
		return cursor{}, false
	}

	id, ok := parseStreamID(idText)
	if !ok {
		return cursor{}, false
	}

	return cursor{shard: shard, id: id}, true
}

// streamID is a Redis stream entry id. Ids are compared numerically: two
// string ids of different widths do not sort the way the stream does.
type streamID struct {
	ms  uint64
	seq uint64
}

func parseStreamID(raw string) (streamID, bool) {
	msText, seqText, found := strings.Cut(raw, "-")
	if !found {
		return streamID{}, false
	}

	ms, err := strconv.ParseUint(msText, 10, 64)
	if err != nil {
		return streamID{}, false
	}

	seq, err := strconv.ParseUint(seqText, 10, 64)
	if err != nil {
		return streamID{}, false
	}

	return streamID{ms: ms, seq: seq}, true
}

func (id streamID) String() string {
	return strconv.FormatUint(id.ms, 10) + "-" + strconv.FormatUint(id.seq, 10)
}

func (id streamID) IsZero() bool {
	return id.ms == 0 && id.seq == 0
}

func (id streamID) Less(other streamID) bool {
	if id.ms != other.ms {
		return id.ms < other.ms
	}
	return id.seq < other.seq
}
