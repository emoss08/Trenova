package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
)

type GlobalSearchRequest struct {
	Query       string
	TenantInfo  pagination.TenantInfo
	Principal   PrincipalInfo
	Limit       int
	EntityTypes []search.EntityType
}

type PrincipalInfo struct {
	Type     PrincipalType
	ID       pulid.ID
	UserID   pulid.ID
	APIKeyID pulid.ID
}

type GlobalSearchResult struct {
	Query  string               `json:"query"`
	Groups []*GlobalSearchGroup `json:"groups"`
}

type GlobalSearchGroup struct {
	EntityType search.EntityType  `json:"entityType"`
	Label      string             `json:"label"`
	Hits       []*GlobalSearchHit `json:"hits"`
}

type GlobalSearchHit struct {
	ID         string            `json:"id"`
	EntityType search.EntityType `json:"entityType"`
	Title      string            `json:"title"`
	Subtitle   string            `json:"subtitle,omitempty"`
	Href       string            `json:"href"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type GlobalSearchService interface {
	Search(ctx context.Context, req *GlobalSearchRequest) (*GlobalSearchResult, error)
}
