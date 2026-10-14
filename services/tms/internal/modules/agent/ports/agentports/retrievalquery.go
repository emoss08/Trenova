package agentports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
)

const MaxQueryTextRunes = 2000

const DefaultCatalogSimilarityFloor = 0.5

var ErrQueryTextRequired = errors.New("a query vector needs text to embed")

var ErrQueryTenantRequired = errors.New("a query vector needs an organization and business unit")

var ErrCatalogRequestInvalid = errors.New("a catalog similarity request needs a corpus and items")

var ErrCatalogQueryNotUsable = errors.New(
	"a catalog similarity request needs a usable query vector",
)

var ErrCatalogItemKeyTooLong = errors.New("a catalog item key is longer than the table allows")

var ErrCatalogItemKeyRepeated = errors.New("a catalog item key appears twice in one corpus")

type QueryVectorRequest struct {
	TenantInfo pagination.TenantInfo
	Text       string
	// Focus is the turn's own message, which Text may follow with the one
	// before it. It is never embedded; it is read for the turn's words.
	Focus       string
	Attribution aiusageports.AIUsageAttribution
}

type QueryVector struct {
	Available  bool
	Reason     airetrieval.UnavailableReason
	Vector     []float32
	ModelKey   string
	Dimensions int
}

func (q QueryVector) Usable() bool {
	return q.Available && len(q.Vector) > 0 && len(q.Vector) == q.Dimensions && q.ModelKey != ""
}

func UnavailableQueryVector(reason airetrieval.UnavailableReason) QueryVector {
	return QueryVector{Reason: reason}
}

type QueryVectorizer interface {
	Vectorize(ctx context.Context, req *QueryVectorRequest) (QueryVector, error)
	Availability(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (airetrieval.Availability, error)
}

func TurnQueryText(input string, history []conversation.Message) string {
	for idx := len(history) - 1; idx >= 0; idx-- {
		message := history[idx]
		if message.Role != conversation.RoleUser || message.Delegated() {
			continue
		}

		return input + "\n" + message.Content
	}

	return input
}

func TurnQueryRequest(req *RunRequest) QueryVectorRequest {
	if req == nil || req.Actor == nil {
		return QueryVectorRequest{}
	}

	attribution := aiusageports.AIUsageAttribution{
		UserID:   req.Actor.UserID,
		ThreadID: req.ThreadID,
		RunID:    req.RunID,
		Purpose:  req.AttributedPurpose(),
	}
	if req.Definition != nil {
		attribution.AgentDefinitionID = req.Definition.ID
	}

	return QueryVectorRequest{
		TenantInfo:  req.Actor.TenantInfo(),
		Text:        TurnQueryText(req.Input, req.History),
		Focus:       req.Input,
		Attribution: attribution,
	}
}

type EmbeddingCatalogItem struct {
	Key         string
	ContentHash string
	Text        string
}

func NewEmbeddingCatalogItem(key, text string) EmbeddingCatalogItem {
	return EmbeddingCatalogItem{Key: key, ContentHash: hashutils.SHA256Hex(text), Text: text}
}

type CatalogSimilarityRequest struct {
	TenantInfo pagination.TenantInfo
	Corpus     airetrieval.CatalogCorpus
	Items      []EmbeddingCatalogItem
	Query      QueryVector
}

type CatalogSimilarities struct {
	Available bool
	Reason    airetrieval.UnavailableReason
	ByKey     map[string]float64
}

type CatalogVectorIndex interface {
	Similarities(ctx context.Context, req *CatalogSimilarityRequest) (CatalogSimilarities, error)
}
