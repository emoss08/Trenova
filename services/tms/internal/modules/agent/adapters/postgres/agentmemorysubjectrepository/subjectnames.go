package agentmemorysubjectrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxSubjectCandidates = 10
	maxNameWords         = 4
)

type namedRow struct {
	ID   pulid.ID
	Name string
	Code string
}

type personRow struct {
	ID        pulid.ID
	FirstName string
	LastName  string
}

func (r *repository) FindSubjectsByName(
	ctx context.Context,
	req *agentmemoryports.FindMemorySubjectsRequest,
) ([]agentmemoryports.MemorySubjectCandidate, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return []agentmemoryports.MemorySubjectCandidate{}, nil
	}
	limit := req.Limit
	if limit <= 0 || limit > maxSubjectCandidates {
		limit = maxSubjectCandidates
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentmemoryports.MemorySubjectCandidate, error) {
			switch req.Type {
			case agentmemory.MemorySubjectCustomer:
				c := customerdb.Customers

				return r.named(ctx, c.Pick(c.ID, c.Name, c.Code).
					Where(keel.Or(c.Name.ContainsFold(name), c.Code.ContainsFold(name))).
					Order(c.Name.Asc()).Limit(limit))
			case agentmemory.MemorySubjectLocation:
				l := locationdb.Locations

				return r.named(ctx, l.Pick(l.ID, l.Name, l.Code).
					Where(keel.Or(l.Name.ContainsFold(name), l.Code.ContainsFold(name))).
					Order(l.Name.Asc()).Limit(limit))
			case agentmemory.MemorySubjectCarrier:
				c := carrierdb.Carriers

				return r.named(ctx, c.Pick(c.ID, c.Name, c.Code).
					Where(keel.Or(
						c.Name.ContainsFold(name),
						c.DBAName.ContainsFold(name),
						c.Code.ContainsFold(name),
					)).
					Order(c.Name.Asc()).Limit(limit))
			case agentmemory.MemorySubjectWorker:
				return r.people(ctx, name, limit)
			default:
				return nil, fmt.Errorf("%q is not a subject a memory can be about", req.Type)
			}
		},
	)
}

func (r *repository) named(
	ctx context.Context,
	q keel.Projection,
) ([]agentmemoryports.MemorySubjectCandidate, error) {
	var rows []namedRow
	if err := keel.Batch(ctx, r.db.Keel(ctx), keel.IntoSlice(q, &rows)); err != nil {
		return nil, fmt.Errorf("find memory subjects by name: %w", err)
	}

	out := make([]agentmemoryports.MemorySubjectCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentmemoryports.MemorySubjectCandidate{
			ID:    row.ID,
			Label: row.Name,
			Code:  row.Code,
		})
	}

	return out, nil
}

func (r *repository) people(
	ctx context.Context,
	name string,
	limit int,
) ([]agentmemoryports.MemorySubjectCandidate, error) {
	w := workerdb.Workers
	words := strings.Fields(name)
	if len(words) > maxNameWords {
		words = words[:maxNameWords]
	}
	preds := make([]keel.Pred, 0, len(words))
	for _, word := range words {
		preds = append(
			preds,
			keel.Or(w.FirstName.ContainsFold(word), w.LastName.ContainsFold(word)),
		)
	}

	var rows []personRow
	q := w.Pick(w.ID, w.FirstName, w.LastName).
		Where(preds...).
		Order(w.LastName.Asc(), w.FirstName.Asc()).
		Limit(limit)
	if err := keel.Batch(ctx, r.db.Keel(ctx), keel.IntoSlice(q, &rows)); err != nil {
		return nil, fmt.Errorf("find workers by name: %w", err)
	}

	out := make([]agentmemoryports.MemorySubjectCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentmemoryports.MemorySubjectCandidate{
			ID:    row.ID,
			Label: strings.TrimSpace(row.FirstName + " " + row.LastName),
		})
	}

	return out, nil
}
