package pgvector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/uptrace/bun"
)

type VectorState string

const (
	VectorReady            = VectorState("Ready")
	VectorExtensionMissing = VectorState("ExtensionMissing")
	VectorExtensionTooOld  = VectorState("ExtensionTooOld")
	VectorSchemaMissing    = VectorState("SchemaMissing")
)

const (
	VectorExtensionName       = "vector"
	VectorMinExtensionVersion = "0.8"
	VectorSchemaTable         = "ai_embeddings"
)

const vectorProbeTemplate = `SELECT
	ext.extversion AS extension_version,
	COALESCE(
		string_to_array(regexp_replace(ext.extversion, '[^0-9.].*$', ''), '.')::int[] >= ARRAY[0, 8],
		false
	) AS version_supported,
	to_regclass(%[1]s) IS NOT NULL AS schema_present
FROM (SELECT 1) AS probe
LEFT JOIN pg_extension AS ext ON ext.extname = %[2]s`

var (
	vectorProbeQuery    = fmt.Sprintf(vectorProbeTemplate, "?", "?")
	vectorProbeQueryPgx = fmt.Sprintf(vectorProbeTemplate, "$1", "$2")
)

type RowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type VectorSupport struct {
	State            VectorState
	ExtensionVersion string
}

func (v VectorSupport) Ready() bool { return v.State == VectorReady }

func (v VectorSupport) ExtensionInstalled() bool {
	return v.ExtensionVersion != ""
}

type vectorProbeRow struct {
	ExtensionVersion *string `bun:"extension_version"`
	VersionSupported bool    `bun:"version_supported"`
	SchemaPresent    bool    `bun:"schema_present"`
}

func ProbeVector(ctx context.Context, db bun.IDB) (VectorSupport, error) {
	var row vectorProbeRow
	if err := db.NewRaw(vectorProbeQuery, VectorSchemaTable, VectorExtensionName).
		Scan(ctx, &row); err != nil {
		return VectorSupport{}, fmt.Errorf("probe pgvector: %w", err)
	}

	return classifyVector(row), nil
}

func ProbeVectorPgx(ctx context.Context, q RowQuerier) (VectorSupport, error) {
	var row vectorProbeRow
	if err := q.QueryRow(ctx, vectorProbeQueryPgx, VectorSchemaTable, VectorExtensionName).
		Scan(&row.ExtensionVersion, &row.VersionSupported, &row.SchemaPresent); err != nil {
		return VectorSupport{}, fmt.Errorf("probe pgvector: %w", err)
	}

	return classifyVector(row), nil
}

func classifyVector(row vectorProbeRow) VectorSupport {
	if row.ExtensionVersion == nil || *row.ExtensionVersion == "" {
		return VectorSupport{State: VectorExtensionMissing}
	}

	support := VectorSupport{ExtensionVersion: *row.ExtensionVersion}
	switch {
	case !row.VersionSupported:
		support.State = VectorExtensionTooOld
	case !row.SchemaPresent:
		support.State = VectorSchemaMissing
	default:
		support.State = VectorReady
	}

	return support
}
