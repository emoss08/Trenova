package masterdatatools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/tablequeryservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tableconfiguration"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingViews struct {
	saved *tableconfiguration.TableConfiguration
	guard agenttoolstest.WriteGuard
}

func (f *savingViews) Create(
	_ context.Context,
	entity *tableconfiguration.TableConfiguration,
	_ *permissionports.RequestActor,
) (*tableconfiguration.TableConfiguration, error) {
	if err := f.guard.Write(); err != nil {
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

func savedViewTool(views *savingViews, composer *composingViews) *SaveTableViewTool {
	return &SaveTableViewTool{
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
	params := agenttoolstest.ExecuteParams(map[string]any{
		"entity":      "shipments",
		"name":        "In transit",
		"description": "loads that are on the road",
		"shared":      true,
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&views.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Zero(t, composer.composed, "a preview does not ask a model to compose the filters")
	assert.True(t, preview.Partial)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceTableConfiguration, change.Resource)
	assert.Equal(t, "Table", agenttoolstest.FieldByPath(t, change, "resource").Label)
	assert.Equal(t, "Public", agenttoolstest.FieldByPath(t, change, "visibility").After)
	assert.Contains(t, preview.Summary, "loads that are on the road")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, views.saved, savedViewOptions()...)
}

func TestSaveTableView_PreviewWarnsOnAnUnknownTable(t *testing.T) {
	t.Parallel()

	views := &savingViews{}
	tool := savedViewTool(views, &composingViews{})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&views.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"entity":      "spaceships",
				"name":        "x",
				"description": "y",
			}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}
