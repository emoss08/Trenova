package deskrecords

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordsummaryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Summaries   recordsummaryports.RecordSummarizer
	Permissions permissionports.PermissionEngine
}

type Labeler struct {
	summaries   recordsummaryports.RecordSummarizer
	permissions permissionports.PermissionEngine
	now         func() int64
}

func New(p Params) *Labeler {
	return &Labeler{summaries: p.Summaries, permissions: p.Permissions, now: timeutils.NowUnix}
}

func (l *Labeler) Label(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
	records []*deskattentionports.Record,
) error {
	byResource := map[permission.Resource][]pulid.ID{}
	for _, record := range records {
		if record == nil {
			continue
		}
		resource := record.Kind.Resource()
		if resource == "" {
			continue
		}
		byResource[resource] = append(byResource[resource], record.ID)
	}
	if len(byResource) == 0 {
		return nil
	}

	readable := make(map[permission.Resource][]pulid.ID, len(byResource))
	allowed := make(map[permission.Resource]bool, len(byResource))
	for resource, ids := range byResource {
		result, err := l.permissions.Check(ctx, actor.PermissionCheck(resource, permission.OpRead))
		if err != nil {
			return fmt.Errorf("check %s read: %w", resource, err)
		}
		if result != nil && result.Allowed {
			allowed[resource] = true
			readable[resource] = ids
		}
	}

	summaries, err := l.summaries.Summaries(ctx, &recordsummaryports.RecordSummaryRequest{
		Tenant: tenant,
		Refs:   readable,
		Now:    l.now(),
	})
	if err != nil {
		return err
	}

	for _, record := range records {
		if record == nil {
			continue
		}
		resource := record.Kind.Resource()
		record.Readable = allowed[resource]
		if !record.Readable {
			record.Label, record.Subtitle, record.Status = "", "", ""
			continue
		}
		if summary, ok := summaries[resource][record.ID]; ok {
			record.Label = summary.Label
			record.Subtitle = summary.Subtitle
			record.Status = summary.Status
		}
	}

	return nil
}
