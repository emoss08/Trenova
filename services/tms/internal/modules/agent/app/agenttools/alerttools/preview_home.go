package alerttools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/homelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
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
		Version:  changepreview.PreviewVersion(p.current.Version),
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

func (e homeEditor) PlanPreview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	edit homeEdit,
) (*agent.ToolPreview, error) {
	plan, err := e.plan(ctx, *params, edit)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would change your home page."),
				err,
			), nil
		}

		return nil, err
	}

	change, err := plan.change()
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(plan.summary(), change), nil
}

func (t *AddHomeWidgetTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.editor.PlanPreview(ctx, &params, t.edit(params))
}

func (t *RemoveHomeWidgetTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.editor.PlanPreview(ctx, &params, t.edit(params))
}

func (t *ArrangeHomeLayoutTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.editor.PlanPreview(ctx, &params, t.edit(params))
}
