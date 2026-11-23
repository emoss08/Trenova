package recordnumbers

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordnumberports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Records     recordnumberports.RecordNumberReader
	Permissions permissionports.PermissionEngine
}

type Resolver struct {
	records     recordnumberports.RecordNumberReader
	permissions permissionports.PermissionEngine
	registry    *permissioncatalog.Registry
}

func New(p Params) *Resolver {
	return &Resolver{
		records:     p.Records,
		permissions: p.Permissions,
		registry:    fieldsensitivity.DefaultRegistry(),
	}
}

type Request struct {
	Actor      *permissionports.RequestActor
	DataAccess permission.FieldSensitivity
	Contract   argcheck.Contract
}

func (r *Resolver) Resolve(ctx context.Context, req *Request) (argcheck.Contract, error) {
	refs := req.Contract.Numbers()
	if len(refs) == 0 {
		return req.Contract, nil
	}

	numbered := r.readableNumberings(ctx, req, refs)
	asked := make(map[recordkind.Kind][]string, len(numbered))
	for _, ref := range refs {
		for _, kind := range ref.Kinds {
			if _, ok := numbered[kind]; ok {
				asked[kind] = append(asked[kind], ref.Number)
			}
		}
	}

	found := recordnumberports.NumberedRecords{}
	if len(asked) > 0 {
		var err error
		found, err = r.records.FindNumbers(ctx, req.Actor.TenantInfo(), asked)
		if err != nil {
			return argcheck.Contract{}, fmt.Errorf("read the records these numbers name: %w", err)
		}
	}

	candidates := make(map[string][]argcheck.NumberCandidate, len(refs))
	for _, ref := range refs {
		candidates[ref.Path] = candidatesFor(ref, numbered, found)
	}

	return req.Contract.ReadNumbers(candidates)
}

func (r *Resolver) readableNumberings(
	ctx context.Context,
	req *Request,
	refs []argcheck.NumberRef,
) map[recordkind.Kind]recordnumberports.Numbering {
	ceilings := fieldsensitivity.NewAgentCeilings(r.permissions, req.Actor, req.DataAccess)
	seen := make(map[recordkind.Kind]struct{}, len(refs))
	numbered := make(map[recordkind.Kind]recordnumberports.Numbering, len(refs))
	for _, ref := range refs {
		for _, kind := range ref.Kinds {
			if _, done := seen[kind]; done {
				continue
			}
			seen[kind] = struct{}{}

			if numbering, ok := r.readable(ctx, ceilings, kind); ok {
				numbered[kind] = numbering
			}
		}
	}

	return numbered
}

func candidatesFor(
	ref argcheck.NumberRef,
	numbered map[recordkind.Kind]recordnumberports.Numbering,
	found recordnumberports.NumberedRecords,
) []argcheck.NumberCandidate {
	seen := make(map[pulid.ID]struct{}, 1)
	var candidates []argcheck.NumberCandidate
	for _, kind := range ref.Kinds {
		numbering, ok := numbered[kind]
		if !ok {
			continue
		}
		for _, record := range found[kind][ref.Number] {
			if _, dup := seen[record.ID]; dup {
				continue
			}
			seen[record.ID] = struct{}{}
			candidates = append(candidates, argcheck.NumberCandidate{
				Kind:       kind,
				ID:         record.ID,
				NumberName: numbering.Name,
				Number:     record.Number,
			})
		}
	}

	return candidates
}
