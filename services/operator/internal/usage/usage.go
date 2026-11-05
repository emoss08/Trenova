package usage

import (
	"context"
	"time"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	MaxReadings = 1000
	Lookback    = 2 * time.Hour
)

type ControlPlane interface {
	ReportUsage(ctx context.Context, report instanceapi.UsageReport) (instanceapi.UsageAck, error)
}

type TMS interface {
	Readings(ctx context.Context, since time.Time) ([]tms.Reading, error)
	PutState(ctx context.Context, state tms.State) error
}

type Relay struct {
	cp  ControlPlane
	tms TMS
	now func() time.Time
}

type Params struct {
	ControlPlane ControlPlane
	TMS          TMS
	Now          func() time.Time
}

func New(p Params) *Relay {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Relay{cp: p.ControlPlane, tms: p.TMS, now: now}
}

type Result struct {
	Readings int
	Dropped  int
	Meters   int
}

func (r *Relay) Run(ctx context.Context, allowances []instanceapi.SpecAllowance) (Result, error) {
	m := newMapping(allowances)
	since := r.now().UTC().Truncate(time.Hour).Add(-Lookback)
	readings, err := r.tms.Readings(ctx, since)
	if err != nil {
		return Result{}, err
	}
	mapped, dropped := m.readings(readings)
	var ack instanceapi.UsageAck
	for _, chunk := range chunks(mapped, MaxReadings) {
		if ack, err = r.cp.ReportUsage(ctx, instanceapi.UsageReport{Readings: chunk}); err != nil {
			return Result{}, err
		}
	}
	state := m.state(ack)
	if err = r.tms.PutState(ctx, state); err != nil {
		return Result{}, err
	}
	return Result{Readings: len(mapped), Dropped: dropped, Meters: len(state.Meters)}, nil
}

type mapping struct {
	toCatalog map[string]string
	toApp     map[string]string
}

func newMapping(allowances []instanceapi.SpecAllowance) mapping {
	m := mapping{
		toCatalog: make(map[string]string, len(allowances)),
		toApp:     make(map[string]string, len(allowances)),
	}
	for _, a := range allowances {
		if a.AppMeter == nil || *a.AppMeter == "" {
			continue
		}
		if _, seen := m.toCatalog[*a.AppMeter]; seen {
			continue
		}
		if _, seen := m.toApp[a.Meter]; seen {
			continue
		}
		m.toCatalog[*a.AppMeter] = a.Meter
		m.toApp[a.Meter] = *a.AppMeter
	}
	return m
}

func (m mapping) readings(in []tms.Reading) ([]instanceapi.UsageReading, int) {
	out := make([]instanceapi.UsageReading, 0, len(in))
	dropped := 0
	for _, rd := range in {
		meter, ok := m.toCatalog[rd.Meter]
		if !ok || rd.Quantity < 0 {
			dropped++
			continue
		}
		out = append(out, instanceapi.UsageReading{
			Meter:    meter,
			Hour:     rd.Hour.UTC(),
			Quantity: rd.Quantity,
		})
	}
	return out, dropped
}

func (m mapping) state(ack instanceapi.UsageAck) tms.State {
	state := tms.State{Overage: ack.Overage, Meters: make([]tms.Meter, 0, len(ack.Meters))}
	if ack.Window != nil {
		state.Window = &tms.Window{Start: ack.Window.Start.UTC(), End: ack.Window.End.UTC()}
	}
	for _, mu := range ack.Meters {
		app, ok := m.toApp[mu.Meter]
		if !ok {
			continue
		}
		state.Meters = append(state.Meters, tms.Meter{
			Meter:     app,
			Unit:      string(mu.Unit),
			Quantity:  mu.Quantity,
			Allowance: mu.Allowance,
			HardCap:   mu.HardCap,
			Stopped:   mu.Stopped,
		})
	}
	return state
}

func chunks(readings []instanceapi.UsageReading, size int) [][]instanceapi.UsageReading {
	size = max(size, 1)
	if len(readings) == 0 {
		return [][]instanceapi.UsageReading{{}}
	}
	out := make([][]instanceapi.UsageReading, 0, (len(readings)+size-1)/size)
	for start := 0; start < len(readings); start += size {
		out = append(out, readings[start:min(start+size, len(readings))])
	}
	return out
}
