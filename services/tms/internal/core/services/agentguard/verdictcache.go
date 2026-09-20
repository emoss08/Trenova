package agentguard

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	lru "github.com/hashicorp/golang-lru/v2"
)

// verdictCacheSize bounds what the guard remembers. Each entry is a category
// and a short reason against a hash, so a few thousand costs very little and
// covers a busy day of one organization's questions.
const verdictCacheSize = 4096

/*
Remembering a verdict the classifier already gave.

Classification is a pure function of the text: the same question asked twice
gets the same answer, and asking a model again costs a round trip plus the
~900 tokens of classifier prompt that go with every call. In operations the
same question is asked constantly — the transcripts behind this change show
one question asked three times in a few minutes, each one paying full price.

Only a verdict the classifier actually produced is stored. A failure is never
cached: the fallback to deterministic rules is a degraded state, and freezing
it would turn a blip into an outage that outlives it.

The key includes the organization. Verdicts do not currently vary by tenant,
but the cache would be the wrong place to discover that they had started to.
*/
type verdictCache struct {
	entries *lru.Cache[string, ClassifierResult]
}

func newVerdictCache() *verdictCache {
	entries, err := lru.New[string, ClassifierResult](verdictCacheSize)
	if err != nil {
		// Only a non-positive size can fail, and the size is a constant above.
		return &verdictCache{}
	}

	return &verdictCache{entries: entries}
}

func (c *verdictCache) get(orgID pulid.ID, input string) (ClassifierResult, bool) {
	if c == nil || c.entries == nil {
		return ClassifierResult{}, false
	}

	return c.entries.Get(verdictKey(orgID, input))
}

func (c *verdictCache) put(orgID pulid.ID, input string, result ClassifierResult) {
	if c == nil || c.entries == nil {
		return
	}

	c.entries.Add(verdictKey(orgID, input), result)
}

// verdictKey hashes the message rather than keying on it directly, so a bounded
// cache cannot be filled with the text of what people asked. Case and
// surrounding whitespace are normalised because they change nothing about what
// a request is asking for.
func verdictKey(orgID pulid.ID, input string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(input))))

	return orgID.String() + ":" + hex.EncodeToString(sum[:])
}
