package retrievalservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	skipInactiveMemory     = "The memory is retired, a suggestion or past its expiry."
	skipUnsearchableDoc    = "The document is not the current version, or it was rejected."
	skipSensitiveDocument  = "The document belongs to a record whose text is not sent for embedding."
	skipUnreadDocument     = "No text has been read from the document yet."
	skipEmptySource        = "The source has no text to index."
	failureEmbeddingPrefix = "Embedding failed: "
	failureWritePrefix     = "Storing the embeddings failed: "
)

type sourceOutcome int

const (
	outcomeIndexed sourceOutcome = iota
	outcomeSkipped
	outcomeFailed
	outcomeMissing
)

type sourceWork struct {
	entry   *airetrieval.IndexEntry
	chunks  []Chunk
	pending []int
	vectors map[int][]float32
	outcome sourceOutcome
	message string
	retryAt int64
}

func (w *sourceWork) ref() repositories.AIRetrievalSourceRef {
	return repositories.AIRetrievalSourceRef{
		TenantInfo: w.entry.Key().TenantInfo(),
		SourceType: w.entry.SourceType,
		SourceID:   w.entry.SourceID,
	}
}

func (w *sourceWork) skip(message string) {
	w.outcome = outcomeSkipped
	w.message = message
	w.chunks = nil
}

func (w *sourceWork) fail(message string, retryAt int64) {
	w.outcome = outcomeFailed
	w.message = stringutils.TruncateRunes(message, maxOutcomeMessage)
	w.retryAt = retryAt
}

type indexBatch struct {
	service    *Service
	tenant     pagination.TenantInfo
	modelKey   string
	dimensions int
	now        int64
	works      []*sourceWork
	cost       decimal.Decimal
	embedded   int
	superseded int
}

func (b *indexBatch) run(ctx context.Context, entries []*airetrieval.IndexEntry) error {
	b.cost = decimal.Zero
	b.works = make([]*sourceWork, 0, len(entries))
	for _, entry := range entries {
		b.works = append(b.works, &sourceWork{entry: entry})
	}

	if err := b.read(ctx); err != nil {
		return err
	}
	if err := b.diff(ctx); err != nil {
		return err
	}
	b.embed(ctx)
	if err := b.write(ctx); err != nil {
		return err
	}

	return b.mark(ctx)
}

func (b *indexBatch) byType(sourceType airetrieval.SourceType) ([]*sourceWork, []pulid.ID) {
	works := make([]*sourceWork, 0, len(b.works))
	ids := make([]pulid.ID, 0, len(b.works))
	for _, work := range b.works {
		if work.entry.SourceType == sourceType {
			works = append(works, work)
			ids = append(ids, work.entry.SourceID)
		}
	}

	return works, ids
}

func (b *indexBatch) read(ctx context.Context) error {
	if err := b.readMemories(ctx); err != nil {
		return err
	}
	if err := b.readDocuments(ctx); err != nil {
		return err
	}

	return b.readMessages(ctx)
}

func (b *indexBatch) readMemories(ctx context.Context) error {
	works, ids := b.byType(airetrieval.SourceTypeMemory)
	if len(works) == 0 {
		return nil
	}

	memories, err := b.service.sources.GetMemories(ctx, repositories.RetrievalSourcesRequest{
		TenantInfo: b.tenant,
		IDs:        ids,
	})
	if err != nil {
		return fmt.Errorf("read memories: %w", err)
	}

	found := make(map[pulid.ID]*agent.Memory, len(memories))
	for _, memory := range memories {
		found[memory.ID] = memory
	}
	for _, work := range works {
		memory, ok := found[work.entry.SourceID]
		switch {
		case !ok:
			work.outcome = outcomeMissing
		case !memory.Active(b.now):
			work.skip(skipInactiveMemory)
		default:
			work.chunks = ChunkMemory(memory)
		}
	}

	return nil
}

func (b *indexBatch) readDocuments(ctx context.Context) error {
	works, ids := b.byType(airetrieval.SourceTypeDocument)
	if len(works) == 0 {
		return nil
	}

	sources, err := b.service.sources.GetDocuments(ctx, repositories.RetrievalDocumentsRequest{
		TenantInfo:   b.tenant,
		IDs:          ids,
		IncludePages: true,
	})
	if err != nil {
		return fmt.Errorf("read documents: %w", err)
	}

	found := make(map[pulid.ID]*repositories.RetrievalDocumentSource, len(sources))
	for _, source := range sources {
		found[source.Document.ID] = source
	}
	for _, work := range works {
		source, ok := found[work.entry.SourceID]
		switch {
		case !ok:
			work.outcome = outcomeMissing
		case !source.Document.Searchable():
			work.skip(skipUnsearchableDoc)
		case !DocumentTextEmbeddable(b.service.registry, source.Document.OwnerResource()):
			work.skip(skipSensitiveDocument)
		default:
			work.chunks = ChunkDocument(source)
			if len(work.chunks) == 0 {
				work.skip(skipUnreadDocument)
			}
		}
	}

	return nil
}

func (b *indexBatch) readMessages(ctx context.Context) error {
	works, ids := b.byType(airetrieval.SourceTypeInboundMessage)
	if len(works) == 0 {
		return nil
	}

	messages, err := b.service.sources.GetInboundMessages(
		ctx,
		repositories.RetrievalSourcesRequest{TenantInfo: b.tenant, IDs: ids},
	)
	if err != nil {
		return fmt.Errorf("read inbound messages: %w", err)
	}

	found := make(map[pulid.ID]*inboundmessage.InboundMessage, len(messages))
	for _, message := range messages {
		found[message.ID] = message
	}
	for _, work := range works {
		message, ok := found[work.entry.SourceID]
		if !ok {
			work.outcome = outcomeMissing
			continue
		}
		work.chunks = ChunkEmail(message)
		if len(work.chunks) == 0 {
			work.skip(skipEmptySource)
		}
	}

	return nil
}

func DocumentTextEmbeddable(registry *permission.Registry, owner permission.Resource) bool {
	if registry == nil || owner == "" {
		return false
	}

	for _, resource := range []permission.Resource{permission.ResourceDocument, owner} {
		definition, ok := registry.Get(resource.String())
		if !ok {
			return false
		}
		if !permission.SensitivityInternal.CanAccess(definition.DefaultSensitivity) {
			return false
		}
	}

	return true
}

func (b *indexBatch) diff(ctx context.Context) error {
	for _, work := range b.works {
		if work.outcome != outcomeIndexed || len(work.chunks) == 0 {
			continue
		}

		stored, err := b.service.repo.ListChunkHashes(ctx, repositories.ListEmbeddingChunkHashesRequest{
			Source:   work.ref(),
			ModelKey: b.modelKey,
		})
		if err != nil {
			return fmt.Errorf("list stored chunk hashes: %w", err)
		}

		work.pending = ChangedChunks(work.chunks, stored)
	}

	return nil
}

func ChangedChunks(chunks []Chunk, stored []repositories.EmbeddingChunkHash) []int {
	hashes := make(map[int]string, len(stored))
	for _, chunk := range stored {
		hashes[chunk.ChunkIndex] = chunk.ContentHash
	}

	changed := make([]int, 0, len(chunks))
	for idx, chunk := range chunks {
		if hash, ok := hashes[chunk.Index]; !ok || hash != chunk.Hash {
			changed = append(changed, idx)
		}
	}

	return changed
}

func (b *indexBatch) embed(ctx context.Context) {
	inputs := make([]string, 0, len(b.works))
	owners := make([]*sourceWork, 0, len(b.works))
	positions := make([]int, 0, len(b.works))
	for _, work := range b.works {
		if work.outcome != outcomeIndexed {
			continue
		}
		for _, idx := range work.pending {
			inputs = append(inputs, work.chunks[idx].Text)
			owners = append(owners, work)
			positions = append(positions, idx)
		}
	}
	if len(inputs) == 0 {
		return
	}

	result, err := b.service.embeddings.Embed(ctx, serviceports.EmbedRequest{
		TenantInfo: b.tenant,
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     inputs,
		ModelKey:   b.modelKey,
		Surface:    aiusage.SurfaceIndexing,
	})
	if result.CostUSD != nil {
		b.cost = b.cost.Add(*result.CostUSD)
	}
	if err == nil && len(result.Vectors) != len(inputs) {
		err = fmt.Errorf("%w: %d vectors for %d inputs",
			serviceports.ErrEmbeddingResponseInvalid, len(result.Vectors), len(inputs))
	}
	if err != nil {
		b.failEmbedding(err, owners)
		return
	}

	b.embedded += len(inputs)
	for idx, work := range owners {
		if work.vectors == nil {
			work.vectors = make(map[int][]float32, len(work.pending))
		}
		work.vectors[positions[idx]] = result.Vectors[idx]
	}
}

func (b *indexBatch) failEmbedding(err error, owners []*sourceWork) {
	b.service.l.Warn("embedding a retrieval batch failed",
		zap.String("organizationId", b.tenant.OrgID.String()),
		zap.String("modelKey", b.modelKey),
		zap.Error(err),
	)

	permanent := errortypes.IsBusinessError(err) && !errors.Is(err, serviceports.ErrNoProviderConfigured)
	for _, work := range owners {
		if work.outcome == outcomeFailed {
			continue
		}
		retryAt := RetryAt(b.now, work.entry.Attempts)
		if permanent {
			retryAt = 0
		}
		work.fail(failureEmbeddingPrefix+err.Error(), retryAt)
	}
}

func (b *indexBatch) write(ctx context.Context) error {
	for _, work := range b.works {
		switch work.outcome {
		case outcomeMissing:
			if _, err := b.service.repo.DeleteSource(ctx, work.ref()); err != nil {
				return fmt.Errorf("drop a source that no longer exists: %w", err)
			}
		case outcomeSkipped:
			if err := b.replace(ctx, work, nil); err != nil {
				return err
			}
		case outcomeIndexed:
			chunks := make([]repositories.EmbeddingChunk, 0, len(work.chunks))
			for idx, chunk := range work.chunks {
				chunks = append(chunks, repositories.EmbeddingChunk{
					ChunkIndex:  chunk.Index,
					ContentHash: chunk.Hash,
					Vector:      work.vectors[idx],
				})
			}
			if err := b.replace(ctx, work, chunks); err != nil {
				return err
			}
		case outcomeFailed:
		}
	}

	return nil
}

func (b *indexBatch) replace(
	ctx context.Context,
	work *sourceWork,
	chunks []repositories.EmbeddingChunk,
) error {
	_, err := b.service.repo.ReplaceChunks(ctx, repositories.ReplaceEmbeddingChunksRequest{
		Source:     work.ref(),
		ModelKey:   b.modelKey,
		Dimensions: b.dimensions,
		Chunks:     chunks,
	})
	switch {
	case err == nil:
		return nil
	case errors.Is(err, airetrieval.ErrVectorUnavailable):
		return fmt.Errorf("store embeddings: %w", err)
	default:
		work.fail(failureWritePrefix+err.Error(), RetryAt(b.now, work.entry.Attempts))
		return nil
	}
}

func (b *indexBatch) outcomes(outcome sourceOutcome) []repositories.IndexEntryOutcome {
	outcomes := make([]repositories.IndexEntryOutcome, 0, len(b.works))
	for _, work := range b.works {
		if work.outcome != outcome {
			continue
		}
		outcomes = append(outcomes, repositories.IndexEntryOutcome{
			Key:        work.entry.Key(),
			Generation: work.entry.Generation,
			ChunkCount: len(work.chunks),
			Error:      work.message,
			RetryAt:    work.retryAt,
		})
	}

	return outcomes
}

func (b *indexBatch) mark(ctx context.Context) error {
	marks := []struct {
		outcome sourceOutcome
		apply   func(context.Context, repositories.MarkIndexEntriesRequest) (repositories.MarkIndexEntriesResult, error)
	}{
		{outcome: outcomeIndexed, apply: b.service.repo.MarkIndexed},
		{outcome: outcomeSkipped, apply: b.service.repo.MarkSkipped},
		{outcome: outcomeFailed, apply: b.service.repo.MarkFailed},
	}

	for _, mark := range marks {
		outcomes := b.outcomes(mark.outcome)
		if len(outcomes) == 0 {
			continue
		}
		applied, err := mark.apply(ctx, repositories.MarkIndexEntriesRequest{
			Outcomes: outcomes,
			Now:      b.service.now(),
		})
		if err != nil {
			return fmt.Errorf("record index outcomes: %w", err)
		}
		b.superseded += applied.Superseded
	}

	return nil
}

func (b *indexBatch) fill(result *serviceports.RetrievalIndexBatchResult) {
	for _, work := range b.works {
		switch work.outcome {
		case outcomeIndexed:
			result.Indexed++
		case outcomeSkipped, outcomeMissing:
			result.Skipped++
		case outcomeFailed:
			result.Failed++
		}
	}
	result.ChunksEmbedded = b.embedded
	result.CostUSD = b.cost
	result.Superseded = b.superseded
}
