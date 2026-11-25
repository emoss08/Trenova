package recordlabelrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// NewRecordFinder is the labeler as a wait reads it: whether a record it
// names is the organization's, and who changed one.
func NewRecordFinder(p Params) agentwaitports.RecordFinder {
	return &Labeler{db: p.DB}
}

// Find reads one record of a kind the labeler knows. A record whose label is
// empty is still there: the row decides, not the label.
func (l *Labeler) Find(
	ctx context.Context,
	tenant pagination.TenantInfo,
	resource permission.Resource,
	id pulid.ID,
) (*agentwaitports.FoundRecord, error) {
	return dbtx.Read(ctx, l.db, func(ctx context.Context) (*agentwaitports.FoundRecord, error) {
		if src, ok := sources[resource]; ok {
			rows, err := keel.Into[labelRow](ctx, l.db.Keel(ctx), src([]pulid.ID{id}))
			if err != nil {
				return nil, err
			}
			if len(rows) == 0 {
				return &agentwaitports.FoundRecord{Known: true}, nil
			}
			return l.withStatus(ctx, resource, id, &agentwaitports.FoundRecord{
				Known:  true,
				Exists: true,
				Label:  strings.TrimSpace(rows[0].Label),
			})
		}

		read := l.reader(resource)
		if read == nil {
			return &agentwaitports.FoundRecord{}, nil
		}
		found, err := read(ctx, tenant, []pulid.ID{id})
		if err != nil {
			return nil, err
		}
		label, exists := found[id]
		if !exists {
			return &agentwaitports.FoundRecord{Known: true}, nil
		}

		return l.withStatus(ctx, resource, id,
			&agentwaitports.FoundRecord{Known: true, Exists: true, Label: label})
	})
}

func (l *Labeler) withStatus(
	ctx context.Context,
	resource permission.Resource,
	id pulid.ID,
	found *agentwaitports.FoundRecord,
) (*agentwaitports.FoundRecord, error) {
	status, ok := statusSources[resource]
	if !ok {
		return found, nil
	}
	rows, err := keel.Into[labelRow](ctx, l.db.Keel(ctx), status(id))
	if err != nil {
		return nil, err
	}
	found.HasStatus = true
	if len(rows) > 0 {
		found.Status = strings.TrimSpace(rows[0].Label)
	}

	return found, nil
}

func (l *Labeler) UserNames(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	ids = distinctIDs(ids)
	if len(ids) == 0 {
		return map[pulid.ID]string{}, nil
	}

	return l.userLabels(ctx, tenant, ids)
}
