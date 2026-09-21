package agentguard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
)

// verdictCacheSize bounds what one replica remembers on its own. Each entry
// is a category and a short reason against a hash, so a few thousand costs
// very little and covers a busy day of one organization's questions.
const verdictCacheSize = 4096

// sharedTimeout bounds a trip to the shared store. A verdict is worth a few
// milliseconds of waiting and no more: past this the classifier is faster
// than the cache, and a hung store must never hold a question hostage.
const sharedTimeout = 250 * time.Millisecond

/*
Remembering a verdict the classifier already gave.

Classification is a pure function of the text: the same question asked twice
gets the same answer, and asking a model again costs a round trip plus the
~900 tokens of classifier prompt that go with every call. In operations the
same question is asked constantly — the transcripts behind this change show
one question asked three times in a few minutes, each one paying full price.

The cache has two tiers. The local one is a bounded map on this replica and
answers in nanoseconds. The shared one is the store every replica reads, so a
question the classifier answered on one pod is not asked again on the next —
which, behind a load balancer, is where the second ask usually lands. A local
miss consults the shared store and keeps what it finds; a fresh verdict goes
to both. The shared tier is optional and best-effort: a store that is slow
or down turns into a miss, never a failure, because a cache that can take the
classifier down with it is worse than no cache.

Only a verdict the classifier actually produced is stored. A failure is never
cached: the fallback to deterministic rules is a degraded state, and freezing
it would turn a blip into an outage that outlives it.

The key includes the organization. Verdicts do not currently vary by tenant,
but the cache would be the wrong place to discover that they had started to.
*/
type verdictCache struct {
	entries *lru.Cache[string, ClassifierResult]
	shared  repositories.ScopeVerdictCacheRepository
	ttl     time.Duration
	logger  *zap.Logger
}

func newVerdictCache(
	shared repositories.ScopeVerdictCacheRepository,
	ttl time.Duration,
	logger *zap.Logger,
) *verdictCache {
	if logger == nil {
		logger = zap.NewNop()
	}
	cache := &verdictCache{shared: shared, ttl: ttl, logger: logger}
	entries, err := lru.New[string, ClassifierResult](verdictCacheSize)
	if err != nil {
		// Only a non-positive size can fail, and the size is a constant above.
		return cache
	}
	cache.entries = entries

	return cache
}

func (c *verdictCache) get(
	ctx context.Context,
	orgID pulid.ID,
	conversation, input string,
) (ClassifierResult, bool) {
	if c == nil {
		return ClassifierResult{}, false
	}
	key := verdictKey(orgID, conversation, input)

	if c.entries != nil {
		if result, ok := c.entries.Get(key); ok {
			return result, true
		}
	}

	if c.shared == nil {
		return ClassifierResult{}, false
	}

	bounded, cancel := context.WithTimeout(ctx, sharedTimeout)
	defer cancel()
	cached, err := c.shared.Get(bounded, key)
	if err != nil {
		c.logger.Debug("shared verdict lookup failed; classifying", zap.Error(err))

		return ClassifierResult{}, false
	}
	if cached == nil {
		return ClassifierResult{}, false
	}

	result := ClassifierResult{Category: cached.Category, Reasoning: cached.Reasoning}
	if c.entries != nil {
		c.entries.Add(key, result)
	}

	return result, true
}

func (c *verdictCache) put(
	ctx context.Context,
	orgID pulid.ID,
	conversation, input string,
	result ClassifierResult,
) {
	if c == nil {
		return
	}
	key := verdictKey(orgID, conversation, input)

	if c.entries != nil {
		c.entries.Add(key, result)
	}
	if c.shared == nil {
		return
	}

	// The write rides a context the request cannot cancel: the verdict is
	// already in hand and the person already answered, and a write abandoned
	// because the reader went away is a classification the next replica
	// pays for again.
	bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), sharedTimeout)
	defer cancel()
	err := c.shared.Set(bounded, key, &repositories.CachedScopeVerdict{
		Category:  result.Category,
		Reasoning: result.Reasoning,
	}, c.ttl)
	if err != nil {
		c.logger.Debug("shared verdict write failed; kept locally", zap.Error(err))
	}
}

// verdictKey hashes the message rather than keying on it directly, so a bounded
// cache cannot be filled with the text of what people asked. Case and
// surrounding whitespace are normalised because they change nothing about what
// a request is asking for.
func verdictKey(orgID pulid.ID, conversation, input string) string {
	// The conversation is hashed with the message rather than beside it: what
	// was classified is the pair, and a key that ignored half of it would hand
	// a follow-up the verdict from a different thread.
	normalized := strings.ToLower(strings.TrimSpace(conversation)) +
		"\x00" + strings.ToLower(strings.TrimSpace(input))
	sum := sha256.Sum256([]byte(normalized))

	return orgID.String() + ":" + hex.EncodeToString(sum[:])
}
