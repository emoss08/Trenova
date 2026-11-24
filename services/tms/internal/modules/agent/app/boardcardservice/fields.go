package boardcardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/readscope"
)

type fieldReach struct {
	groups   protectedfield.GroupSet
	ceilings *fieldsensitivity.AgentCeilings
	registry *permissioncatalog.Registry
}

func (s *Service) reachOf(
	ctx context.Context,
	req *assistantports.BoardFillRequest,
) *fieldReach {
	held := permissionports.VisibleFieldGroups(ctx, s.fields, req.Actor)
	reach := &fieldReach{groups: held, registry: s.registry}
	if req.Agent == nil {
		return reach
	}
	reach.ceilings = fieldsensitivity.NewAgentCeilings(s.engine, req.Actor, req.Agent.DataAccess)
	reach.groups = fieldsensitivity.AgentVisibleGroups(ctx, held, reach.ceilings.For)

	return reach
}

func (r *fieldReach) shows(ctx context.Context, fields []boardcatalog.Field) bool {
	for _, field := range fields {
		group, protected := protectedfield.GroupAt(field.Resource, []string{field.Name}, nil)
		if protected && !r.groups.Has(group) {
			return false
		}
		if r.ceilings != nil && !fieldsensitivity.Visible(
			r.registry, field.Resource, field.Name, r.ceilings.For(ctx, field.Resource),
		) {
			return false
		}
	}

	return true
}

func measureFields(kind recordkind.Kind, measure *boardcatalog.Measure) []boardcatalog.Field {
	if len(measure.Reads) > 0 {
		return measure.Reads
	}

	return boardcatalog.Read(kind.Resource(), "")
}

func metricFields(metric *boardcatalog.Metric) []boardcatalog.Field {
	if len(metric.Reads) > 0 {
		return metric.Reads
	}
	fields := make([]boardcatalog.Field, 0, len(metric.Requires))
	for _, resource := range metric.Requires {
		fields = append(fields, boardcatalog.Field{Resource: resource})
	}

	return fields
}

func (s *Service) scoped(ctx context.Context, actor *permissionports.RequestActor) context.Context {
	return readscope.With(ctx, s.scopes.Resolver(actor.PermissionCheck("", "")))
}

func wholeReadable(ctx context.Context, metric *boardcatalog.Metric) (bool, error) {
	for _, kind := range metric.Unscoped {
		narrowed, err := readscope.Narrows(ctx, kind.String())
		if err != nil || narrowed {
			return false, err
		}
	}

	return true, nil
}
