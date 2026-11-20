package usage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const MaxSharedReadings = 5000

type SharedControlPlane interface {
	ReportSharedUsage(
		ctx context.Context,
		report instanceapi.SharedUsageReport,
	) (instanceapi.SharedUsageAck, error)
}

type SharedTMS interface {
	TenantReadings(ctx context.Context, from, to time.Time) ([]tms.TenantReadings, error)
	PutTenantUsage(ctx context.Context, tenants []tms.TenantUsage) (tms.TenantUsageAnswer, error)
}

type SharedRelay struct {
	cp  SharedControlPlane
	tms SharedTMS
	now func() time.Time
}

type SharedParams struct {
	ControlPlane SharedControlPlane
	TMS          SharedTMS
	Now          func() time.Time
}

func NewShared(p SharedParams) *SharedRelay {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &SharedRelay{cp: p.ControlPlane, tms: p.TMS, now: now}
}

type SharedResult struct {
	Readings      int
	Dropped       int
	Answered      int
	RefusedByCP   []string
	RefusedByTMS  []string
	TenantsHeard  int
	TenantsMapped int
}

func (r *SharedRelay) Run(
	ctx context.Context,
	tenants []instanceapi.SharedTenant,
) (SharedResult, error) {
	mappings := make(map[uuid.UUID]mapping, len(tenants))
	for i := range tenants {
		mappings[tenants[i].InstanceId] = newMapping(tenants[i].Allowances)
	}
	to := r.now().UTC().Truncate(time.Hour)
	held, err := r.tms.TenantReadings(ctx, to.Add(-Lookback), to)
	if err != nil {
		return SharedResult{}, err
	}

	var result SharedResult
	readings := make([]instanceapi.SharedUsageReading, 0)
	for _, t := range held {
		id, perr := uuid.Parse(t.InstanceID)
		m, placed := mappings[id]
		if perr != nil || !placed {
			result.Dropped += len(t.Readings)
			continue
		}
		mapped, dropped := m.readings(asReadings(t.Readings))
		result.Dropped += dropped
		for _, rd := range mapped {
			readings = append(readings, instanceapi.SharedUsageReading{
				InstanceId: id,
				Meter:      rd.Meter,
				Hour:       rd.Hour,
				Quantity:   rd.Quantity,
			})
		}
	}
	result.Readings = len(readings)

	var ack instanceapi.SharedUsageAck
	refused := make(map[string]struct{})
	for _, chunk := range chunks(readings, MaxSharedReadings) {
		ack, err = r.cp.ReportSharedUsage(ctx, instanceapi.SharedUsageReport{Readings: chunk})
		if err != nil {
			return result, err
		}
		for _, e := range ack.Refused {
			refused[e.InstanceId.String()] = struct{}{}
		}
	}
	for id := range refused {
		result.RefusedByCP = append(result.RefusedByCP, id)
	}
	slices.Sort(result.RefusedByCP)

	answers := make([]tms.TenantUsage, 0, len(ack.Tenants))
	for _, t := range ack.Tenants {
		result.TenantsHeard++
		m, placed := mappings[t.InstanceId]
		if !placed {
			continue
		}
		state := m.state(
			instanceapi.UsageAck{Window: t.Window, Overage: t.Overage, Meters: t.Meters},
		)
		meters := make([]tms.TenantMeter, 0, len(state.Meters))
		for _, mu := range state.Meters {
			meters = append(meters, tms.TenantMeter(mu))
		}
		answers = append(answers, tms.TenantUsage{
			InstanceID: t.InstanceId.String(),
			Window:     state.Window,
			Overage:    state.Overage,
			Meters:     meters,
		})
	}
	result.TenantsMapped = len(answers)
	if len(answers) == 0 {
		return result, nil
	}
	answer, err := r.tms.PutTenantUsage(ctx, answers)
	if err != nil {
		return result, fmt.Errorf("write the tenants' usage answers to the TMS: %w", err)
	}
	result.Answered = answer.Applied
	for _, e := range answer.Refused {
		result.RefusedByTMS = append(result.RefusedByTMS, e.InstanceID)
	}
	return result, nil
}

func asReadings(in []tms.TenantReading) []tms.Reading {
	out := make([]tms.Reading, 0, len(in))
	for _, rd := range in {
		out = append(out, tms.Reading(rd))
	}
	return out
}
