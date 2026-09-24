package dbdialect

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

type VectorState string

const (
	VectorReady            = VectorState("Ready")
	VectorExtensionMissing = VectorState("ExtensionMissing")
	VectorExtensionTooOld  = VectorState("ExtensionTooOld")
	VectorSchemaMissing    = VectorState("SchemaMissing")
	VectorUnsupported      = VectorState("Unsupported")
)

const (
	VectorExtensionName       = "vector"
	VectorMinExtensionVersion = "0.8"
	VectorSchemaTable         = "ai_embeddings"
)

const vectorProbeQuery = `SELECT
	ext.extversion AS extension_version,
	COALESCE(
		string_to_array(regexp_replace(ext.extversion, '[^0-9.].*$', ''), '.')::int[] >= ARRAY[0, 8],
		false
	) AS version_supported,
	to_regclass(?) IS NOT NULL AS schema_present
FROM (SELECT 1) AS probe
LEFT JOIN pg_extension AS ext ON ext.extname = ?`

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
	if FromBun(db).IsSQLite() {
		return VectorSupport{State: VectorUnsupported}, nil
	}

	var row vectorProbeRow
	if err := db.NewRaw(vectorProbeQuery, VectorSchemaTable, VectorExtensionName).
		Scan(ctx, &row); err != nil {
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

type Capabilities struct {
	kind   Kind
	vector VectorSupport
}

func NewCapabilities(kind Kind, vector VectorSupport) Capabilities {
	if kind.IsSQLite() {
		vector = VectorSupport{State: VectorUnsupported}
	}

	return Capabilities{kind: kind, vector: vector}
}

func ProbeCapabilities(ctx context.Context, db bun.IDB) (Capabilities, error) {
	vector, err := ProbeVector(ctx, db)
	if err != nil {
		return Capabilities{}, err
	}

	return NewCapabilities(FromBun(db), vector), nil
}

func (c Capabilities) Kind() Kind { return c.kind }

func (c Capabilities) Vector() VectorSupport { return c.vector }

func (c Capabilities) Supports(capability Capability) bool {
	if capability == CapVectorSearch {
		return c.kind.IsPostgres() && c.vector.Ready()
	}

	return c.kind.Supports(capability)
}
