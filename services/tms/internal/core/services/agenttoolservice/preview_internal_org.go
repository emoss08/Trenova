package agenttoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func warnWouldFail(preview *agent.ToolPreview, err error) *agent.ToolPreview {
	if err == nil {
		return preview
	}

	return toolpreview.Warn(
		preview,
		agent.PreviewWarningWouldFail,
		"This would be refused as it stands: "+strings.TrimSpace(err.Error()),
	)
}

func warnRefusal(preview *agent.ToolPreview, err error) (*agent.ToolPreview, error) {
	switch {
	case err == nil:
		return preview, nil
	case isRefusal(err):
		return warnWouldFail(preview, err), nil
	default:
		return nil, err
	}
}

func isRefusal(err error) bool {
	return errortypes.IsError(err) ||
		errortypes.IsBusinessError(err) ||
		errortypes.IsConflictError(err) ||
		errortypes.IsVersionMismatchError(err) ||
		errortypes.IsAuthorizationError(err)
}

func previewVersion(version int64) *int64 {
	return &version
}

func labelRefs(change *agent.RecordChange, labels map[string]string) {
	if change == nil {
		return
	}
	for i := range change.Fields {
		field := &change.Fields[i]
		label, ok := labels[field.Path]
		if !ok || strings.TrimSpace(label) == "" {
			continue
		}
		if field.AfterRef != nil {
			field.AfterRef.Label = label
		}
	}
}

type plannedChange struct {
	change  *agent.RecordChange
	refused error
}

func planUpdate[T any](
	rec toolpreview.Record,
	before *T,
	plan func(*T) error,
	opts ...toolpreview.Option,
) (plannedChange, error) {
	return planned(toolpreview.Update[T], rec, before, plan, opts)
}

func planArchive[T any](
	rec toolpreview.Record,
	before *T,
	plan func(*T) error,
	opts ...toolpreview.Option,
) (plannedChange, error) {
	return planned(toolpreview.Archive[T], rec, before, plan, opts)
}

func planned[T any](
	build func(toolpreview.Record, *T, func(*T) error, ...toolpreview.Option) (*agent.RecordChange, error),
	rec toolpreview.Record,
	before *T,
	plan func(*T) error,
	opts []toolpreview.Option,
) (plannedChange, error) {
	var refused error
	change, err := build(rec, before, func(after *T) error {
		refused = plan(after)

		return refused
	}, opts...)
	if refused == nil && err != nil {
		return plannedChange{}, err
	}

	return plannedChange{change: change, refused: refused}, nil
}

// accepted reports whether the plan would run; a refused one previews as a
// would-fail warning.
func (p plannedChange) accepted() bool {
	return p.refused == nil
}

func (p plannedChange) preview(
	summary string,
	extra ...*agent.RecordChange,
) *agent.ToolPreview {
	if p.refused != nil {
		return warnWouldFail(toolpreview.Build(summary), p.refused)
	}

	changes := make([]*agent.RecordChange, 0, len(extra)+1)
	changes = append(changes, p.change)
	changes = append(changes, extra...)

	return toolpreview.Build(summary, changes...)
}

func countOf(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", n, noun)
}
