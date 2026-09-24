package airetrieval

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/pgvector/pgvector-go"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Embedding)(nil)
	_ bun.BeforeAppendModelHook = (*CatalogEmbedding)(nil)
)

type Embedding struct {
	bun.BaseModel `bun:"table:ai_embeddings,alias:aiemb" json:"-"`

	OrganizationID pulid.ID            `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	SourceType     SourceType          `json:"sourceType"     bun:"source_type,pk,type:VARCHAR(30),notnull"`
	SourceID       pulid.ID            `json:"sourceId"       bun:"source_id,pk,type:VARCHAR(100),notnull"`
	ChunkIndex     int                 `json:"chunkIndex"     bun:"chunk_index,pk,type:INTEGER,notnull"`
	ModelKey       string              `json:"modelKey"       bun:"model_key,pk,type:VARCHAR(300),notnull"`
	Dimensions     int                 `json:"dimensions"     bun:"dimensions,type:INTEGER,notnull"`
	ContentHash    string              `json:"contentHash"    bun:"content_hash,type:VARCHAR(64),notnull"`
	Embedding      pgvector.HalfVector `json:"-"              bun:"embedding,type:halfvec,notnull"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (e *Embedding) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *Embedding) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *Embedding) GetTableName() string { return "ai_embeddings" }

func (e *Embedding) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.CreatedAt == 0 {
			e.CreatedAt = now
		}
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}

type CatalogEmbedding struct {
	bun.BaseModel `bun:"table:ai_catalog_embeddings,alias:aicat" json:"-"`

	ModelKey    string              `json:"modelKey"    bun:"model_key,pk,type:VARCHAR(300),notnull"`
	Corpus      CatalogCorpus       `json:"corpus"      bun:"corpus,pk,type:VARCHAR(30),notnull"`
	ItemKey     string              `json:"itemKey"     bun:"item_key,pk,type:VARCHAR(200),notnull"`
	ContentHash string              `json:"contentHash" bun:"content_hash,pk,type:VARCHAR(64),notnull"`
	Dimensions  int                 `json:"dimensions"  bun:"dimensions,type:INTEGER,notnull"`
	Embedding   pgvector.HalfVector `json:"-"           bun:"embedding,type:halfvec,notnull"`
	CreatedAt   int64               `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (c *CatalogEmbedding) GetTableName() string { return "ai_catalog_embeddings" }

func (c *CatalogEmbedding) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok && c.CreatedAt == 0 {
		c.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (c *CatalogEmbedding) Vector() []float32 { return c.Embedding.Slice() }

type CatalogItemRef struct {
	ItemKey     string `json:"itemKey"     bun:"item_key"`
	ContentHash string `json:"contentHash" bun:"content_hash"`
}
