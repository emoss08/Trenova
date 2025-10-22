package meilisearchtype

import (
	"github.com/emoss08/trenova/pkg/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type EntityType string

const (
	EntityTypeShipment          = EntityType("shipment")
	EntityTypeInvoice           = EntityType("invoice")
	EntityTypeWorker            = EntityType("worker")
	EntityTypeCustomer          = EntityType("customer")
	EntityTypeCommodity         = EntityType("commodity")
	EntityTypeLocation          = EntityType("location")
	EntityTypeHazardousMaterial = EntityType("hazardous_material")
)

func (e EntityType) String() string {
	return string(e)
}

func (e EntityType) IsValid() bool {
	switch e {
	case EntityTypeShipment, EntityTypeInvoice, EntityTypeWorker,
		EntityTypeCustomer, EntityTypeCommodity, EntityTypeLocation,
		EntityTypeHazardousMaterial:
		return true
	default:
		return false
	}
}

type SearchDocument struct {
	ID             string         `json:"id"`
	EntityType     EntityType     `json:"entityType"`
	OrganizationID string         `json:"organizationId"`
	BusinessUnitID string         `json:"businessUnitId"`
	Title          string         `json:"title"`
	Subtitle       string         `json:"subtitle,omitempty"`
	Content        string         `json:"content"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      int64          `json:"createdAt"`
	UpdatedAt      int64          `json:"updatedAt"`
}

func (s *SearchDocument) Validate() error {
	return validation.ValidateStruct(
		s,
		validation.Field(
			&s.ID,
			validation.Required.Error("ID is required"),
		),
		validation.Field(
			&s.EntityType,
			validation.Required.Error("Entity type is required"),
		),
		validation.Field(
			&s.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&s.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&s.Title,
			validation.Required.Error("Title is required"),
		),
	)
}

func (s *SearchDocument) ToMap() map[string]any {
	return map[string]any{
		"id":             s.ID,
		"entityType":     s.EntityType,
		"organizationId": s.OrganizationID,
		"businessUnitId": s.BusinessUnitID,
		"title":          s.Title,
		"subtitle":       s.Subtitle,
		"content":        s.Content,
		"metadata":       s.Metadata,
		"createdAt":      s.CreatedAt,
		"updatedAt":      s.UpdatedAt,
	}
}

type SearchRequest struct {
	Query          string
	EntityTypes    []EntityType
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Limit          int
	Offset         int
	Filters        map[string]any
}

func (s *SearchRequest) Validate() error {
	return validation.ValidateStruct(
		s,
		validation.Field(&s.Query, validation.Required.Error("Query is required")),
		validation.Field(
			&s.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&s.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
	)
}

type SearchResponse struct {
	Hits             []SearchHit `json:"hits"`
	Total            int64       `json:"total"`
	Offset           int         `json:"offset"`
	Limit            int         `json:"limit"`
	ProcessingTimeMs int64       `json:"processingTimeMs"`
	Query            string      `json:"query"`
}

type SearchHit struct {
	ID                 string            `json:"id"`
	EntityType         EntityType        `json:"entityType"`
	Title              string            `json:"title"`
	Subtitle           string            `json:"subtitle,omitempty"`
	Metadata           map[string]any    `json:"metadata,omitempty"`
	Score              float64           `json:"score,omitempty"`
	HighlightedContent map[string]string `json:"highlightedContent,omitempty"`
}

type IndexConfig struct {
	Name                 string
	SearchableAttributes []string
	FilterableAttributes []string
	SortableAttributes   []string
	DisplayedAttributes  []string
	RankingRules         []string
	StopWords            []string
}

type BatchOperation struct {
	Action    BatchActionType
	Documents []SearchDocument
	IDs       []string
}

type BatchActionType string

const (
	BatchActionAdd    BatchActionType = "add"
	BatchActionUpdate BatchActionType = "update"
	BatchActionDelete BatchActionType = "delete"
)

type TaskInfo struct {
	TaskUID  int64  `json:"taskUid"`
	IndexUID string `json:"indexUid"`
	Status   string `json:"status"`
	Type     string `json:"type"`
}

type DeleteOperationRequest struct {
	EntityType EntityType
	OrgID      string
	BuID       string
	DocumentID string
}
