package usage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
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

type tenantHour struct {
	instance uuid.UUID
	hour     int64
}

type SharedRelay struct {
	cp     SharedControlPlane
	tms    SharedTMS
	logger *slog.Logger
	now    func() time.Time

	mu      sync.Mutex
	refused map[tenantHour]instanceapi.SharedEntryRefusedCode
	logged  map[uuid.UUID]int64
}

type SharedParams struct {
	ControlPlane SharedControlPlane
	TMS          SharedTMS
	Logger       *slog.Logger
	Now          func() time.Time
}

func NewShared(p SharedParams) *SharedRelay {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &SharedRelay{
		cp:      p.ControlPlane,
		tms:     p.TMS,
		logger:  p.Logger,
		now:     now,
		refused: make(map[tenantHour]instanceapi.SharedEntryRefusedCode),
		logged:  make(map[uuid.UUID]int64),
	}
}

type SharedResult struct {
	Readings      int
	Dropped       int
	Withheld      int
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
	r.mu.Lock()
	defer r.mu.Unlock()
	mappings := make(map[uuid.UUID]mapping, len(tenants))
	for i := range tenants {
		mappings[tenants[i].InstanceId] = newMapping(tenants[i].Allowances)
	}
	now := r.now().UTC()
	to := now.Truncate(time.Hour)
	from := to.Add(-Lookback)
	r.forgetBefore(from)
	held, err := r.tms.TenantReadings(ctx, from, to)
	if err != nil {
		return SharedResult{}, err
	}

	var result SharedResult
	readings := r.withhold(r.mapReadings(held, mappings, &result), &result)
	result.Readings = len(readings)

	var ack instanceapi.SharedUsageAck
	refused := make(map[uuid.UUID]instanceapi.SharedEntryRefusedCode)
	for _, chunk := range chunks(readings, MaxSharedReadings) {
		ack, err = r.cp.ReportSharedUsage(ctx, instanceapi.SharedUsageReport{Readings: chunk})
		if err != nil {
			return result, err
		}
		r.markRefused(chunk, ack.Refused, refused)
	}
	for id, code := range refused {
		result.RefusedByCP = append(result.RefusedByCP, id.String())
		r.logRefusal(id, code, now)
	}
	slices.Sort(result.RefusedByCP)

	answers := answersFor(ack.Tenants, mappings, &result)
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

func (r *SharedRelay) forgetBefore(from time.Time) {
	cutoff := from.Unix()
	for key := range r.refused {
		if key.hour < cutoff {
			delete(r.refused, key)
		}
	}
	for id, hour := range r.logged {
		if hour < cutoff {
			delete(r.logged, id)
		}
	}
}

func (r *SharedRelay) withhold(
	readings []instanceapi.SharedUsageReading,
	result *SharedResult,
) []instanceapi.SharedUsageReading {
	if len(r.refused) == 0 {
		return readings
	}
	kept := readings[:0]
	for _, rd := range readings {
		if _, done := r.refused[tenantHour{instance: rd.InstanceId, hour: rd.Hour.Unix()}]; done {
			result.Withheld++
			continue
		}
		kept = append(kept, rd)
	}
	return kept
}

func (r *SharedRelay) markRefused(
	chunk []instanceapi.SharedUsageReading,
	entries []instanceapi.SharedEntryRefused,
	refused map[uuid.UUID]instanceapi.SharedEntryRefusedCode,
) {
	if len(entries) == 0 {
		return
	}
	codes := make(map[uuid.UUID]instanceapi.SharedEntryRefusedCode, len(entries))
	for _, e := range entries {
		codes[e.InstanceId] = e.Code
		refused[e.InstanceId] = e.Code
	}
	for _, rd := range chunk {
		if code, ok := codes[rd.InstanceId]; ok {
			r.refused[tenantHour{instance: rd.InstanceId, hour: rd.Hour.Unix()}] = code
		}
	}
}

func (r *SharedRelay) logRefusal(
	id uuid.UUID,
	code instanceapi.SharedEntryRefusedCode,
	now time.Time,
) {
	hour := now.Truncate(time.Hour).Unix()
	if last, ok := r.logged[id]; ok && last == hour {
		return
	}
	r.logged[id] = hour
	r.logger.Warn(
		"the control plane refused a tenant's usage readings; those hours are not reported again",
		"instanceId", id.String(),
		"code", string(code),
	)
}

func (r *SharedRelay) mapReadings(
	held []tms.TenantReadings,
	mappings map[uuid.UUID]mapping,
	result *SharedResult,
) []instanceapi.SharedUsageReading {
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
	return readings
}

func answersFor(
	tenants []instanceapi.SharedTenantUsage,
	mappings map[uuid.UUID]mapping,
	result *SharedResult,
) []tms.TenantUsage {
	answers := make([]tms.TenantUsage, 0, len(tenants))
	for _, t := range tenants {
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
	return answers
}

func asReadings(in []tms.TenantReading) []tms.Reading {
	out := make([]tms.Reading, 0, len(in))
	for _, rd := range in {
		out = append(out, tms.Reading(rd))
	}
	return out
}
