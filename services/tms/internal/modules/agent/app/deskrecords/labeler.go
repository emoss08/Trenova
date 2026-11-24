package deskrecords

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Readers     *boardreaderports.Index
	Permissions permissionports.PermissionEngine
	Fields      permissionports.FieldAccess
}

type Labeler struct {
	readers     *boardreaderports.Index
	permissions permissionports.PermissionEngine
	fields      permissionports.FieldAccess
	now         func() int64
}

func New(p Params) *Labeler {
	return &Labeler{
		readers:     p.Readers,
		permissions: p.Permissions,
		fields:      p.Fields,
		now:         timeutils.NowUnix,
	}
}

func (l *Labeler) Label(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
	records []*deskattentionports.Record,
) error {
	byKind := map[recordkind.Kind][]pulid.ID{}
	for _, record := range records {
		if record == nil || record.Kind.Resource() == "" {
			continue
		}
		if !slices.Contains(byKind[record.Kind], record.ID) {
			byKind[record.Kind] = append(byKind[record.Kind], record.ID)
		}
	}
	if len(byKind) == 0 {
		return nil
	}

	allowed := make(map[recordkind.Kind]bool, len(byKind))
	summaries := make(map[recordkind.Kind]map[pulid.ID]boardreading.Summary, len(byKind))
	visible := permissionports.VisibleFieldGroups(ctx, l.fields, actor)
	for kind, ids := range byKind {
		result, err := l.permissions.Check(
			ctx,
			actor.PermissionCheck(kind.Resource(), permission.OpRead),
		)
		if err != nil {
			return fmt.Errorf("check %s read: %w", kind.Resource(), err)
		}
		if result == nil || !result.Allowed {
			continue
		}
		allowed[kind] = true
		reader := l.readers.Records[kind]
		if reader == nil {
			continue
		}
		read, err := reader.Summaries(ctx, &boardreaderports.RecordRequest{
			Tenant:  tenant,
			IDs:     ids,
			Now:     l.now(),
			Visible: visible,
		})
		if err != nil {
			return err
		}
		summaries[kind] = read
	}

	for _, record := range records {
		if record == nil {
			continue
		}
		record.Readable = allowed[record.Kind]
		if !record.Readable {
			record.Label, record.Subtitle, record.Status = "", "", ""
			continue
		}
		if summary, ok := summaries[record.Kind][record.ID]; ok {
			record.Label = summary.Label
			record.Subtitle = summary.Subtitle
			record.Status = summary.Status
		}
	}

	return nil
}
