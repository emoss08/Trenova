package hosttelemetry_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/hosttelemetry"
	"github.com/emoss08/trenova/operator/internal/telemetryrelay"
	"github.com/emoss08/trenova/operator/internal/tms"
)

type machine struct {
	diskErr error
	path    string
}

func (m *machine) CPUPercent(context.Context) (float64, error) { return 37.5, nil }

func (m *machine) CPUCount(context.Context) (int, error) { return 8, nil }

func (m *machine) Memory(context.Context) (uint64, uint64, error) {
	return 6 << 30, 24 << 30, nil
}

func (m *machine) Disk(_ context.Context, path string) (uint64, uint64, error) {
	m.path = path
	return 120 << 30, 400 << 30, m.diskErr
}

type data struct{}

func (data) DataUsage(context.Context) (tms.DataUsage, error) {
	db := int64(5 << 30)
	return tms.DataUsage{DatabaseBytes: &db}, nil
}

func TestASampleReadsTheHostAndTheTMSsData(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := &machine{}
	src := hosttelemetry.New(&hosttelemetry.Params{
		Machine:  m,
		DataPath: "/srv/trenova",
		Data:     telemetryrelay.NewDataReader(data{}, logger),
		Logger:   logger,
	})

	at := time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC)
	sample := src.Sample(t.Context(), at)

	assert.Equal(t, at, sample.At)
	assert.NotNil(t, sample.Parts)
	require.NotNil(t, sample.Machine)
	assert.Equal(t, int64(8000), *sample.Machine.CpuCapacityMillicores)
	assert.Equal(t, int64(3000), *sample.Machine.CpuMillicores)
	assert.Equal(t, int64(24<<30), *sample.Machine.MemoryCapacityBytes)
	assert.Equal(t, int64(6<<30), *sample.Machine.MemoryBytes)
	assert.Equal(t, int64(400<<30), *sample.Machine.DiskCapacityBytes)
	assert.Equal(t, "/srv/trenova", m.path)
	require.NotNil(t, sample.Data)
	assert.Equal(t, int64(5<<30), *sample.Data.DatabaseBytes)
}

func TestAFigureThatCannotBeReadIsLeftOut(t *testing.T) {
	t.Parallel()
	src := hosttelemetry.New(&hosttelemetry.Params{
		Machine:  &machine{diskErr: errors.New("no such volume")},
		DataPath: "/missing",
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	sample := src.Sample(t.Context(), time.Now().UTC().Truncate(time.Minute))
	require.NotNil(t, sample.Machine)
	assert.Nil(t, sample.Machine.DiskBytes)
	assert.Nil(t, sample.Machine.DiskCapacityBytes)
	assert.Nil(t, sample.Data)
}
