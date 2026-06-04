package graphql

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
)

//go:embed persisted-documents.json
var persistedDocumentsFS embed.FS

const persistedDocumentsPath = "persisted-documents.json"

type PersistedOperationManifest struct {
	queries map[string]string
}

type persistedGraphQLRequest struct {
	Extensions    persistedGraphQLRequestExtensions `json:"extensions,omitempty"`
	OperationName string                            `json:"operationName,omitempty"`
	Query         string                            `json:"query,omitempty"`
	Variables     any                               `json:"variables,omitempty"`
}

type persistedGraphQLRequestExtensions struct {
	PersistedQuery persistedGraphQLPersistedQuery `json:"persistedQuery,omitempty"`
}

type persistedGraphQLPersistedQuery struct {
	Hash       string `json:"hash,omitempty"`
	SHA256Hash string `json:"sha256Hash,omitempty"`
	Version    int    `json:"version,omitempty"`
}

func NewPersistedOperationManifest() (*PersistedOperationManifest, error) {
	data, err := persistedDocumentsFS.ReadFile(persistedDocumentsPath)
	if err != nil {
		return nil, fmt.Errorf("reading GraphQL persisted operation manifest: %w", err)
	}

	manifest, err := LoadPersistedOperationManifest(data)
	if err != nil {
		return nil, fmt.Errorf("loading GraphQL persisted operation manifest: %w", err)
	}

	return manifest, nil
}

func LoadPersistedOperationManifest(data []byte) (*PersistedOperationManifest, error) {
	queries := map[string]string{}
	if err := sonic.Unmarshal(data, &queries); err != nil {
		return nil, fmt.Errorf("parsing persisted operations: %w", err)
	}

	for hash, query := range queries {
		if strings.TrimSpace(hash) == "" {
			return nil, errors.New("persisted operation hash cannot be empty")
		}
		if strings.TrimSpace(query) == "" {
			return nil, fmt.Errorf("persisted operation %q has an empty query", hash)
		}
	}

	return &PersistedOperationManifest{
		queries: queries,
	}, nil
}

func (m *PersistedOperationManifest) Query(hash string) (string, bool) {
	if m == nil {
		return "", false
	}

	query, ok := m.queries[normalizePersistedHash(hash)]
	return query, ok
}

func (m *PersistedOperationManifest) KnownHashes() []string {
	if m == nil {
		return []string{}
	}

	hashes := make([]string, 0, len(m.queries))
	for hash := range m.queries {
		hashes = append(hashes, hash)
	}
	return hashes
}

func rewritePersistedOperationRequest(
	req *http.Request,
	manifest *PersistedOperationManifest,
	enforceSafelist bool,
) error {
	if req.Body == nil {
		if enforceSafelist {
			return persistedOperationError("query", "GraphQL persisted operation request body is required")
		}
		return nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("reading GraphQL request body: %w", err)
	}
	if len(body) == 0 {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = 0
		if enforceSafelist {
			return persistedOperationError("query", "GraphQL persisted operation request body is required")
		}
		return nil
	}

	var gqlReq persistedGraphQLRequest
	if err = sonic.Unmarshal(body, &gqlReq); err != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		return persistedOperationError("query", "GraphQL request body must be valid JSON")
	}

	hash := gqlReq.persistedHash()
	if hash == "" {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		if enforceSafelist {
			return persistedOperationError("extensions.persistedQuery.sha256Hash", "GraphQL persisted operation hash is required")
		}
		return nil
	}

	query, ok := manifest.Query(hash)
	if !ok {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		return persistedOperationError("extensions.persistedQuery.sha256Hash", "GraphQL persisted operation is not safelisted")
	}

	gqlReq.Query = query
	rewrittenBody, err := sonic.Marshal(gqlReq)
	if err != nil {
		return fmt.Errorf("rewriting GraphQL persisted operation request: %w", err)
	}

	req.Body = io.NopCloser(bytes.NewReader(rewrittenBody))
	req.ContentLength = int64(len(rewrittenBody))
	return nil
}

func enforcePersistedOperations(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.App.IsProduction() || cfg.App.IsStaging()
}

func (r *persistedGraphQLRequest) persistedHash() string {
	if r.Extensions.PersistedQuery.SHA256Hash != "" {
		return r.Extensions.PersistedQuery.SHA256Hash
	}
	return r.Extensions.PersistedQuery.Hash
}

func normalizePersistedHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) == 64 && !strings.Contains(hash, ":") {
		return "sha256:" + hash
	}
	return hash
}

func persistedOperationError(field string, message string) error {
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}
