package providers

import (
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/meilisearchtype"
)

var ErrNilEntity = errors.New("entity is nil")

type BaseProvider struct{}

func (b *BaseProvider) ToSearchDocument(
	entity meilisearchtype.Searchable,
) (*meilisearchtype.SearchDocument, error) {
	if entity == nil {
		return nil, ErrNilEntity
	}

	createdAt, updatedAt := entity.GetSearchTimestamps()

	doc := &meilisearchtype.SearchDocument{
		ID:             entity.GetID(),
		EntityType:     entity.GetSearchEntityType(),
		OrganizationID: entity.GetOrganizationID().String(),
		BusinessUnitID: entity.GetBusinessUnitID().String(),
		Title:          entity.GetSearchTitle(),
		Subtitle:       entity.GetSearchSubtitle(),
		Content:        entity.GetSearchContent(),
		Metadata:       entity.GetSearchMetadata(),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	if err := doc.Validate(); err != nil {
		return nil, err
	}

	return doc, nil
}

func (b *BaseProvider) ToSearchDocuments(
	entities []meilisearchtype.Searchable,
) ([]*meilisearchtype.SearchDocument, error) {
	if len(entities) == 0 {
		return []*meilisearchtype.SearchDocument{}, nil
	}

	documents := make([]*meilisearchtype.SearchDocument, 0, len(entities))

	for i, entity := range entities {
		doc, err := b.ToSearchDocument(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity at index %d: %w", i, err)
		}
		documents = append(documents, doc)
	}

	return documents, nil
}
