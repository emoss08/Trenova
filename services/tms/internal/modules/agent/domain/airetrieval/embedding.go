package airetrieval

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/pgvector/pgvector-go"
)

type Embedding struct {
	keel.Table `keel:"ai_embeddings"`

	OrganizationID pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	SourceType     SourceType          `json:"sourceType" keel:"source_type,pk"`
	SourceID       pulid.ID            `json:"sourceId" keel:"source_id,pk"`
	ChunkIndex     int                 `json:"chunkIndex" keel:"chunk_index,pk"`
	ModelKey       string              `json:"modelKey" keel:"model_key,pk"`
	Dimensions     int                 `json:"dimensions" keel:"dimensions"`
	ContentHash    string              `json:"contentHash" keel:"content_hash"`
	Embedding      pgvector.HalfVector `json:"-" keel:"embedding"`
	CreatedAt      int64               `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (e *Embedding) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *Embedding) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *Embedding) GetTableName() string { return "ai_embeddings" }

type CatalogEmbedding struct {
	keel.Table `keel:"ai_catalog_embeddings"`

	ModelKey    string              `json:"modelKey" keel:"model_key,pk"`
	Corpus      CatalogCorpus       `json:"corpus" keel:"corpus,pk"`
	ItemKey     string              `json:"itemKey" keel:"item_key,pk"`
	ContentHash string              `json:"contentHash" keel:"content_hash,pk"`
	Dimensions  int                 `json:"dimensions" keel:"dimensions"`
	Embedding   pgvector.HalfVector `json:"-" keel:"embedding"`
	CreatedAt   int64               `json:"createdAt" keel:"created_at,autocreate"`
}

func (c *CatalogEmbedding) GetTableName() string { return "ai_catalog_embeddings" }

func (c *CatalogEmbedding) Vector() []float32 { return c.Embedding.Slice() }

type CatalogItemRef struct {
	ItemKey     string `json:"itemKey"`
	ContentHash string `json:"contentHash"`
}
