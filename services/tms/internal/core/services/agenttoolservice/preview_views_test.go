package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/services/tablequeryservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingViews struct {
	saved *tableconfiguration.TableConfiguration
	guard writeGuard
}

func (f *savingViews) Create(
	_ context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	entity.ID = pulid.MustNew("tc_")
	f.saved = entity

	return entity, nil
}

type composingViews struct {
	composed int
}

func (f *composingViews) Compose(
	context.Context,
	*tablequeryservice.ComposeRequest,
) (*tablequeryservice.ComposeResult, error) {
	f.composed++

	return &tablequeryservice.ComposeResult{
		Explanation:  "Shipments with status In transit",
		FieldFilters: []domaintypes.FieldFilter{{Field: "status", Value: "InTransit"}},
	}, nil
}

func savedViewTool(views *savingViews, composer *composingViews) *saveTableViewTool {
	return &saveTableViewTool{
		configs:  views,
		composer: composer,
		catalog: filtercatalog.New(filtercatalog.Resource{
			Resource: permission.ResourceShipment,
			Entity:   "shipments",
		}),
	}
}

func TestSaveTableView_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	views := &savingViews{}
	composer := &composingViews{}
	tool := savedViewTool(views, composer)
	params := executeParams(map[string]any{
		"entity":      "shipments",
		"name":        "In transit",
		"description": "loads that are on the road",
		"shared":      true,
	})

	preview := previewWithoutWrites(t, &views.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Zero(t, composer.composed, "a preview does not ask a model to compose the filters")
	assert.True(t, preview.Partial)
	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceTableConfiguration, change.Resource)
	assert.Equal(t, "Table", fieldByPath(t, change, "resource").Label)
	assert.Equal(t, "Public", fieldByPath(t, change, "visibility").After)
	assert.Contains(t, preview.Summary, "loads that are on the road")

	require.NoError(t, tool.Execute(t.Context(), params))
	requireCreateParity(t, change, views.saved, savedViewOptions()...)
}

func TestSaveTableView_PreviewWarnsOnAnUnknownTable(t *testing.T) {
	t.Parallel()

	views := &savingViews{}
	tool := savedViewTool(views, &composingViews{})

	preview := previewWithoutWrites(t, &views.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			"entity":      "spaceships",
			"name":        "x",
			"description": "y",
		}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
