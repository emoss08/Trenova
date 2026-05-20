package tenant

import "github.com/emoss08/trenova/shared/pulid"

type SyncBusinessUnit struct {
	ID        pulid.ID          `json:"id"        bun:"id"`
	Name      string            `json:"name"      bun:"name"`
	Code      string            `json:"code"      bun:"code"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt int64             `json:"createdAt" bun:"created_at"`
	UpdatedAt int64             `json:"updatedAt" bun:"updated_at"`
}

type SyncOrganization struct {
	ID             pulid.ID          `json:"id"                     bun:"id"`
	BusinessUnitID pulid.ID          `json:"businessUnitId"         bun:"business_unit_id"`
	Name           string            `json:"name"                   bun:"name"`
	LoginSlug      string            `json:"loginSlug,omitempty"    bun:"login_slug"`
	ScacCode       string            `json:"scacCode"               bun:"scac_code"`
	DOTNumber      string            `json:"dotNumber"              bun:"dot_number"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      int64             `json:"createdAt"              bun:"created_at"`
	UpdatedAt      int64             `json:"updatedAt"              bun:"updated_at"`
}
