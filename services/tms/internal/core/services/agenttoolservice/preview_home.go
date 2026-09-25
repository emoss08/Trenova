package agenttoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/homelayoutservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var (
	_ serviceports.ToolPreviewer = (*addHomeWidgetTool)(nil)
	_ serviceports.ToolPreviewer = (*removeHomeWidgetTool)(nil)
	_ serviceports.ToolPreviewer = (*arrangeHomeLayoutTool)(nil)
)

const homePageLabel = "Your home page"

var homePageLabels = map[string]string{"widgets": "Widgets, in order"}

type homePlan struct {
	current *homelayoutservice.EffectiveLayout
	update  *homelayoutservice.UpdateRequest
}

type homePageView struct {
	Widgets []string `json:"widgets"`
}

func homePageOf(layout *homelayout.Layout) *homePageView {
	view := &homePageView{Widgets: []string{}}
	if layout == nil {
		return view
	}

	view.Widgets = make([]string, 0, len(layout.Widgets))
	for idx := range layout.Widgets {
		view.Widgets = append(view.Widgets, widgetLabel(&layout.Widgets[idx]))
	}

	return view
}

func (p *homePlan) change() (*agent.RecordChange, error) {
	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceHomeLayoutPreset,
		Label:    homePageLabel,
		Version:  previewVersion(p.current.Version),
	}, homePageOf(p.current.Layout), homePageOf(p.update.Document.Layout),
		toolpreview.Labels(homePageLabels),
	)
}

func (p *homePlan) summary() string {
	after := homePageOf(p.update.Document.Layout)
	summary := "The home page would show, in order: " + strings.Join(after.Widgets, ", ") + "."
	if p.current.Source != homelayoutservice.SourceUser {
		summary += " It would become your own layout rather than the one assigned to you."
	}

	return summary
}

func (e homeEditor) preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
	edit homeEdit,
) (*agent.ToolPreview, error) {
	plan, err := e.plan(ctx, params, edit)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build("Would change your home page."), err), nil
		}

		return nil, err
	}

	change, err := plan.change()
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(plan.summary(), change), nil
}

func (t *addHomeWidgetTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	return t.editor.preview(ctx, params, t.edit(params))
}

func (t *removeHomeWidgetTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	return t.editor.preview(ctx, params, t.edit(params))
}

func (t *arrangeHomeLayoutTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	return t.editor.preview(ctx, params, t.edit(params))
}
