package hosttelemetry

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/telemetryrelay"
)

const readTimeout = 5 * time.Second

type Machine interface {
	CPUPercent(ctx context.Context) (float64, error)
	CPUCount(ctx context.Context) (int, error)
	Memory(ctx context.Context) (used, total uint64, err error)
	Disk(ctx context.Context, path string) (used, total uint64, err error)
}

type Source struct {
	machine  Machine
	dataPath string
	data     *telemetryrelay.DataReader
	logger   *slog.Logger
}

type Params struct {
	Machine  Machine
	DataPath string
	Data     *telemetryrelay.DataReader
	Logger   *slog.Logger
}

func New(p *Params) *Source {
	machine := p.Machine
	if machine == nil {
		machine = Host{}
	}
	return &Source{machine: machine, dataPath: p.DataPath, data: p.Data, logger: p.Logger}
}

var _ telemetryrelay.Source = (*Source)(nil)

func (s *Source) Sample(ctx context.Context, at time.Time) instanceapi.TelemetrySample {
	sample := instanceapi.TelemetrySample{At: at, Parts: []instanceapi.PartUsage{}}
	run, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	machine := s.sampleMachine(run)
	if machine != (instanceapi.MachineUsage{}) {
		sample.Machine = &machine
	}
	if s.data != nil {
		if data := s.data.Read(ctx, at); data != (instanceapi.DataUsage{}) {
			sample.Data = &data
		}
	}
	return sample
}

func (s *Source) sampleMachine(ctx context.Context) instanceapi.MachineUsage {
	var m instanceapi.MachineUsage
	if cores, err := s.machine.CPUCount(ctx); err == nil && cores > 0 {
		capacity := int64(cores) * 1000
		m.CpuCapacityMillicores = &capacity
		if percent, perr := s.machine.CPUPercent(ctx); perr == nil {
			used := int64(math.Round(percent / 100 * float64(capacity)))
			used = min(max(used, 0), capacity)
			m.CpuMillicores = &used
		} else {
			s.logger.Warn("telemetry: cannot read the processor's use", "error", perr)
		}
	} else if err != nil {
		s.logger.Warn("telemetry: cannot count the processors", "error", err)
	}
	if used, total, err := s.machine.Memory(ctx); err == nil && total > 0 {
		m.MemoryBytes, m.MemoryCapacityBytes = count(used), count(total)
	} else if err != nil {
		s.logger.Warn("telemetry: cannot read the memory", "error", err)
	}
	if s.dataPath != "" {
		if used, total, err := s.machine.Disk(ctx, s.dataPath); err == nil && total > 0 {
			m.DiskBytes, m.DiskCapacityBytes = count(used), count(total)
		} else if err != nil {
			s.logger.Warn("telemetry: cannot read the data disk", "path", s.dataPath, "error", err)
		}
	}
	return m
}

func count(n uint64) *int64 {
	v := int64(min(n, math.MaxInt64))
	return &v
}

type Host struct{}

func (Host) CPUPercent(ctx context.Context) (float64, error) {
	percents, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil || len(percents) == 0 {
		return 0, err
	}
	return percents[0], nil
}

func (Host) CPUCount(ctx context.Context) (int, error) {
	return cpu.CountsWithContext(ctx, true)
}

func (Host) Memory(ctx context.Context) (uint64, uint64, error) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, 0, err
	}
	return v.Total - v.Available, v.Total, nil
}

func (Host) Disk(ctx context.Context, path string) (uint64, uint64, error) {
	u, err := disk.UsageWithContext(ctx, path)
	if err != nil {
		return 0, 0, err
	}
	return u.Used, u.Total, nil
}
