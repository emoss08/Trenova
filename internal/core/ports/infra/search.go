package infra

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/types/pulid"
)

type SearchOptions struct {
	Query   string
	Limit   int
	Offset  int
	Types   []string // Filter by entity types
	OrgID   pulid.ID // Filter by organization
	BuID    pulid.ID // Filter by business unit
	SortBy  []string // Sorting options
	Filters []string // Additional filters
}

type SearchDocument struct {
	ID             string         `mapstructure:"id" json:"id"`
	Type           string         `mapstructure:"type" json:"type"`
	BusinessUnitID string         `mapstructure:"businessUnitId" json:"businessUnitId"`
	OrganizationID string         `mapstructure:"organizationId" json:"organizationId"`
	CreatedAt      int64          `mapstructure:"createdAt" json:"createdAt"`
	UpdatedAt      int64          `mapstructure:"updatedAt" json:"updatedAt"`
	Title          string         `mapstructure:"title" json:"title"`
	Description    string         `mapstructure:"description" json:"description"`
	SearchableText string         `mapstructure:"searchableText" json:"searchableText"`
	Metadata       map[string]any `mapstructure:"metadata" json:"metadata"`
}

type SearchTaskError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Link    string `json:"link"`
}

type SearchTask struct {
	Status     string          `json:"status"`
	TaskUID    int64           `json:"taskUid"`
	IndexUID   string          `json:"indexUid"`
	Type       string          `json:"type"`
	Error      SearchTaskError `json:"error"`
	Duration   string          `json:"duration,omitempty"`
	EnqueuedAt time.Time       `json:"enqueuedAt"`
	StartedAt  time.Time       `json:"startedAt,omitempty"`
	FinishedAt time.Time       `json:"finishedAt,omitempty"`
	Details    map[string]any  `json:"details,omitempty"`
	CanceledBy int64           `json:"canceledBy,omitempty"`
}

type SearchTaskInfo struct {
	UID        int64  `json:"uid"`
	IndexUID   string `json:"indexUid"`
	Type       string `json:"type"`
	EnqueuedAt string `json:"enqueuedAt"`
	TaskUID    int64  `json:"taskUid"`
	Status     string `json:"status"`
	Duration   string `json:"duration"`
}

type SearchClient interface {
	IndexDocuments(indexName string, docs []*SearchDocument) (*SearchTaskInfo, error)
	Search(ctx context.Context, opts *SearchOptions) ([]*SearchDocument, error)
	GetIndexName() string
	WaitForTask(taskUID int64, timeout time.Duration) (*SearchTask, error)
}

type SearchableEntity interface {
	GetID() string         // Returns unique identifier
	GetSearchType() string // Returns entity type (e.g., "shipment", "worker", etc.)
	ToDocument() SearchDocument
}
