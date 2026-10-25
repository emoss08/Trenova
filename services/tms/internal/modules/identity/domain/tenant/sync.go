package tenant

import "github.com/emoss08/trenova/shared/pulid"

type SyncBusinessUnit struct {
	ID        pulid.ID          `json:"id"`
	Name      string            `json:"name"`
	Code      string            `json:"code"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt int64             `json:"createdAt"`
	UpdatedAt int64             `json:"updatedAt"`
}

type SyncOrganization struct {
	ID             pulid.ID          `json:"id"`
	BusinessUnitID pulid.ID          `json:"businessUnitId"`
	Name           string            `json:"name"`
	LoginSlug      string            `json:"loginSlug,omitempty"`
	ScacCode       string            `json:"scacCode"`
	DOTNumber      string            `json:"dotNumber"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      int64             `json:"createdAt"`
	UpdatedAt      int64             `json:"updatedAt"`
}
