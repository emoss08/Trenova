package meilisearch

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/shared/pulid"
	meili "github.com/meilisearch/meilisearch-go"
)

const (
	organizationAttribute = "organization_id"
	indexPage             = 100
	maxIndexes            = 1000
)

type IndexPurge struct {
	Index  string
	TaskID int64
}

func (c *Client) DeleteOrganizationDocuments(
	ctx context.Context,
	organizationID pulid.ID,
) ([]IndexPurge, error) {
	if !c.Enabled() || organizationID.IsNil() {
		return []IndexPurge{}, nil
	}
	filter := organizationAttribute + " = " + strconv.Quote(organizationID.String())
	purged := make([]IndexPurge, 0)
	for offset := int64(0); offset < maxIndexes; offset += indexPage {
		page, err := c.client.ListIndexesWithContext(ctx, &meili.IndexesQuery{
			Limit:  indexPage,
			Offset: offset,
		})
		if err != nil {
			return purged, fmt.Errorf("list the search indexes: %w", err)
		}
		for _, idx := range page.Results {
			index := c.client.Index(idx.UID)
			filterable, ferr := index.GetFilterableAttributesWithContext(ctx)
			if ferr != nil {
				return purged, fmt.Errorf("read the filterable attributes of %s: %w", idx.UID, ferr)
			}
			if filterable == nil || !slices.ContainsFunc(*filterable, isOrganizationAttribute) {
				continue
			}
			task, derr := index.DeleteDocumentsByFilterWithContext(ctx, filter, nil)
			if derr != nil {
				return purged, fmt.Errorf(
					"delete the organization's documents from %s: %w",
					idx.UID,
					derr,
				)
			}
			purged = append(purged, IndexPurge{Index: idx.UID, TaskID: task.TaskUID})
		}
		if int64(len(page.Results)) < indexPage {
			break
		}
	}
	return purged, nil
}

func isOrganizationAttribute(attribute any) bool {
	name, ok := attribute.(string)
	return ok && name == organizationAttribute
}
