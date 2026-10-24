package airetrieval

import (
	"context"

	"github.com/emoss08/keel"
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
	keel.Table    `keel:"ai_embeddings"`

	OrganizationID pulid.ID            `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	SourceType     SourceType          `json:"sourceType"     bun:"source_type,pk,type:VARCHAR(30),notnull" keel:"source_type,pk"`
	SourceID       pulid.ID            `json:"sourceId"       bun:"source_id,pk,type:VARCHAR(100),notnull" keel:"source_id,pk"`
	ChunkIndex     int                 `json:"chunkIndex"     bun:"chunk_index,pk,type:INTEGER,notnull" keel:"chunk_index,pk"`
	ModelKey       string              `json:"modelKey"       bun:"model_key,pk,type:VARCHAR(300),notnull" keel:"model_key,pk"`
	Dimensions     int                 `json:"dimensions"     bun:"dimensions,type:INTEGER,notnull" keel:"dimensions"`
	ContentHash    string              `json:"contentHash"    bun:"content_hash,type:VARCHAR(64),notnull" keel:"content_hash"`
	Embedding      pgvector.HalfVector `json:"-"              bun:"embedding,type:halfvec,notnull" keel:"embedding"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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
	keel.Table    `keel:"ai_catalog_embeddings"`

	ModelKey    string              `json:"modelKey"    bun:"model_key,pk,type:VARCHAR(300),notnull" keel:"model_key,pk"`
	Corpus      CatalogCorpus       `json:"corpus"      bun:"corpus,pk,type:VARCHAR(30),notnull" keel:"corpus,pk"`
	ItemKey     string              `json:"itemKey"     bun:"item_key,pk,type:VARCHAR(200),notnull" keel:"item_key,pk"`
	ContentHash string              `json:"contentHash" bun:"content_hash,pk,type:VARCHAR(64),notnull" keel:"content_hash,pk"`
	Dimensions  int                 `json:"dimensions"  bun:"dimensions,type:INTEGER,notnull" keel:"dimensions"`
	Embedding   pgvector.HalfVector `json:"-"           bun:"embedding,type:halfvec,notnull" keel:"embedding"`
	CreatedAt   int64               `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
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
