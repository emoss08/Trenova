package emailevents

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const MaxBatch = 50

type ControlPlane interface {
	EmailEvents(ctx context.Context) ([]instanceapi.EmailEvent, error)
	AckEmailEvents(ctx context.Context, ids []uuid.UUID) error
}

type TMS interface {
	RelayEmailEvents(ctx context.Context, events []tms.EmailEvent) ([]string, error)
}

type Result struct {
	Listed  int
	Handled int
}

type Relay struct {
	cp     ControlPlane
	tms    TMS
	logger *slog.Logger
}

type Params struct {
	ControlPlane ControlPlane
	TMS          TMS
	Logger       *slog.Logger
}

func New(p *Params) (*Relay, error) {
	var missing []string
	if p.ControlPlane == nil {
		missing = append(missing, "the control plane")
	}
	if p.TMS == nil {
		missing = append(missing, "the TMS")
	}
	if p.Logger == nil {
		missing = append(missing, "a logger")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("emailevents: the relay needs %s", strings.Join(missing, ", "))
	}
	return &Relay{cp: p.ControlPlane, tms: p.TMS, logger: p.Logger}, nil
}

func (r *Relay) Run(ctx context.Context) (Result, error) {
	listed, err := r.cp.EmailEvents(ctx)
	if err != nil {
		return Result{}, err
	}
	if len(listed) > MaxBatch {
		listed = listed[:MaxBatch]
	}
	if len(listed) == 0 {
		return Result{}, nil
	}
	sent := make(map[string]uuid.UUID, len(listed))
	events := make([]tms.EmailEvent, 0, len(listed))
	for _, e := range listed {
		id := e.Id.String()
		sent[id] = e.Id
		events = append(events, tms.EmailEvent{
			ID:       id,
			Provider: string(e.Provider),
			Payload:  e.Payload,
		})
	}
	handled, err := r.tms.RelayEmailEvents(ctx, events)
	if err != nil {
		return Result{Listed: len(listed)}, fmt.Errorf(
			"%d system email events not delivered to the TMS: %w",
			len(listed),
			err,
		)
	}
	ack := make([]uuid.UUID, 0, len(handled))
	for _, id := range handled {
		known, ok := sent[id]
		if !ok {
			r.logger.WarnContext(ctx, "the TMS named a system email event it was not sent; it is not acknowledged",
				"event", id)
			continue
		}
		delete(sent, id)
		ack = append(ack, known)
	}
	result := Result{Listed: len(listed), Handled: len(ack)}
	if len(sent) > 0 {
		r.logger.WarnContext(ctx, "the TMS left system email events to retry",
			"events", slices.Sorted(maps.Keys(sent)))
	}
	if len(ack) == 0 {
		return result, nil
	}
	if err = r.cp.AckEmailEvents(ctx, ack); err != nil {
		return result, fmt.Errorf("acknowledge %d system email events: %w", len(ack), err)
	}
	return result, nil
}
