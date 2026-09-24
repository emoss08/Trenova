package agentevalgate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/productguideservice"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/vectorutils"
)

const (
	EmbeddingFixturePath  = "evals/embeddings/nomic-embed-text.json"
	EmbeddingFixtureModel = "nomic-embed-text"
	OllamaURLEnv          = "TRENOVA_EVAL_OLLAMA_URL"
	DefaultOllamaURL      = "http://localhost:11434"
	RequireHybridEnv      = "TRENOVA_EVAL_REQUIRE_HYBRID"

	RecordEmbeddingsCommand = "cd services/tms && ollama pull nomic-embed-text && " +
		OllamaURLEnv + "=" + DefaultOllamaURL + " go test -tags nofitz -count=1 " +
		"-run 'TestRecordEmbeddingFixture' ./internal/core/services/agentevalgate/ -record && " +
		"go test -tags nofitz -count=1 -run 'AgainstFloors' " +
		"./internal/core/services/agentevalgate/ -update"

	recordBatchSize = 64
)

var ErrEmbeddingFixtureMissing = errors.New("the embedding fixture has not been recorded")

type EmbeddingFixture struct {
	Model      string            `json:"model"`
	Dimensions int               `json:"dimensions"`
	InputStyle string            `json:"inputStyle"`
	Documents  map[string][]byte `json:"documents"`
	Queries    map[string][]byte `json:"queries"`
}

type EmbeddingInputs struct {
	Documents []serviceports.EmbeddingCatalogItem
	Queries   []string
}

type Embed func(
	ctx context.Context,
	purpose serviceports.EmbeddingPurpose,
	inputs []string,
) ([][]float32, error)

func QueryHash(query string) string { return hashutils.SHA256Hex(query) }

func GuideItems() []serviceports.EmbeddingCatalogItem {
	return productguideservice.CatalogItems(productguide.Default)
}

func LoadEmbeddingFixture(path string) (*EmbeddingFixture, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrEmbeddingFixtureMissing, path)
	}
	if err != nil {
		return nil, err
	}

	var fixture EmbeddingFixture
	if err = sonic.Unmarshal(raw, &fixture); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if !airetrieval.IsAllowedDimension(fixture.Dimensions) {
		return nil, fmt.Errorf("%s records %d dimensions; re-record it with:\n  %s",
			path, fixture.Dimensions, RecordEmbeddingsCommand)
	}

	return &fixture, nil
}

func (f *EmbeddingFixture) Document(hash string) ([]float32, bool) {
	return f.vector(f.Documents, hash)
}

func (f *EmbeddingFixture) Query(query string) ([]float32, bool) {
	return f.vector(f.Queries, QueryHash(query))
}

func (f *EmbeddingFixture) vector(from map[string][]byte, hash string) ([]float32, bool) {
	packed, ok := from[hash]
	if !ok {
		return nil, false
	}

	vector, err := vectorutils.Unpack(packed)
	if err != nil || len(vector) != f.Dimensions {
		return nil, false
	}

	return vector, true
}

func (f *EmbeddingFixture) Stale(inputs EmbeddingInputs) error {
	var missing, unused []string

	wantDocuments := make(map[string]struct{}, len(inputs.Documents))
	for _, item := range inputs.Documents {
		wantDocuments[item.ContentHash] = struct{}{}
		if _, ok := f.Document(item.ContentHash); !ok {
			missing = append(missing, "document "+item.Key)
		}
	}
	wantQueries := make(map[string]struct{}, len(inputs.Queries))
	for _, query := range inputs.Queries {
		wantQueries[QueryHash(query)] = struct{}{}
		if _, ok := f.Query(query); !ok {
			missing = append(missing, fmt.Sprintf("query %q", query))
		}
	}
	for hash := range f.Documents {
		if _, ok := wantDocuments[hash]; !ok {
			unused = append(unused, "document "+hash)
		}
	}
	for hash := range f.Queries {
		if _, ok := wantQueries[hash]; !ok {
			unused = append(unused, "query "+hash)
		}
	}

	if len(missing) == 0 && len(unused) == 0 {
		return nil
	}
	slices.Sort(missing)
	slices.Sort(unused)

	var b strings.Builder
	fmt.Fprintf(&b, "%s is stale: %d entries have no vector and %d vectors are no longer used",
		EmbeddingFixturePath, len(missing), len(unused))
	for _, entry := range firstN(missing, 10) {
		fmt.Fprintf(&b, "\n  missing %s", entry)
	}
	for _, entry := range firstN(unused, 5) {
		fmt.Fprintf(&b, "\n  unused %s", entry)
	}
	fmt.Fprintf(&b, "\nRe-record it with:\n  %s", RecordEmbeddingsCommand)

	return errors.New(b.String())
}

func (f *EmbeddingFixture) Similarities(
	query string,
	items []serviceports.EmbeddingCatalogItem,
) (map[string]float64, error) {
	vector, ok := f.Query(query)
	if !ok {
		return nil, fmt.Errorf("no vector for query %q; re-record with:\n  %s",
			query, RecordEmbeddingsCommand)
	}

	similarity := make(map[string]float64, len(items))
	for _, item := range items {
		document, found := f.Document(item.ContentHash)
		if !found {
			return nil, fmt.Errorf("no vector for %s; re-record with:\n  %s",
				item.Key, RecordEmbeddingsCommand)
		}
		value, err := vectorutils.Cosine(vector, document)
		if err != nil {
			return nil, fmt.Errorf("compare %q with %s: %w", query, item.Key, err)
		}
		similarity[item.Key] = value
	}

	return similarity, nil
}

func RecordEmbeddingFixture(
	ctx context.Context,
	embed Embed,
	dimensions int,
	inputs EmbeddingInputs,
) (*EmbeddingFixture, error) {
	fixture := &EmbeddingFixture{
		Model:      EmbeddingFixtureModel,
		Dimensions: dimensions,
		InputStyle: string(aiprovider.EmbeddingInputStyleNomicPrefix),
		Documents:  make(map[string][]byte, len(inputs.Documents)),
		Queries:    make(map[string][]byte, len(inputs.Queries)),
	}

	documents := make([]string, 0, len(inputs.Documents))
	hashes := make([]string, 0, len(inputs.Documents))
	seen := make(map[string]struct{}, len(inputs.Documents)+len(inputs.Queries))
	for _, item := range inputs.Documents {
		if _, dup := seen[item.ContentHash]; dup {
			continue
		}
		seen[item.ContentHash] = struct{}{}
		documents = append(documents, item.Text)
		hashes = append(hashes, item.ContentHash)
	}
	if err := recordBatches(ctx, embed, serviceports.EmbeddingPurposeDocument, documents,
		hashes, dimensions, fixture.Documents); err != nil {
		return nil, err
	}

	queries := make([]string, 0, len(inputs.Queries))
	queryHashes := make([]string, 0, len(inputs.Queries))
	asked := make(map[string]struct{}, len(inputs.Queries))
	for _, query := range inputs.Queries {
		hash := QueryHash(query)
		if _, dup := asked[hash]; dup {
			continue
		}
		asked[hash] = struct{}{}
		queries = append(queries, query)
		queryHashes = append(queryHashes, hash)
	}
	if err := recordBatches(ctx, embed, serviceports.EmbeddingPurposeQuery, queries,
		queryHashes, dimensions, fixture.Queries); err != nil {
		return nil, err
	}

	return fixture, nil
}

func recordBatches(
	ctx context.Context,
	embed Embed,
	purpose serviceports.EmbeddingPurpose,
	texts []string,
	hashes []string,
	dimensions int,
	into map[string][]byte,
) error {
	for start := 0; start < len(texts); start += recordBatchSize {
		end := min(start+recordBatchSize, len(texts))
		vectors, err := embed(ctx, purpose, texts[start:end])
		if err != nil {
			return fmt.Errorf("embed %s inputs %d-%d: %w", purpose, start, end, err)
		}
		if len(vectors) != end-start {
			return fmt.Errorf("embed %s inputs %d-%d: got %d vectors",
				purpose, start, end, len(vectors))
		}
		for idx, vector := range vectors {
			if len(vector) != dimensions {
				return fmt.Errorf("embed %s input %d: got %d dimensions, want %d",
					purpose, start+idx, len(vector), dimensions)
			}
			into[hashes[start+idx]] = vectorutils.Pack(vector)
		}
	}

	return nil
}

func firstN(values []string, n int) []string {
	if len(values) <= n {
		return values
	}

	return values[:n]
}
