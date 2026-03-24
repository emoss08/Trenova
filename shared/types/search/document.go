package search

import validation "github.com/go-ozzo/ozzo-validation/v4"

type Document struct {
	ID             string         `json:"id"`
	EntityType     EntityType     `json:"entityType"`
	OrganizationID string         `json:"organizationId"`
	BusinessUnitID string         `json:"businessUnitId"`
	Title          string         `json:"title"`
	Subtitle       string         `json:"subtitle"`
	Content        string         `json:"content"`
	Metadata       map[string]any `json:"metadata"` // Unstructured data specific to the entity type
	CreatedAt      int64          `json:"createdAt"`
	UpdatedAt      int64          `json:"updatedAt"`
}

func (d *Document) Validate() error {
	return validation.ValidateStruct(d,
		validation.Field(
			&d.ID,
			validation.Required.Error("ID is required"),
		),
		validation.Field(
			&d.EntityType,
			validation.Required.Error("Entity type is required"),
		),
		validation.Field(
			&d.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&d.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&d.Title,
			validation.Required.Error("Title is required"),
		),
	)
}
